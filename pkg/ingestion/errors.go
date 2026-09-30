package ingestion

import (
	"errors"
	"fmt"
)

// Sentinels for errors.Is. Send returns a *RefusedError or an *UnavailableError
// that matches one of them.
var (
	// ErrRefused means the gateway answered and refused the event. Sending it
	// again gets the same answer.
	ErrRefused = errors.New("ingestion: refused by the gateway")
	// ErrUnavailable means the gateway did not take the event within the
	// attempts: it was busy, failing or unreachable. The event may be sent again.
	ErrUnavailable = errors.New("ingestion: gateway unavailable")
)

// RefusedError is the gateway's refusal: a 4xx, or a 202 whose body rejects
// the event.
type RefusedError struct {
	// Table is the event's table_name.
	Table string
	// Status is the HTTP status the gateway answered with.
	Status int
	// Reason is the gateway's own words.
	Reason string
}

func (e *RefusedError) Error() string {
	return fmt.Sprintf("ingestion: %s refused by the gateway (%d): %s", e.Table, e.Status, e.Reason)
}

// Is makes errors.Is(err, ErrRefused) true.
func (e *RefusedError) Is(target error) bool { return target == ErrRefused }

// UnavailableError is the last failure after every attempt was spent.
type UnavailableError struct {
	// Table is the event's table_name.
	Table string
	// Attempts is how many requests were made.
	Attempts int
	// Status is the last HTTP status, or zero when no answer came.
	Status int
	// Err is the last transport error, if any.
	Err error
}

func (e *UnavailableError) Error() string {
	if e.Status != 0 {
		return fmt.Sprintf("ingestion: %s not accepted after %d attempts: last status %d", e.Table, e.Attempts, e.Status)
	}
	return fmt.Sprintf("ingestion: %s not accepted after %d attempts: %v", e.Table, e.Attempts, e.Err)
}

// Is makes errors.Is(err, ErrUnavailable) true.
func (e *UnavailableError) Is(target error) bool { return target == ErrUnavailable }

// Unwrap exposes the last transport error.
func (e *UnavailableError) Unwrap() error { return e.Err }
