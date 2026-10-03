package pubsub

import (
	"context"
	"fmt"

	//nolint:staticcheck // v1 client kept for compatibility; upgrade to v2 pending.
	"cloud.google.com/go/pubsub"
	"google.golang.org/api/impersonate"
	"google.golang.org/api/option"
)

// Client represents a Google Cloud Pub/Sub client that manages connections and creates publishers/subscribers.
type Client interface {
	// NewPublisher creates a new publisher for the specified topic.
	NewPublisher(topicID string, opts ...PublisherOption) (Publisher, error)
	// NewSubscriber creates a new subscriber for the specified subscription.
	NewSubscriber(subscriptionID string, handler SubscriberHandler, opts ...SubscriberOption) (Subscriber, error)
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
	// ImpersonateServiceAccount, when set (a service-account email), makes the
	// client act as that account by minting short-lived tokens for it from the
	// ambient credentials (Workload Identity). The ambient identity needs
	// roles/iam.serviceAccountTokenCreator on the target. This is the keyless way
	// to run under a dedicated identity (e.g. one in the project that owns the
	// topic) when the pod's own SA belongs elsewhere — the API's consumer project
	// becomes the target account's project.
	ImpersonateServiceAccount string
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
		//nolint:staticcheck // WithCredentialsJSON deprecated; kept to support legacy secret format.
		opts = append(opts, option.WithCredentialsJSON(cfg.CredentialsJSON))
	}
	if cfg.ImpersonateServiceAccount != "" {
		ts, err := impersonate.CredentialsTokenSource(ctx, impersonate.CredentialsConfig{
			TargetPrincipal: cfg.ImpersonateServiceAccount,
			Scopes:          []string{"https://www.googleapis.com/auth/pubsub"},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create impersonated token source for %q: %w", cfg.ImpersonateServiceAccount, err)
		}
		// Pin the quota/billing project to ProjectID. Without this, a client on
		// GCE/GKE bills the API call to the node's project (not the impersonated
		// account's), which may differ and have the API disabled. The impersonated
		// account needs roles/serviceusage.serviceUsageConsumer on ProjectID.
		opts = append(opts, option.WithTokenSource(ts), option.WithQuotaProject(cfg.ProjectID))
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
