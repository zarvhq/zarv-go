package pubsub

import (
	"context"
	"fmt"

	"cloud.google.com/go/pubsub"
	"google.golang.org/api/option"
)

// Client represents a Google Cloud Pub/Sub client that manages connections and creates publishers/subscribers.
type Client interface {
	// NewPublisher creates a new publisher for the specified topic.
	NewPublisher(topicID string) (Publisher, error)
	// NewSubscriber creates a new subscriber for the specified subscription.
	NewSubscriber(subscriptionID string, handler SubscriberHandler) (Subscriber, error)
	// CreateTopic creates a new topic if it doesn't exist.
	CreateTopic(topicID string) error
	// CreateSubscription creates a new subscription for a topic if it doesn't exist.
	CreateSubscription(topicID, subscriptionID string) error
	// Close closes the Pub/Sub client.
	Close() error
}

type client struct {
	pubsubClient *pubsub.Client
	projectID    string
	context      context.Context
}

// Cfg holds the configuration for creating a Pub/Sub client.
type Cfg struct {
	ProjectID       string
	CredentialsJSON []byte // Optional: if not provided, uses Application Default Credentials (Workload Identity)
}

// NewClient creates a new Google Cloud Pub/Sub client with the given context and configuration.
func NewClient(ctx context.Context, cfg *Cfg) (Client, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}

	if cfg == nil || cfg.ProjectID == "" {
		return nil, fmt.Errorf("project ID cannot be empty")
	}

	var opts []option.ClientOption
	if len(cfg.CredentialsJSON) > 0 {
		opts = append(opts, option.WithCredentialsJSON(cfg.CredentialsJSON))
	}

	pubsubClient, err := pubsub.NewClient(ctx, cfg.ProjectID, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create pub/sub client: %w", err)
	}

	return &client{
		pubsubClient: pubsubClient,
		projectID:    cfg.ProjectID,
		context:      ctx,
	}, nil
}

// CreateTopic creates a new topic if it doesn't exist.
func (c *client) CreateTopic(topicID string) error {
	if topicID == "" {
		return fmt.Errorf("topic ID cannot be empty")
	}

	topic := c.pubsubClient.Topic(topicID)
	exists, err := topic.Exists(c.context)
	if err != nil {
		return fmt.Errorf("failed to check if topic exists: %w", err)
	}

	if !exists {
		_, err = c.pubsubClient.CreateTopic(c.context, topicID)
		if err != nil {
			return fmt.Errorf("failed to create topic: %w", err)
		}
	}

	return nil
}

// CreateSubscription creates a new subscription for a topic if it doesn't exist.
func (c *client) CreateSubscription(topicID, subscriptionID string) error {
	if topicID == "" {
		return fmt.Errorf("topic ID cannot be empty")
	}
	if subscriptionID == "" {
		return fmt.Errorf("subscription ID cannot be empty")
	}

	sub := c.pubsubClient.Subscription(subscriptionID)
	exists, err := sub.Exists(c.context)
	if err != nil {
		return fmt.Errorf("failed to check if subscription exists: %w", err)
	}

	if !exists {
		topic := c.pubsubClient.Topic(topicID)
		_, err = c.pubsubClient.CreateSubscription(c.context, subscriptionID, pubsub.SubscriptionConfig{
			Topic:       topic,
			AckDeadline: 60, // 60 seconds
		})
		if err != nil {
			return fmt.Errorf("failed to create subscription: %w", err)
		}
	}

	return nil
}

// Close closes the Pub/Sub client gracefully.
func (c *client) Close() error {
	if c.pubsubClient == nil {
		return nil
	}
	return c.pubsubClient.Close()
}
