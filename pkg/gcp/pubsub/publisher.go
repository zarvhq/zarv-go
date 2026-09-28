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
	// PublishAsync enqueues a message and returns immediately, without blocking
	// on the server round-trip. When the publish settles, callback is invoked
	// with the server-assigned message ID or the error. callback may be nil, in
	// which case the outcome is dropped (fire-and-forget). A marshaling or
	// validation error is reported through callback, not returned. Use this on a
	// request path where the publish must not add latency (e.g. audit events);
	// call Stop to flush pending messages before shutdown.
	PublishAsync(ctx context.Context, body any, attributes map[string]string, callback func(id string, err error))
	// Stop waits for all published messages to be acknowledged and stops the publisher.
	Stop()
}

// PublisherOption configures a publisher at creation time.
type PublisherOption func(*publisherConfig)

type publisherConfig struct {
	skipExistsCheck bool
	batchSettings   func(*pubsub.PublishSettings)
}

// WithBatchSettings tunes the client-side publish batching (grouped flush). The
// client already batches by default (DelayThreshold 10ms, CountThreshold 100,
// ByteThreshold 1MB): each Publish enqueues into the current batch, which is
// flushed when any threshold trips, so one long-lived publisher groups many
// messages into few requests. mutate receives a copy of
// pubsub.DefaultPublishSettings, so fields left untouched keep those defaults.
// Raise DelayThreshold/CountThreshold to form larger batches under high volume
// (e.g. audit access events).
func WithBatchSettings(mutate func(*pubsub.PublishSettings)) PublisherOption {
	return func(c *publisherConfig) { c.batchSettings = mutate }
}

// WithoutTopicExistsCheck skips the topic.Exists lookup in NewPublisher. That
// lookup requires the pubsub.topics.get permission; a publish-only service
// account (roles/pubsub.publisher) does not have it, so skip the check when the
// topic is known to exist and the caller should not be granted get.
func WithoutTopicExistsCheck() PublisherOption {
	return func(c *publisherConfig) { c.skipExistsCheck = true }
}

type publisher struct {
	topic   *pubsub.Topic
	mu      sync.Mutex
	stopped bool
}

// NewPublisher creates a new publisher for publishing messages to a topic.
// By default it validates that the topic exists; pass WithoutTopicExistsCheck
// to skip that lookup (see the option's docs for when).
func (c *client) NewPublisher(topicID string, opts ...PublisherOption) (Publisher, error) {
	if topicID == "" {
		return nil, fmt.Errorf("topic ID cannot be empty")
	}

	cfg := publisherConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	topic := c.pubsubClient.Topic(topicID)

	if cfg.batchSettings != nil {
		settings := pubsub.DefaultPublishSettings
		cfg.batchSettings(&settings)
		topic.PublishSettings = settings
	}

	if !cfg.skipExistsCheck {
		exists, err := topic.Exists(c.context)
		if err != nil {
			return nil, fmt.Errorf("failed to check if topic exists: %w", err)
		}
		if !exists {
			return nil, fmt.Errorf("topic %s does not exist", topicID)
		}
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

// PublishAsync enqueues a message without blocking on the server round-trip and
// reports the outcome through callback. Thread-safe.
func (p *publisher) PublishAsync(ctx context.Context, body any, attributes map[string]string, callback func(id string, err error)) {
	report := func(id string, err error) {
		if callback != nil {
			callback(id, err)
		}
	}

	p.mu.Lock()
	stopped := p.stopped
	p.mu.Unlock()
	if stopped {
		report("", fmt.Errorf("publisher has been stopped"))
		return
	}

	if body == nil {
		report("", fmt.Errorf("message body cannot be nil"))
		return
	}

	bytes, err := json.Marshal(body)
	if err != nil {
		report("", fmt.Errorf("failed to marshal message body: %w", err))
		return
	}

	result := p.topic.Publish(ctx, &pubsub.Message{
		Data:       bytes,
		Attributes: attributes,
	})

	// Resolve the publish off the caller's goroutine so the request path is not
	// blocked on the server round-trip.
	go func() {
		messageID, err := result.Get(ctx)
		if err != nil {
			report("", fmt.Errorf("failed to publish message: %w", err))
			return
		}
		report(messageID, nil)
	}()
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
