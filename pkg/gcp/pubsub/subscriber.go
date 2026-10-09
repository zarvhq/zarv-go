package pubsub

import (
	"context"
	"fmt"
	"log/slog"

	//nolint:staticcheck // v1 client kept for compatibility; upgrade to v2 pending.
	"cloud.google.com/go/pubsub"
)

// Subscriber receives messages from Google Cloud Pub/Sub subscriptions.
type Subscriber interface {
	// Receive starts receiving messages with the specified concurrency.
	// Returns error if subscription fails or nil on graceful context cancellation.
	Receive(concurrency int) error
}

type subscriber struct {
	subscription *pubsub.Subscription
	handler      SubscriberHandler
	context      context.Context
	name         string
}

// SubscriberOption configures a subscriber at creation time.
type SubscriberOption func(*subscriberConfig)

type subscriberConfig struct {
	skipExistsCheck bool
}

// WithoutSubscriptionExistsCheck skips the subscription.Exists lookup in
// NewSubscriber. That lookup requires the pubsub.subscriptions.get permission; a
// receive-only service account (roles/pubsub.subscriber) does not have it, so
// skip the check when the subscription is known to exist (e.g. provisioned by
// IaC) and the caller should not be granted get. Mirrors WithoutTopicExistsCheck.
func WithoutSubscriptionExistsCheck() SubscriberOption {
	return func(c *subscriberConfig) { c.skipExistsCheck = true }
}

// NewSubscriber creates a new subscriber for receiving messages from a subscription.
// By default it validates that the subscription exists; pass
// WithoutSubscriptionExistsCheck to skip that lookup (see the option's docs).
func (c *client) NewSubscriber(subscriptionID string, handler SubscriberHandler, opts ...SubscriberOption) (Subscriber, error) {
	if subscriptionID == "" {
		return nil, fmt.Errorf("subscription ID cannot be empty")
	}

	if handler == nil {
		return nil, fmt.Errorf("handler cannot be nil")
	}

	cfg := subscriberConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	sub := c.pubsubClient.Subscription(subscriptionID)

	if !cfg.skipExistsCheck {
		exists, err := sub.Exists(c.context)
		if err != nil {
			return nil, fmt.Errorf("failed to check if subscription exists: %w", err)
		}
		if !exists {
			return nil, fmt.Errorf("subscription %s does not exist", subscriptionID)
		}
	}

	return &subscriber{
		subscription: sub,
		handler:      handler,
		context:      c.context,
		name:         subscriptionID,
	}, nil
}

// Receive starts receiving messages with the specified concurrency.
// The method blocks until the context is canceled or an error occurs.
// It returns nil on graceful shutdown (context cancellation) or error on failure.
//
// Graceful Shutdown:
// When the context is canceled, Receive will:
//  1. Stop accepting new messages
//  2. Wait for in-flight messages to complete processing
//  3. Return nil after cleanup
func (s *subscriber) Receive(concurrency int) error {
	if concurrency <= 0 {
		return fmt.Errorf("concurrency must be greater than 0")
	}

	// Configure subscription settings
	s.subscription.ReceiveSettings.MaxOutstandingMessages = concurrency
	s.subscription.ReceiveSettings.NumGoroutines = concurrency

	slog.Info("subscriber started",
		slog.String("subscription", s.name),
		slog.Int("concurrency", concurrency))

	// Receive blocks until context is canceled
	err := s.subscription.Receive(s.context, func(ctx context.Context, msg *pubsub.Message) {
		s.handleMessage(ctx, msg)
	})

	if err != nil {
		slog.Error("subscriber error",
			slog.String("error", err.Error()),
			slog.String("subscription", s.name))
		return fmt.Errorf("subscription receive error: %w", err)
	}

	// Context was canceled - graceful shutdown
	slog.Info("subscriber stopped gracefully", slog.String("subscription", s.name))
	return nil
}

func (s *subscriber) handleMessage(_ context.Context, msg *pubsub.Message) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("panic recovered in message handler",
				slog.Any("panic", r),
				slog.String("subscription", s.name),
				slog.String("messageID", msg.ID))
			msg.Nack()
		}
	}()

	if err := s.handler.HandleMessage(msg.Data, msg.Attributes); err != nil {
		slog.Error("error handling message",
			slog.String("error", err.Error()),
			slog.String("subscription", s.name),
			slog.String("messageID", msg.ID))
		msg.Nack()
		return
	}

	slog.Debug("message handled successfully",
		slog.String("subscription", s.name),
		slog.String("messageID", msg.ID))
	msg.Ack()
}
