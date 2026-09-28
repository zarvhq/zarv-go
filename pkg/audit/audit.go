package audit

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

// Publisher is the transport the Emitter publishes through. The zarv-go
// pkg/gcp/pubsub Publisher satisfies it; audit does not import pubsub so it
// stays decoupled and unit-testable. Attributes carry routing/filter keys
// (action, service).
type Publisher interface {
	PublishAsync(ctx context.Context, body any, attributes map[string]string, callback func(id string, err error))
}

// Config configures an Emitter. Service and Publisher are required. HMACKey is
// needed only if Subject is used; it comes from the caller (Secret Manager),
// never from this package.
type Config struct {
	Service   string
	Publisher Publisher
	HMACKey   []byte
	// Logger receives async publish failures; defaults to slog.Default().
	Logger *slog.Logger
}

// Emitter builds, validates and publishes audit events for one service.
type Emitter struct {
	service string
	pub     Publisher
	hmacKey []byte
	log     *slog.Logger
	now     func() time.Time
	newID   func() string
}

// New builds an Emitter from cfg; Service and Publisher are required.
func New(cfg Config) (*Emitter, error) {
	if cfg.Service == "" {
		return nil, errors.New("audit: Service is required")
	}
	if cfg.Publisher == nil {
		return nil, errors.New("audit: Publisher is required")
	}
	log := cfg.Logger
	if log == nil {
		log = slog.Default()
	}
	return &Emitter{
		service: cfg.Service,
		pub:     cfg.Publisher,
		hmacKey: cfg.HMACKey,
		log:     log,
		now:     time.Now,
		newID:   newUUIDv4,
	}, nil
}

// Emit stamps, validates, sanitizes and asynchronously publishes an event. It
// returns an error only for a programming mistake (an invalid event); transport
// failures are asynchronous and logged, never returned, so auditing never
// blocks or fails the audited operation.
func (e *Emitter) Emit(ctx context.Context, ev Event) error {
	if ev.ID == "" {
		ev.ID = e.newID()
	}
	ev.Schema = SchemaVersion
	if ev.OccurredAt == "" {
		ev.OccurredAt = e.now().UTC().Format(time.RFC3339Nano)
	}
	if ev.Actor.Type == "" {
		return errors.New("audit: Actor.Type is required")
	}
	if ev.Action == "" {
		return errors.New("audit: Action is required")
	}
	if ev.Outcome.Status == "" {
		ev.Outcome.Status = OutcomeSuccess
	}
	ev.Changes = redact(ev.Changes)
	ev.Metadata = redact(ev.Metadata)

	attrs := map[string]string{"action": ev.Action, "service": e.service}
	e.pub.PublishAsync(ctx, ev, attrs, func(id string, err error) {
		if err != nil {
			e.log.Error("audit: publish failed",
				slog.String("action", ev.Action),
				slog.String("event_id", ev.ID),
				slog.String("error", err.Error()),
			)
		}
	})
	return nil
}

func newUUIDv4() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("00000000-0000-4000-8000-%012x", time.Now().UnixNano())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
