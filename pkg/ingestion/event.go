package ingestion

import (
	"errors"
	"fmt"
	"strings"
)

// ErrInvalidEvent is what the package wraps when an event cannot become a
// request: no table to route it to, or no record to send.
var ErrInvalidEvent = errors.New("ingestion: invalid event")

// Event is the gateway's body, field for field. The package adds nothing to it
// and translates nothing: the gateway judges the table name and the operation,
// and its answer reaches the caller.
type Event struct {
	// TableName is the bronze table the record lands in. Required.
	TableName string
	// Data is the record. Required.
	Data map[string]any
	// Operation is INSERT, UPDATE or DELETE. Empty means INSERT.
	Operation string
	// Description is free text, stored with the row.
	Description string
	// UniqueKey names the field of Data that identifies the record. Empty
	// means the gateway's default.
	UniqueKey string
}

// envelope is the event as the gateway reads it. It never modifies e.Data, and
// it leaves an empty optional field out so the gateway's default applies.
func (e Event) envelope() (map[string]any, error) {
	if strings.TrimSpace(e.TableName) == "" {
		return nil, fmt.Errorf("%w: TableName is required", ErrInvalidEvent)
	}
	if e.Data == nil {
		return nil, fmt.Errorf("%w: %s has no Data", ErrInvalidEvent, e.TableName)
	}
	env := map[string]any{"table_name": e.TableName, "data": e.Data}
	for field, value := range map[string]string{
		"operation":   e.Operation,
		"description": e.Description,
		"unique_key":  e.UniqueKey,
	} {
		if value != "" {
			env[field] = value
		}
	}
	return env, nil
}
