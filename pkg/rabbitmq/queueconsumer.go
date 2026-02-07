package rabbitmq

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/rabbitmq/amqp091-go"
)

// Consumer consumes messages from a RabbitMQ queue.
type Consumer interface {
	Consume(concurrency int) error
}

type consumer struct {
	name      string
	queueName string
	conn      *amqp091.Connection
	handler   ConsumerHandler
	context   context.Context
}

// NewConsumer creates a new queue consumer bound to the provided queue and handler.
func (k *client) NewConsumer(consumerName, queueName string, handler ConsumerHandler) (Consumer, error) {
	return &consumer{
		name:      consumerName,
		queueName: queueName,
		conn:      k.conn,
		handler:   handler,
		context:   k.context,
	}, nil
}

// Consume starts consuming messages with a given concurrency level.
func (c *consumer) Consume(concurrency int) error {
	ch, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("error opening channel: %w", err)
	}

	closeChan := make(chan *amqp091.Error, 1)
	ch.NotifyClose(closeChan)

	defer func() {
		if err := ch.Close(); err != nil {
			slog.Error("error closing channel", slog.String("error", err.Error()), slog.String("handler", c.name))
		}
	}()

	// Declare queue as durable for production reliability
	q, err := ch.QueueDeclare(
		c.queueName, // name
		true,        // durable - survive broker restart
		false,       // auto-delete
		false,       // exclusive
		false,       // no-wait
		nil,         // arguments
	)
	if err != nil {
		return fmt.Errorf("error declaring queue: %w", err)
	}

	// Set QoS to limit unacknowledged messages per consumer
	if err := ch.Qos(concurrency, 0, false); err != nil {
		return fmt.Errorf("error setting QoS: %w", err)
	}

	msgs, err := ch.Consume(
		q.Name, // queue name
		c.name, // consumer name
		false,  // auto-ack (acknowledgement is handled by the handler)
		false,  // exclusive (only this consumer can access the queue)
		false,  // no-local (can't consume messages from this connection)
		false,  // no-wait (don't wait for the server to confirm the request)
		nil,    // args (optional arguments)
	)

	if err != nil {
		return fmt.Errorf("error consuming messages: %w", err)
	}

	slog.Info("consumer started", slog.String("handler", c.name), slog.Int("concurrency", concurrency))

	wg := sync.WaitGroup{}
	semaphore := make(chan struct{}, concurrency)
	done := make(chan struct{})
	var shutdownErr error

	// Cleanup goroutine
	go func() {
		select {
		case err := <-closeChan:
			if err != nil {
				slog.Error("channel closed with error", slog.String("error", err.Error()), slog.String("handler", c.name))
				shutdownErr = fmt.Errorf("channel closed: %w", err)
			} else {
				slog.Info("channel closed gracefully", slog.String("handler", c.name))
			}
			close(done)
		case <-c.context.Done():
			slog.Info("context canceled", slog.String("handler", c.name))
			close(done)
		}
	}()

	// Message processing loop
	for {
		select {
		case <-done:
			slog.Info("stopping consumer, waiting for in-flight messages", slog.String("handler", c.name))
			wg.Wait()
			slog.Info("consumer stopped", slog.String("handler", c.name))
			return shutdownErr

		case msg, ok := <-msgs:
			if !ok {
				// Channel closed
				slog.Info("messages channel closed", slog.String("handler", c.name))
				wg.Wait()
				return nil
			}

			if len(msg.Body) == 0 {
				if err := msg.Ack(false); err != nil {
					slog.Error("failed to ack empty message", slog.String("error", err.Error()), slog.String("handler", c.name))
				}
				continue
			}

			go c.HandleMessage(msg, &wg, semaphore)
		}
	}
}

// HandleMessage wraps handler invocation with ack/nack and panic recovery.
func (c *consumer) HandleMessage(msg amqp091.Delivery, wg *sync.WaitGroup, semaphore chan struct{}) {
	semaphore <- struct{}{}
	wg.Add(1)

	defer wg.Done()
	defer func() { <-semaphore }()
	defer func() {
		if r := recover(); r != nil {
			slog.Error("panic recovered in message handler",
				slog.Any("panic", r),
				slog.String("handler", c.name))
			if err := msg.Nack(false, true); err != nil {
				slog.Error("failed to nack message after panic", slog.String("error", err.Error()), slog.String("handler", c.name))
			}
		}
	}()

	if err := c.handler.HandleMessage(msg.Body); err != nil {
		slog.Error("error handling message",
			slog.String("error", err.Error()),
			slog.String("handler", c.name))
		if err := msg.Nack(false, true); err != nil {
			slog.Error("failed to nack message", slog.String("error", err.Error()), slog.String("handler", c.name))
		}
		return
	}

	slog.Debug("message handled successfully", slog.String("handler", c.name))
	if err := msg.Ack(false); err != nil {
		slog.Error("failed to ack message", slog.String("error", err.Error()), slog.String("handler", c.name))
	}
}
