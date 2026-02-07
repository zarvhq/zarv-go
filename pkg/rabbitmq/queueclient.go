package rabbitmq

import (
	"context"
	"fmt"

	"github.com/rabbitmq/amqp091-go"
)

// Client represents a RabbitMQ client that manages connections and creates consumers/producers.
type Client interface {
	// NewConsumer creates a new consumer for the specified queue.
	NewConsumer(consumerName, queueName string, handler ConsumerHandler) (Consumer, error)
	// NewProducer creates a new producer for publishing messages.
	NewProducer() (Producer, error)
	// Close closes the RabbitMQ connection.
	Close() error
	// IsClosed returns true if the connection is closed.
	IsClosed() bool
}

type client struct {
	conn    *amqp091.Connection
	context context.Context
}

// NewClient creates a new RabbitMQ client with the given context and connection URL.
// The URL format is: amqp://username:password@host:port/vhost
// Example: amqp://guest:guest@localhost:5672/
func NewClient(ctx context.Context, url string) (Client, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}

	if url == "" {
		return nil, fmt.Errorf("connection URL cannot be empty")
	}

	conn, err := amqp091.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	if conn.IsClosed() {
		return nil, fmt.Errorf("connection is closed")
	}

	mqClient := &client{conn: conn, context: ctx}

	return mqClient, nil
}

// IsClosed returns true if the RabbitMQ connection is closed.
func (c *client) IsClosed() bool {
	return c.conn.IsClosed()
}

// Close closes the RabbitMQ connection gracefully.
func (c *client) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}
