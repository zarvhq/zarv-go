package ingestion

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// gateway is an httptest stand-in for the ingestion gateway. Each request is
// answered by the next reply; the last one repeats.
type gateway struct {
	t       *testing.T
	mu      sync.Mutex
	replies []reply
	bodies  [][]byte
	headers []http.Header
	paths   []string
}

type reply struct {
	status     int
	body       string
	retryAfter string
}

func newGateway(t *testing.T, replies ...reply) (*gateway, *httptest.Server) {
	t.Helper()
	g := &gateway{t: t, replies: replies}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		g.mu.Lock()
		g.bodies = append(g.bodies, b)
		g.headers = append(g.headers, r.Header.Clone())
		g.paths = append(g.paths, r.Method+" "+r.URL.Path)
		n := len(g.bodies) - 1
		g.mu.Unlock()
		rep := g.replies[min(n, len(g.replies)-1)]
		if rep.retryAfter != "" {
			w.Header().Set("Retry-After", rep.retryAfter)
		}
		w.WriteHeader(rep.status)
		_, _ = io.WriteString(w, rep.body)
	}))
	t.Cleanup(srv.Close)
	return g, srv
}

func (g *gateway) calls() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return len(g.bodies)
}

// client builds a Client against srv whose waits are recorded, not slept.
func client(t *testing.T, url string, attempts int) (*Client, *[]time.Duration) {
	t.Helper()
	c, err := New(Config{URL: url, Key: secret, MaxAttempts: attempts})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	var waits []time.Duration
	c.sleep = func(ctx context.Context, d time.Duration) error {
		waits = append(waits, d)
		return ctx.Err()
	}
	return c, &waits
}

var plan = Event{TableName: "billing_plan", Operation: "UPDATE", Data: map[string]any{"id": "p1"}}

const accepted = `{"accepted":1,"rejected":null}`

func TestSendPostsTheEventWithTheKey(t *testing.T) {
	g, srv := newGateway(t, reply{status: http.StatusAccepted, body: accepted})
	c, _ := client(t, srv.URL, 0)

	if err := c.Send(context.Background(), plan); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if g.paths[0] != "POST /v1/ingestion" {
		t.Errorf("request = %s, want POST /v1/ingestion", g.paths[0])
	}
	if got := g.headers[0].Get("Authorization"); got != "Bearer "+secret {
		t.Errorf("Authorization = %q", got)
	}
	if got := g.headers[0].Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q", got)
	}
	var body map[string]any
	if err := json.Unmarshal(g.bodies[0], &body); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if body["table_name"] != "billing_plan" || body["operation"] != "UPDATE" {
		t.Errorf("body = %v", body)
	}
}

// The gateway archives an oversize event whole and says so. That is kept, not
// lost, so it is a success.
func TestAnArchivedEventIsASuccess(t *testing.T) {
	_, srv := newGateway(t, reply{status: http.StatusAccepted, body: `{"accepted":0,"rejected":null,"archived":1}`})
	c, _ := client(t, srv.URL, 0)
	if err := c.Send(context.Background(), plan); err != nil {
		t.Fatalf("Send: %v", err)
	}
}

// A 202 is not a success when it carries a refusal: the gateway accepted the
// request and refused the event inside it. Treating it as success is silent loss.
func TestA202CarryingARefusalIsAnError(t *testing.T) {
	g, srv := newGateway(t, reply{status: http.StatusAccepted,
		body: `{"accepted":0,"rejected":["event 0: \"operation\" is \"CREATE\" (use INSERT, UPDATE or DELETE)"]}`})
	c, _ := client(t, srv.URL, 0)

	err := c.Send(context.Background(), plan)
	var refused *RefusedError
	if !errors.As(err, &refused) || !errors.Is(err, ErrRefused) {
		t.Fatalf("err = %v, want a RefusedError", err)
	}
	if refused.Status != http.StatusAccepted || !strings.Contains(refused.Reason, `"CREATE"`) || refused.Table != "billing_plan" {
		t.Errorf("RefusedError = %+v", refused)
	}
	if g.calls() != 1 {
		t.Errorf("a refusal was retried: %d calls", g.calls())
	}
}

// 400, 401 and 413 are the gateway's answer about the request itself; sending
// it again gets the same answer.
func TestARefusedRequestIsNotRetried(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusRequestEntityTooLarge} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			g, srv := newGateway(t, reply{status: status, body: "the gateway's reason\n"})
			c, _ := client(t, srv.URL, 0)

			err := c.Send(context.Background(), plan)
			var refused *RefusedError
			if !errors.As(err, &refused) {
				t.Fatalf("err = %v, want a RefusedError", err)
			}
			if refused.Status != status || refused.Reason != "the gateway's reason" {
				t.Errorf("RefusedError = %+v", refused)
			}
			if g.calls() != 1 {
				t.Errorf("retried %d times", g.calls()-1)
			}
		})
	}
}

// A 503 is the gateway's buffer full, with Retry-After. The id is the record's
// content, so sending again is the same row.
func TestA503IsRetriedAfterWhatTheGatewaySays(t *testing.T) {
	g, srv := newGateway(t,
		reply{status: http.StatusServiceUnavailable, retryAfter: "2"},
		reply{status: http.StatusServiceUnavailable, retryAfter: "2"},
		reply{status: http.StatusAccepted, body: accepted},
	)
	c, waits := client(t, srv.URL, 0)

	if err := c.Send(context.Background(), plan); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if g.calls() != 3 {
		t.Errorf("calls = %d, want 3", g.calls())
	}
	if len(*waits) != 2 || (*waits)[0] != 2*time.Second || (*waits)[1] != 2*time.Second {
		t.Errorf("waits = %v, want [2s 2s]", *waits)
	}
}

// Without Retry-After the wait grows, and never passes the ceiling.
func TestA5xxBacksOffAndGrows(t *testing.T) {
	_, srv := newGateway(t,
		reply{status: http.StatusInternalServerError},
		reply{status: http.StatusBadGateway},
		reply{status: http.StatusGatewayTimeout},
		reply{status: http.StatusAccepted, body: accepted},
	)
	c, waits := client(t, srv.URL, 5)

	if err := c.Send(context.Background(), plan); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if len(*waits) != 3 {
		t.Fatalf("waits = %v, want three", *waits)
	}
	for i := 1; i < len(*waits); i++ {
		if (*waits)[i] <= (*waits)[i-1] {
			t.Errorf("the backoff did not grow: %v", *waits)
		}
	}
	for _, w := range *waits {
		if w <= 0 || w > maxBackoff {
			t.Errorf("a wait of %s is outside (0, %s]", w, maxBackoff)
		}
	}
}

// A Retry-After beyond the ceiling is capped: a gateway asking for an hour
// would hold the caller's goroutine for one.
func TestAHugeRetryAfterIsCapped(t *testing.T) {
	_, srv := newGateway(t,
		reply{status: http.StatusServiceUnavailable, retryAfter: "3600"},
		reply{status: http.StatusAccepted, body: accepted},
	)
	c, waits := client(t, srv.URL, 0)
	if err := c.Send(context.Background(), plan); err != nil {
		t.Fatal(err)
	}
	if (*waits)[0] != maxRetryAfter {
		t.Errorf("wait = %s, want the %s cap", (*waits)[0], maxRetryAfter)
	}
}

func TestRetriesStopAtMaxAttempts(t *testing.T) {
	g, srv := newGateway(t, reply{status: http.StatusServiceUnavailable, retryAfter: "1"})
	c, _ := client(t, srv.URL, 3)

	err := c.Send(context.Background(), plan)
	var unavailable *UnavailableError
	if !errors.As(err, &unavailable) || !errors.Is(err, ErrUnavailable) {
		t.Fatalf("err = %v, want an UnavailableError", err)
	}
	if unavailable.Attempts != 3 || unavailable.Status != http.StatusServiceUnavailable {
		t.Errorf("UnavailableError = %+v", unavailable)
	}
	if g.calls() != 3 {
		t.Errorf("calls = %d, want 3", g.calls())
	}
}

// A gateway that cannot be reached is retried like a 503, and ends in an
// UnavailableError rather than a raw transport error.
func TestAnUnreachableGatewayIsRetried(t *testing.T) {
	_, srv := newGateway(t, reply{status: http.StatusAccepted})
	url := srv.URL
	srv.Close()
	c, waits := client(t, url, 3)

	err := c.Send(context.Background(), plan)
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("err = %v, want ErrUnavailable", err)
	}
	if len(*waits) != 2 {
		t.Errorf("waits = %v, want two between three attempts", *waits)
	}
}

func TestACancelledContextStopsTheRetries(t *testing.T) {
	g, srv := newGateway(t, reply{status: http.StatusServiceUnavailable, retryAfter: "1"})
	c, _ := client(t, srv.URL, 10)
	ctx, cancel := context.WithCancel(context.Background())
	c.sleep = func(context.Context, time.Duration) error {
		cancel()
		return ctx.Err()
	}

	err := c.Send(ctx, plan)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if g.calls() != 1 {
		t.Errorf("calls = %d, want the one before the wait", g.calls())
	}
}

// Every attempt sends the same bytes: the envelope is built once, so a retry
// can never be a different record.
func TestEveryAttemptSendsTheSameBytes(t *testing.T) {
	g, srv := newGateway(t,
		reply{status: http.StatusServiceUnavailable, retryAfter: "1"},
		reply{status: http.StatusServiceUnavailable, retryAfter: "1"},
		reply{status: http.StatusAccepted, body: accepted},
	)
	c, _ := client(t, srv.URL, 0)
	if err := c.Send(context.Background(), plan); err != nil {
		t.Fatal(err)
	}
	for i := 1; i < len(g.bodies); i++ {
		if string(g.bodies[i]) != string(g.bodies[0]) {
			t.Errorf("attempt %d sent other bytes:\n%s\n%s", i+1, g.bodies[0], g.bodies[i])
		}
	}
}

func TestAnInvalidEventIsNeverSent(t *testing.T) {
	g, srv := newGateway(t, reply{status: http.StatusAccepted, body: accepted})
	c, _ := client(t, srv.URL, 0)
	if err := c.Send(context.Background(), Event{Data: map[string]any{"id": 1}}); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("err = %v, want ErrInvalidEvent", err)
	}
	if g.calls() != 0 {
		t.Errorf("an invalid event reached the gateway")
	}
}

// Errors end up in logs; the key must not be in any of them.
func TestNoErrorCarriesTheKey(t *testing.T) {
	for _, rep := range []reply{
		{status: http.StatusUnauthorized, body: "unauthorized"},
		{status: http.StatusServiceUnavailable, retryAfter: "1"},
	} {
		_, srv := newGateway(t, rep)
		c, _ := client(t, srv.URL, 2)
		if err := c.Send(context.Background(), plan); err == nil || strings.Contains(err.Error(), secret) {
			t.Errorf("status %d: err = %v", rep.status, err)
		}
	}
}
