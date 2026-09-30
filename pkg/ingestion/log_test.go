package ingestion

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"testing"
)

// logged builds a Client whose log lines, at every level, land in the buffer.
func logged(t *testing.T, url string, attempts int) (*Client, *bytes.Buffer) {
	t.Helper()
	var buf bytes.Buffer
	c, _ := client(t, url, attempts)
	c.log = slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	return c, &buf
}

func lines(buf *bytes.Buffer) []map[string]any {
	var out []map[string]any
	for _, l := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if l == "" {
			continue
		}
		var m map[string]any
		_ = json.Unmarshal([]byte(l), &m)
		out = append(out, m)
	}
	return out
}

func atOrAbove(ls []map[string]any, level string) []map[string]any {
	rank := map[string]int{"DEBUG": 0, "INFO": 1, "WARN": 2, "ERROR": 3}
	var out []map[string]any
	for _, l := range ls {
		if rank[l["level"].(string)] >= rank[level] {
			out = append(out, l)
		}
	}
	return out
}

func TestASuccessfulSendLogsNothingAtInfo(t *testing.T) {
	_, srv := newGateway(t, reply{status: http.StatusAccepted, body: accepted})
	c, buf := logged(t, srv.URL, 0)
	if err := c.Send(context.Background(), plan); err != nil {
		t.Fatal(err)
	}
	if got := atOrAbove(lines(buf), "INFO"); len(got) != 0 {
		t.Errorf("a success logged %v", got)
	}
}

// One line per failed send, whatever the number of attempts behind it.
func TestAFailedSendLogsOneLine(t *testing.T) {
	_, srv := newGateway(t, reply{status: http.StatusServiceUnavailable, retryAfter: "1"})
	c, buf := logged(t, srv.URL, 3)
	_ = c.Send(context.Background(), plan)

	got := atOrAbove(lines(buf), "INFO")
	if len(got) != 1 {
		t.Fatalf("lines = %v, want one", got)
	}
	l := got[0]
	if l["level"] != "ERROR" || l["table"] != plan.TableName || l["status"] != float64(503) || l["attempts"] != float64(3) {
		t.Errorf("line = %v", l)
	}
}

func TestARefusalLogsTheGatewaysReason(t *testing.T) {
	_, srv := newGateway(t, reply{status: http.StatusAccepted,
		body: `{"accepted":0,"rejected":["event 0: \"table_name\" does not match the naming pattern"]}`})
	c, buf := logged(t, srv.URL, 0)
	_ = c.Send(context.Background(), plan)

	got := atOrAbove(lines(buf), "INFO")
	if len(got) != 1 || got[0]["status"] != float64(202) || !strings.Contains(got[0]["reason"].(string), "naming pattern") {
		t.Errorf("lines = %v", got)
	}
}

// The log is read by more people than the data: no key, no record, at any level.
func TestTheLogNeverCarriesTheKeyOrThePayload(t *testing.T) {
	ev := Event{TableName: plan.TableName, Data: map[string]any{"cpf": "12345678909"}}
	for _, rep := range []reply{
		{status: http.StatusUnauthorized, body: "unauthorized"},
		{status: http.StatusServiceUnavailable, retryAfter: "1"},
		{status: http.StatusAccepted, body: accepted},
	} {
		_, srv := newGateway(t, rep)
		c, buf := logged(t, srv.URL, 2)
		_ = c.Send(context.Background(), ev)
		if s := buf.String(); strings.Contains(s, secret) || strings.Contains(s, "12345678909") {
			t.Errorf("status %d logged %s", rep.status, s)
		}
	}
}
