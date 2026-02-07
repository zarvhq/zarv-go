// Package rabbitmq provides a client for RabbitMQ message queue operations.
//
// This package offers a simple interface for publishing and consuming messages
// from RabbitMQ queues, supporting concurrent consumers and custom message handlers.
//
// Features:
//   - Automatic channel reconnection for producers
//   - Concurrent message processing for consumers
//   - Persistent messages (survive broker restarts)
//   - Durable queues
//   - Thread-safe producer operations
//   - Context-aware operations
//
// Example Producer:
//
//	import (
//		"context"
//		"github.com/zarvhq/zarv-go/pkg/rabbitmq"
//	)
//
//	func main() {
//		ctx := context.Background()
//		client, err := rabbitmq.NewClient(ctx, "amqp://guest:guest@localhost:5672/")
//		if err != nil {
//			panic(err)
//		}
//		defer client.Close()
//
//		producer, err := client.NewProducer()
//		if err != nil {
//			panic(err)
//		}
//
//		// Publish a message
//		// If channel closes, Publish will automatically reconnect
//		err = producer.Publish("my-queue", map[string]string{
//			"message": "Hello, World!",
//		})
//		if err != nil {
//			panic(err)
//		}
//	}
//
// Automatic Reconnection:
//
// The producer automatically handles channel reconnections. If the channel closes
// due to network issues or broker restarts, the next Publish call will automatically
// recreate the channel. If the underlying connection is closed, you must create a new Client.
//
//	if client.IsClosed() {
//		// Recreate client and producer
//		client, _ = rabbitmq.NewClient(ctx, url)
//		producer, _ = client.NewProducer()
//	}
//
// Graceful Shutdown:
//
// The consumer supports graceful shutdown via context cancellation.
// When the context is canceled, the consumer will:
//
//  1. Stop accepting new messages
//
//  2. Wait for in-flight messages to complete processing
//
//  3. Acknowledge or reject all processed messages
//
//  4. Close the channel gracefully
//
//  5. Return nil after cleanup
//
//     ctx, cancel := context.WithCancel(context.Background())
//     defer cancel()
//
//     sigChan := make(chan os.Signal, 1)
//     signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
//
//     client, _ := rabbitmq.NewClient(ctx, url)
//     consumer, _ := client.NewConsumer("worker", "queue", handler)
//
//     go func() {
//     consumer.Consume(5) // Will stop gracefully on context cancellation
//     }()
//
//     <-sigChan
//     cancel() // Triggers graceful shutdown
package rabbitmq
