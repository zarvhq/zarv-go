package pubsub

import (
	"context"
	"sync"
	"testing"
	"time"

	//nolint:staticcheck // v1 client kept for compatibility with the package under test.
	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/pubsub/pstest"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const testProject = "test-project"

// newTestClient spins up an in-memory Pub/Sub fake and wraps it in the
// package's own client so the exported publisher API can be exercised without
// network or credentials.
func newTestClient(t *testing.T) (*client, *pstest.Server) {
	t.Helper()
	ctx := context.Background()

	srv := pstest.NewServer()
	t.Cleanup(func() { _ = srv.Close() })

	conn, err := grpc.NewClient(srv.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial fake: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	psClient, err := pubsub.NewClient(ctx, testProject, option.WithGRPCConn(conn))
	if err != nil {
		t.Fatalf("pubsub client: %v", err)
	}
	t.Cleanup(func() { _ = psClient.Close() })

	return &client{pubsubClient: psClient, projectID: testProject, context: ctx}, srv
}

func mustTopic(t *testing.T, c *client, id string) {
	t.Helper()
	if _, err := c.pubsubClient.CreateTopic(c.context, id); err != nil {
		t.Fatalf("create topic: %v", err)
	}
}

func TestNewPublisher_MissingTopicErrorsByDefault(t *testing.T) {
	c, _ := newTestClient(t)

	if _, err := c.NewPublisher("does-not-exist"); err == nil {
		t.Fatal("expected an error for a missing topic, got nil")
	}
}

func TestNewPublisher_WithoutTopicExistsCheckSkipsLookup(t *testing.T) {
	c, _ := newTestClient(t)
	mustTopic(t, c, "events")

	// The existence check needs pubsub.topics.get; a publish-only SA should be
	// able to build a publisher without it.
	p, err := c.NewPublisher("events", WithoutTopicExistsCheck())
	if err != nil {
		t.Fatalf("expected no error when skipping the exists check, got %v", err)
	}
	defer p.Stop()

	if _, err := p.Publish(context.Background(), map[string]string{"k": "v"}); err != nil {
		t.Fatalf("publish after skip-check: %v", err)
	}
}

func TestPublishAsync_DeliversAndReportsID(t *testing.T) {
	c, srv := newTestClient(t)
	mustTopic(t, c, "events")

	p, err := c.NewPublisher("events")
	if err != nil {
		t.Fatalf("new publisher: %v", err)
	}
	defer p.Stop()

	var (
		mu     sync.Mutex
		gotID  string
		gotErr error
		done   = make(chan struct{})
	)
	p.PublishAsync(context.Background(), map[string]string{"hello": "world"}, nil, func(id string, err error) {
		mu.Lock()
		gotID, gotErr = id, err
		mu.Unlock()
		close(done)
	})

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("callback was not invoked")
	}

	mu.Lock()
	defer mu.Unlock()
	if gotErr != nil {
		t.Fatalf("unexpected callback error: %v", gotErr)
	}
	if gotID == "" {
		t.Fatal("expected a server-assigned message id")
	}
	if got := len(srv.Messages()); got != 1 {
		t.Fatalf("expected 1 delivered message, got %d", got)
	}
}

func TestPublishAsync_NilBodyReportsErrorToCallback(t *testing.T) {
	c, _ := newTestClient(t)
	mustTopic(t, c, "events")

	p, err := c.NewPublisher("events")
	if err != nil {
		t.Fatalf("new publisher: %v", err)
	}
	defer p.Stop()

	done := make(chan error, 1)
	p.PublishAsync(context.Background(), nil, nil, func(_ string, err error) { done <- err })

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected an error for a nil body")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("callback was not invoked for a nil body")
	}
}

func TestWithBatchSettings_TunesOverDefaults(t *testing.T) {
	c, _ := newTestClient(t)
	mustTopic(t, c, "events")

	p, err := c.NewPublisher("events", WithBatchSettings(func(s *pubsub.PublishSettings) {
		s.CountThreshold = 250
		s.DelayThreshold = 100 * time.Millisecond
	}))
	if err != nil {
		t.Fatalf("new publisher: %v", err)
	}
	defer p.Stop()

	got := p.(*publisher).topic.PublishSettings
	if got.CountThreshold != 250 {
		t.Fatalf("CountThreshold = %d, want 250", got.CountThreshold)
	}
	if got.DelayThreshold != 100*time.Millisecond {
		t.Fatalf("DelayThreshold = %v, want 100ms", got.DelayThreshold)
	}
	// A field left untouched keeps the client default.
	if got.ByteThreshold != pubsub.DefaultPublishSettings.ByteThreshold {
		t.Fatalf("ByteThreshold = %d, want default %d", got.ByteThreshold, pubsub.DefaultPublishSettings.ByteThreshold)
	}
}
