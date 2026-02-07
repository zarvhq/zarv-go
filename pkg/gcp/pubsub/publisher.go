package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	//nolint:staticcheck // v1 client kept for compatibility; upgrade to v2 pending.
	"cloud.google.com/go/pubsub"
)

// Publisher publishes messages to Google Cloud Pub/Sub topics.
// The publisher automatically handles topic validation and provides thread-safe operations.
type Publisher interface {
	// Publish sends a message to the topic.
	// The body will be automatically marshaled to JSON.
	// Returns the message ID on success.
	Publish(ctx context.Context, body any) (string, error)
	// PublishWithAttributes sends a message with custom attributes to the topic.
	PublishWithAttributes(ctx context.Context, body any, attributes map[string]string) (string, error)
	// Stop waits for all published messages to be acknowledged and stops the publisher.
	Stop()
}

type publisher struct {
	topic   *pubsub.Topic
	mu      sync.Mutex
	stopped bool
}

// NewPublisher creates a new publisher for publishing messages to a topic.
// The publisher validates that the topic exists before creating.
func (c *client) NewPublisher(topicID string) (Publisher, error) {
	if topicID == "" {
		return nil, fmt.Errorf("topic ID cannot be empty")
	}

	topic := c.pubsubClient.Topic(topicID)

	// Check if topic exists
	exists, err := topic.Exists(c.context)
	if err != nil {
		return nil, fmt.Errorf("failed to check if topic exists: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("topic %s does not exist", topicID)
	}

	return &publisher{
		topic:   topic,
		stopped: false,
	}, nil
}

// Publish sends a message to the topic.
// The message body is automatically marshaled to JSON.
// Thread-safe: Multiple goroutines can safely call Publish concurrently.
func (p *publisher) Publish(ctx context.Context, body any) (string, error) {
	return p.PublishWithAttributes(ctx, body, nil)
}

// PublishWithAttributes sends a message with custom attributes to the topic.
// The message body is automatically marshaled to JSON.
// Thread-safe: Multiple goroutines can safely call Publish concurrently.
func (p *publisher) PublishWithAttributes(ctx context.Context, body any, attributes map[string]string) (string, error) {
	p.mu.Lock()
	if p.stopped {
		p.mu.Unlock()
		return "", fmt.Errorf("publisher has been stopped")
	}
	p.mu.Unlock()

	if body == nil {
		return "", fmt.Errorf("message body cannot be nil")
	}

	bytes, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("failed to marshal message body: %w", err)
	}

	result := p.topic.Publish(ctx, &pubsub.Message{
		Data:       bytes,
		Attributes: attributes,
	})

	// Block until the message is published and get the server-assigned message ID
	messageID, err := result.Get(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to publish message: %w", err)
	}

	return messageID, nil
}

// Stop waits for all published messages to be acknowledged and stops the publisher.
// This should be called when done publishing messages.
func (p *publisher) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.stopped {
		p.topic.Stop()
		p.stopped = true
	}
}
