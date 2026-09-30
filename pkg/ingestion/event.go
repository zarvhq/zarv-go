package ingestion

import (
	"fmt"
	"strings"
)

// EventMetadataField is where the event's metadata travels inside `data`.
// Not `metadata`: that is a field of the producers' own records in several
// domains (asset/asset, users/audit, id/verification, ...), and it stays theirs.
const EventMetadataField = "event_metadata"

// Event is what a producer sends: which domain and service it belongs to, what
// happened, and the record.
type Event struct {
	// Domain and Service decide the table; see TableName.
	Domain  string
	Service string
	// Operation is INSERT, UPDATE or DELETE, in any case; empty means INSERT.
	// CREATE is read as INSERT, and REPLACE and UPSERT as UPDATE.
	Operation   string
	Description string
	// Data is the record. Required.
	Data map[string]any
	// Metadata travels inside the record, under EventMetadataField. For
	// id/providers it also names the table: see TableName.
	Metadata map[string]any
}

// synonyms are the verbs producers have sent that mean one the gateway accepts.
var synonyms = map[string]string{"CREATE": "INSERT", "REPLACE": "UPDATE", "UPSERT": "UPDATE"}

// envelope is the event in the gateway's shape:
// {table_name, operation, description, data}. It never modifies e.Data.
func (e Event) envelope() (map[string]any, error) {
	table, err := TableName(e.Domain, e.Service, e.Metadata)
	if err != nil {
		return nil, err
	}
	if e.Data == nil {
		return nil, fmt.Errorf("%w: %s has no data", ErrInvalidEvent, table)
	}

	data := make(map[string]any, len(e.Data)+1)
	for k, v := range e.Data {
		data[k] = v
	}
	if e.Metadata != nil {
		data[EventMetadataField] = e.Metadata
	}

	env := map[string]any{"table_name": table, "data": data}
	if op := operation(e.Operation); op != "" {
		env["operation"] = op
	}
	if e.Description != "" {
		env["description"] = e.Description
	}
	return env, nil
}

// operation uppercases the verb and translates its synonyms. An empty one is
// left out, which the gateway reads as INSERT; an unknown one goes through for
// the gateway to refuse, so the producer hears about it.
func operation(op string) string {
	op = strings.ToUpper(strings.TrimSpace(op))
	if s, ok := synonyms[op]; ok {
		return s
	}
	return op
}
