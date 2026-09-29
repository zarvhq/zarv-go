package pubsub

import (
	"testing"

	//nolint:staticcheck // v1 client kept for compatibility with the package under test.
	"cloud.google.com/go/pubsub"
)

// noopHandler satisfies SubscriberHandler for construction-only tests.
type noopHandler struct{}

func (noopHandler) HandleMessage(_ []byte, _ map[string]string) error { return nil }

func mustSubscription(t *testing.T, c *client, topicID, subID string) {
	t.Helper()
	mustTopic(t, c, topicID)
	if _, err := c.pubsubClient.CreateSubscription(c.context, subID, pubsub.SubscriptionConfig{
		Topic: c.pubsubClient.Topic(topicID),
	}); err != nil {
		t.Fatalf("create subscription: %v", err)
	}
}

func TestNewSubscriber_MissingSubscriptionErrorsByDefault(t *testing.T) {
	c, _ := newTestClient(t)

	if _, err := c.NewSubscriber("does-not-exist", noopHandler{}); err == nil {
		t.Fatal("expected an error for a missing subscription, got nil")
	}
}

func TestNewSubscriber_WithoutSubscriptionExistsCheckSkipsLookup(t *testing.T) {
	c, _ := newTestClient(t)
	mustSubscription(t, c, "events", "events-sub")

	// The existence check needs pubsub.subscriptions.get; a receive-only SA
	// should be able to build a subscriber without it.
	if _, err := c.NewSubscriber("events-sub", noopHandler{}, WithoutSubscriptionExistsCheck()); err != nil {
		t.Fatalf("expected no error when skipping the exists check, got %v", err)
	}
}
