package ingestion

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// ErrInvalidEvent is what the package wraps when an event cannot become a
// request: no table to route it to, or no data to send.
var ErrInvalidEvent = errors.New("ingestion: invalid event")

// TablePattern is every table name TableName can return: a lowercase letter,
// then lowercase letters, digits and underscores, 128 characters at most. The
// gateway's naming.pattern has to accept it.
var TablePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,127}$`)

var notName = regexp.MustCompile(`[^a-z0-9]+`)

// TableName is the bronze table an event lands in: `<domain>_<service>`, each
// part lowercased and with every run of characters outside [a-z0-9] turned into
// one underscore, e.g. id + "Verification failed" → id_verification_failed.
//
// The one exception is id + providers, split by the provider called and the
// endpoint it was called on, read from the event's metadata:
// id_providers_<metadata.provider>_<metadata.path>.
//
// It is the only implementation of the rule, so a producer on the data-api
// route and one sending directly land in the same table.
func TableName(domain, service string, metadata map[string]any) (string, error) {
	d, s := normalise(domain), normalise(service)
	if d == "" || s == "" {
		return "", fmt.Errorf("%w: domain and service are required (got %q, %q)", ErrInvalidEvent, domain, service)
	}
	parts := []string{d, s}
	if d == "id" && s == "providers" {
		for _, field := range []string{"provider", "path"} {
			v, _ := metadata[field].(string)
			if normalise(v) == "" {
				return "", fmt.Errorf("%w: an id/providers event needs metadata.%s", ErrInvalidEvent, field)
			}
			parts = append(parts, normalise(v))
		}
	}
	name := strings.Join(parts, "_")
	if !TablePattern.MatchString(name) {
		return "", fmt.Errorf("%w: %q is not a table name (%s)", ErrInvalidEvent, name, TablePattern)
	}
	return name, nil
}

// normalise lowercases a part and turns every run of characters outside
// [a-z0-9] into one underscore, trimmed at both ends.
func normalise(part string) string {
	return strings.Trim(notName.ReplaceAllString(strings.ToLower(part), "_"), "_")
}
