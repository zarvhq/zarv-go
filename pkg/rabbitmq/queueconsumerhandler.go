package rabbitmq

// ConsumerHandler processes a single message payload.
type ConsumerHandler interface {
	HandleMessage([]byte) error
}
