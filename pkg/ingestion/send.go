package ingestion

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Bounds on the wait between attempts.
const (
	baseBackoff   = 200 * time.Millisecond
	maxBackoff    = 5 * time.Second
	maxRetryAfter = 30 * time.Second
	maxReplyBytes = 64 << 10
)

// reply202 is the gateway's answer to an accepted request.
type reply202 struct {
	Accepted int      `json:"accepted"`
	Rejected []string `json:"rejected"`
	Archived int      `json:"archived"`
}

// Send posts ev to the gateway and returns once it is accepted or refused.
//
// A 503, a 5xx or a transport error is retried, honoring Retry-After, up to
// MaxAttempts and within ctx; a retry is safe because the gateway identifies a
// row by its content. A 4xx, or a 202 that rejects the event, is a
// *RefusedError; spent attempts are an *UnavailableError; a canceled ctx
// returns ctx's error.
func (c *Client) Send(ctx context.Context, ev Event) error {
	env, err := ev.envelope()
	if err != nil {
		return err
	}
	body, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("%w: %s does not encode as JSON: %v", ErrInvalidEvent, ev.TableName, err)
	}

	last := &UnavailableError{Table: ev.TableName}
	for attempt := 1; ; attempt++ {
		retry, wait, err := c.attempt(ctx, ev.TableName, body, last)
		if !retry {
			return err
		}
		last.Attempts = attempt
		if attempt >= c.maxAttempts {
			return last
		}
		if wait == 0 {
			wait = backoff(attempt)
		}
		if err := c.sleep(ctx, wait); err != nil {
			return err
		}
	}
}

// attempt makes one request. When retry is false, err is the outcome (nil on
// success); when true, last holds the failure and wait is the gateway's
// Retry-After, zero if it gave none.
func (c *Client) attempt(ctx context.Context, table string, body []byte, last *UnavailableError) (retry bool, wait time.Duration, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return false, 0, fmt.Errorf("ingestion: building the request for %s: %w", table, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.key)

	resp, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return false, 0, ctx.Err()
		}
		last.Status, last.Err = 0, err
		return true, 0, nil
	}
	defer func() { _ = resp.Body.Close() }()
	reply, _ := io.ReadAll(io.LimitReader(resp.Body, maxReplyBytes))

	switch {
	case resp.StatusCode == http.StatusAccepted || resp.StatusCode == http.StatusOK:
		return false, 0, accepted202(table, resp.StatusCode, reply)
	case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500:
		last.Status, last.Err = resp.StatusCode, nil
		return true, retryAfter(resp.Header.Get("Retry-After")), nil
	default:
		return false, 0, &RefusedError{Table: table, Status: resp.StatusCode, Reason: strings.TrimSpace(string(reply))}
	}
}

// accepted202 turns a rejection inside an accepted request into an error: the
// gateway took the request and refused the event in it. A body that does not
// parse is still an acceptance; the status is the gateway's word.
func accepted202(table string, status int, reply []byte) error {
	var r reply202
	if err := json.Unmarshal(reply, &r); err != nil {
		return nil
	}
	if len(r.Rejected) > 0 {
		return &RefusedError{Table: table, Status: status, Reason: strings.Join(r.Rejected, "; ")}
	}
	return nil
}

// retryAfter reads the header in seconds, capped. Zero when absent or unreadable.
func retryAfter(h string) time.Duration {
	s, err := strconv.Atoi(strings.TrimSpace(h))
	if err != nil || s <= 0 {
		return 0
	}
	return min(time.Duration(s)*time.Second, maxRetryAfter)
}

// backoff is the wait when the gateway gave none: doubling from baseBackoff,
// capped at maxBackoff.
func backoff(attempt int) time.Duration {
	return min(baseBackoff<<(attempt-1), maxBackoff)
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
