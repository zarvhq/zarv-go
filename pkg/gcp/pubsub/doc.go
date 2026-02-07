// Package pubsub provides a client for Google Cloud Pub/Sub operations.
//
// This package offers a simple interface for publishing and subscribing to messages
// using Google Cloud Pub/Sub, supporting topics, subscriptions, and custom message handlers.
//
// Features:
//   - Thread-safe publisher operations
//   - Context-aware graceful shutdown
//   - Automatic JSON marshalling
//   - Custom message attributes
//   - Concurrent message processing
//   - Panic recovery in handlers
//
// Example Publisher:
//
//	import (
//		"context"
//		"github.com/zarvhq/zarv-go/pkg/gcp/pubsub"
//	)
//
//	func main() {
//		ctx := context.Background()
//		cfg := &pubsub.Cfg{
//			ProjectID: "my-project",
//		}
//
//		client, err := pubsub.NewClient(ctx, cfg)
//		if err != nil {
//			panic(err)
//		}
//		defer client.Close()
//
//		// Create topic
//		err = client.CreateTopic("orders")
//		if err != nil {
//			panic(err)
//		}
//
//		// Create publisher
//		publisher, err := client.NewPublisher("orders")
//		if err != nil {
//			panic(err)
//		}
//		defer publisher.Stop()
//
//		// Publish a message
//		messageID, err := publisher.Publish(ctx, map[string]string{
//			"order_id": "12345",
//			"status": "pending",
//		})
//		if err != nil {
//			panic(err)
//		}
//	}
//
// Example Subscriber:
//
//	type OrderHandler struct{}
//
//	func (h *OrderHandler) HandleMessage(data []byte, attributes map[string]string) error {
//		var order map[string]string
//		if err := json.Unmarshal(data, &order); err != nil {
//			return err
//		}
//		// Process order
//		return nil
//	}
//
//	func main() {
//		ctx, cancel := context.WithCancel(context.Background())
//		defer cancel()
//
//		cfg := &pubsub.Cfg{
//			ProjectID: "my-project",
//		}
//
//		client, err := pubsub.NewClient(ctx, cfg)
//		if err != nil {
//			panic(err)
//		}
//		defer client.Close()
//
//		// Create subscription
//		err = client.CreateSubscription("orders", "orders-sub")
//		if err != nil {
//			panic(err)
//		}
//
//		// Create subscriber
//		handler := &OrderHandler{}
//		subscriber, err := client.NewSubscriber("orders-sub", handler)
//		if err != nil {
//			panic(err)
//		}
//
//		// Start receiving with 10 concurrent workers
//		err = subscriber.Receive(10)
//		if err != nil {
//			panic(err)
//		}
//	}
//
// Graceful Shutdown:
//
// The subscriber supports graceful shutdown via context cancellation.
// When the context is cancelled, the subscriber will stop accepting new messages,
// wait for in-flight messages to complete, and return nil.
//
//	ctx, cancel := context.WithCancel(context.Background())
//	sigChan := make(chan os.Signal, 1)
//	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
//
//	client, _ := pubsub.NewClient(ctx, cfg)
//	subscriber, _ := client.NewSubscriber("sub-id", handler)
//
//	go func() {
//		subscriber.Receive(10) // Will stop gracefully on context cancellation
//	}()
//
//	<-sigChan
//	cancel() // Triggers graceful shutdown
package pubsub
