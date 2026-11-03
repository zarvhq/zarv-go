package ingestion

import (
	"fmt"
	"regexp"
	"strings"
)

// EventMetadataField is the field of the record that holds the metadata of a
// LegacyEvent, so it lands in the table beside the record's own fields.
const EventMetadataField = "event_metadata"

// LegacyEvent is the envelope producers sent to zarv-data-api's /v1/ingestion:
// a domain and a service instead of a table name, and the metadata beside the
// record. Event turns it into what the gateway reads, exactly as the data-api
// did, so a producer that moves from that route lands its rows in the same
// tables.
type LegacyEvent struct {
	Domain      string
	Service     string
	Operation   string
	Description string
	// Data is the record. Required.
	Data map[string]any
	// Metadata is kept in Data under EventMetadataField when it is not nil.
	Metadata map[string]any
}

var (
	notName           = regexp.MustCompile(`[^a-z0-9]+`)
	operationSynonyms = map[string]string{"CREATE": "INSERT", "REPLACE": "UPDATE", "UPSERT": "UPDATE"}
)

// Event is the gateway event of a LegacyEvent. The table is
// `<domain>_<service>`, normalised; id/providers is split by the provider and
// the endpoint called, `id_providers_<metadata.provider>_<metadata.path>`. CREATE,
// REPLACE and UPSERT are translated; any other operation goes as sent, for the
// gateway to judge. The caller's Data and Metadata are not modified.
func (l LegacyEvent) Event() (Event, error) {
	table, err := l.tableName()
	if err != nil {
		return Event{}, err
	}
	if l.Data == nil {
		return Event{}, fmt.Errorf("%w: %s has no Data", ErrInvalidEvent, table)
	}
	data := make(map[string]any, len(l.Data)+1)
	for k, v := range l.Data {
		data[k] = v
	}
	if l.Metadata != nil {
		data[EventMetadataField] = l.Metadata
	}
	return Event{
		TableName:   table,
		Operation:   legacyOperation(l.Operation),
		Description: l.Description,
		Data:        data,
	}, nil
}

func (l LegacyEvent) tableName() (string, error) {
	d, s := normalise(l.Domain), normalise(l.Service)
	if d == "" || s == "" {
		return "", fmt.Errorf("%w: domain and service are required (got %q, %q)", ErrInvalidEvent, l.Domain, l.Service)
	}
	parts := []string{d, s}
	if d == "id" && s == "providers" {
		for _, field := range []string{"provider", "path"} {
			v, _ := l.Metadata[field].(string)
			if normalise(v) == "" {
				return "", fmt.Errorf("%w: an id/providers event needs metadata.%s", ErrInvalidEvent, field)
			}
			parts = append(parts, normalise(v))
		}
	}
	return strings.Join(parts, "_"), nil
}

func normalise(part string) string {
	return strings.Trim(notName.ReplaceAllString(strings.ToLower(part), "_"), "_")
}

func legacyOperation(op string) string {
	if s, ok := operationSynonyms[strings.ToUpper(strings.TrimSpace(op))]; ok {
		return s
	}
	return op
}
