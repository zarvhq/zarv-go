package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/rabbitmq/amqp091-go"
)

// Producer publishes messages to RabbitMQ queues.
// The producer automatically handles channel reconnection if the channel closes
// due to network issues or broker restarts. If the underlying connection is closed,
// a new Client must be created.
type Producer interface {
	// Publish sends a message to the specified queue.
	// The body will be automatically marshalled to JSON.
	// Messages are published as persistent (survive broker restarts).
	// If the channel is closed, Publish will attempt to reconnect automatically.
	Publish(queueName string, body any) error
	// Close closes the producer's channel.
	Close() error
}

type producer struct {
	conn    *amqp091.Connection
	ch      *amqp091.Channel
	mu      sync.Mutex
	context context.Context
}

// NewProducer creates a new producer for publishing messages.
// The producer maintains a persistent channel that is automatically monitored.
// If the channel closes, it will be automatically recreated on the next Publish call.
// Remember to call Close() when done to release resources.
func (c *client) NewProducer() (Producer, error) {
	ch, err := c.conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	p := &producer{
		conn:    c.conn,
		ch:      ch,
		context: c.context,
	}

	// Monitor channel closures
	go p.monitorChannel()

	return p, nil
}

// Publish sends a message to the specified queue.
// The message body is automatically marshalled to JSON and published as persistent.
//
// Automatic Reconnection:
// If the channel is closed (due to network issues or broker restart), Publish will
// automatically attempt to recreate the channel. If the underlying connection is closed,
// the method will return an error and you must create a new Client.
//
// Thread-safe: Multiple goroutines can safely call Publish concurrently.
func (p *producer) Publish(queueName string, body any) error {
	if queueName == "" {
		return fmt.Errorf("queue name cannot be empty")
	}

	if body == nil {
		return fmt.Errorf("message body cannot be nil")
	}

	bytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal message body: %w", err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Check if channel is closed and try to reconnect
	if p.ch == nil || p.ch.IsClosed() {
		if err := p.reconnect(); err != nil {
			return fmt.Errorf("failed to reconnect channel: %w", err)
		}
		// Channel reconnected successfully, continue with publish
	}

	// Declare queue to ensure it exists
	_, err = p.ch.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // auto-delete
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	err = p.ch.PublishWithContext(
		p.context,
		"",        // exchange (empty for default)
		queueName, // routing key (queue name)
		false,     // mandatory
		false,     // immediate
		amqp091.Publishing{
			ContentType:  "application/json",
			Body:         bytes,
			DeliveryMode: amqp091.Persistent, // 2 = persistent
		},
	)

	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

// Close closes the producer's channel gracefully.
// This should be called when done publishing messages.
func (p *producer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.ch == nil {
		return nil
	}
	return p.ch.Close()
}

// reconnect attempts to recreate the channel when it's closed.
// Must be called with p.mu locked.
func (p *producer) reconnect() error {
	if p.conn == nil || p.conn.IsClosed() {
		return fmt.Errorf("connection is closed, cannot reconnect channel")
	}

	ch, err := p.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to create new channel: %w", err)
	}

	p.ch = ch
	go p.monitorChannel()
	return nil
}

// monitorChannel watches for channel closures and logs them.
func (p *producer) monitorChannel() {
	if p.ch == nil {
		return
	}

	closeChan := make(chan *amqp091.Error, 1)
	p.ch.NotifyClose(closeChan)

	select {
	case err := <-closeChan:
		if err != nil {
			// Channel closed with error - will try to reconnect on next Publish
			_ = err // Log if needed
		}
	case <-p.context.Done():
		// Context cancelled
		return
	}
}
