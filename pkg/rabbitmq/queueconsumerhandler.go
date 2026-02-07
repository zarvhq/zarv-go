package rabbitmq

type ConsumerHandler interface {
	HandleMessage([]byte) error
}
