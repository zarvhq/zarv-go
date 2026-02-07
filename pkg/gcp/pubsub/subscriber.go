package pubsub

import (
	"context"
	"fmt"
	"log/slog"

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

// NewSubscriber creates a new subscriber for receiving messages from a subscription.
// The subscription must exist before calling this method.
func (c *client) NewSubscriber(subscriptionID string, handler SubscriberHandler) (Subscriber, error) {
	if subscriptionID == "" {
		return nil, fmt.Errorf("subscription ID cannot be empty")
	}

	if handler == nil {
		return nil, fmt.Errorf("handler cannot be nil")
	}

	sub := c.pubsubClient.Subscription(subscriptionID)

	// Check if subscription exists
	exists, err := sub.Exists(c.context)
	if err != nil {
		return nil, fmt.Errorf("failed to check if subscription exists: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("subscription %s does not exist", subscriptionID)
	}

	return &subscriber{
		subscription: sub,
		handler:      handler,
		context:      c.context,
		name:         subscriptionID,
	}, nil
}

// Receive starts receiving messages with the specified concurrency.
// The method blocks until the context is cancelled or an error occurs.
// It returns nil on graceful shutdown (context cancellation) or error on failure.
//
// Graceful Shutdown:
// When the context is cancelled, Receive will:
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

	// Receive blocks until context is cancelled
	err := s.subscription.Receive(s.context, func(ctx context.Context, msg *pubsub.Message) {
		s.handleMessage(ctx, msg)
	})

	if err != nil {
		slog.Error("subscriber error",
			slog.String("error", err.Error()),
			slog.String("subscription", s.name))
		return fmt.Errorf("subscription receive error: %w", err)
	}

	// Context was cancelled - graceful shutdown
	slog.Info("subscriber stopped gracefully", slog.String("subscription", s.name))
	return nil
}

func (s *subscriber) handleMessage(ctx context.Context, msg *pubsub.Message) {
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
