package audit

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
)

type capturePublisher struct {
	mu    sync.Mutex
	body  any
	attrs map[string]string
	calls int
	err   error
}

func (c *capturePublisher) PublishAsync(_ context.Context, body any, attrs map[string]string, cb func(string, error)) {
	c.mu.Lock()
	c.body, c.attrs, c.calls = body, attrs, c.calls+1
	c.mu.Unlock()
	cb("srv-1", c.err)
}

func newEmitter(t *testing.T, pub Publisher) *Emitter {
	t.Helper()
	e, err := New(Config{Service: "zarv-id", Publisher: pub, HMACKey: []byte("k")})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return e
}

func TestNew_Validation(t *testing.T) {
	if _, err := New(Config{Publisher: &capturePublisher{}}); err == nil {
		t.Error("expected error without Service")
	}
	if _, err := New(Config{Service: "x"}); err == nil {
		t.Error("expected error without Publisher")
	}
}

func TestEmit_StampsAndPublishesWithAttrs(t *testing.T) {
	pub := &capturePublisher{}
	e := newEmitter(t, pub)

	err := e.Emit(context.Background(), Event{
		Action: "zarv-id.verification.read",
		Actor:  Actor{Type: ActorUser, ID: "u1"},
	})
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	if pub.calls != 1 {
		t.Fatalf("expected 1 publish, got %d", pub.calls)
	}
	ev := pub.body.(Event)
	if ev.ID == "" || ev.OccurredAt == "" {
		t.Error("id/occurredAt not stamped")
	}
	if ev.Schema != SchemaVersion {
		t.Errorf("schema = %d", ev.Schema)
	}
	if ev.Outcome.Status != OutcomeSuccess {
		t.Errorf("default outcome = %q", ev.Outcome.Status)
	}
	if pub.attrs["action"] != "zarv-id.verification.read" || pub.attrs["service"] != "zarv-id" {
		t.Errorf("attrs = %+v", pub.attrs)
	}
}

func TestEmit_RequiresActorTypeAndAction(t *testing.T) {
	e := newEmitter(t, &capturePublisher{})
	if err := e.Emit(context.Background(), Event{Action: "a.b.c"}); err == nil {
		t.Error("expected error without actor type")
	}
	if err := e.Emit(context.Background(), Event{Actor: Actor{Type: ActorSystem}}); err == nil {
		t.Error("expected error without action")
	}
}

func TestEmit_RedactsSensitiveFields(t *testing.T) {
	pub := &capturePublisher{}
	e := newEmitter(t, pub)
	_ = e.Emit(context.Background(), Event{
		Action: "users.user.updated",
		Actor:  Actor{Type: ActorUser},
		Changes: map[string]any{
			"role":     []any{"viewer", "admin"},
			"password": "hunter2",
			"nested":   map[string]any{"api_key": "sk-1", "ok": "v"},
		},
	})
	ev := pub.body.(Event)
	if ev.Changes["password"] != "[redacted]" {
		t.Errorf("password not redacted: %v", ev.Changes["password"])
	}
	if ev.Changes["nested"].(map[string]any)["api_key"] != "[redacted]" {
		t.Errorf("nested api_key not redacted")
	}
	if ev.Changes["nested"].(map[string]any)["ok"] != "v" {
		t.Errorf("non-sensitive nested value dropped")
	}
}

func TestSubject_DeterministicAndFormatted(t *testing.T) {
	e := newEmitter(t, &capturePublisher{})
	a := e.Subject("529.982.247-25")
	b := e.Subject("52998224725") // same digits, different formatting
	if a != b {
		t.Errorf("subject not stable across formatting: %q vs %q", a, b)
	}
	if !strings.HasPrefix(a, "hmac:v1:") {
		t.Errorf("subject not versioned: %q", a)
	}
	if a == "hmac:v1:" || len(a) < 20 {
		t.Errorf("subject looks empty: %q", a)
	}
	if e.Subject("") != "" {
		t.Error("empty input should yield empty subject")
	}
}

func TestEmit_MarshalsToStableJSON(t *testing.T) {
	pub := &capturePublisher{}
	e := newEmitter(t, pub)
	_ = e.Emit(context.Background(), Event{Action: "a.b.c", Actor: Actor{Type: ActorSystem}, Outcome: Outcome{Status: OutcomeDenied, Reason: "no role"}})
	b, err := json.Marshal(pub.body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m["outcome"].(map[string]any)["status"] != "denied" {
		t.Errorf("outcome not serialized: %v", m["outcome"])
	}
}

func TestMaskEmail(t *testing.T) {
	if got := MaskEmail("joao@zarv.com"); got != "j***@zarv.com" {
		t.Errorf("MaskEmail = %q", got)
	}
	if got := MaskEmail("notanemail"); got != "***" {
		t.Errorf("MaskEmail(no @) = %q", got)
	}
}
