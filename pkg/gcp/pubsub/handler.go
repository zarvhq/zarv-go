package pubsub

// SubscriberHandler defines the interface for handling Pub/Sub messages.
// Implementations must be thread-safe as HandleMessage may be called concurrently.
type SubscriberHandler interface {
	// HandleMessage processes a received message.
	// Returns nil to acknowledge the message, or an error to nack it.
	// The message will be redelivered if an error is returned.
	HandleMessage(data []byte, attributes map[string]string) error
}
