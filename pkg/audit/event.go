// Package audit builds and publishes platform audit events (schema v2). It is
// generic and carries nothing environment-specific: the service name, the HMAC
// key and the transport are all supplied by the caller. See the platform audit
// design in zarv-audit.
package audit

// SchemaVersion is the current event schema version stamped on every Event.
const SchemaVersion = 2

// ActorType classifies who (or what) performed an action.
type ActorType string

// Actor types classify who or what performed an action.
const (
	ActorUser      ActorType = "user"       // a logged-in person
	ActorAPIClient ActorType = "api_client" // an API/provider key
	ActorService   ActorType = "service"    // service-to-service call
	ActorSystem    ActorType = "system"     // consumer, cron, job
	ActorAgent     ActorType = "agent"      // AI acting (on behalf of someone)
	ActorExternal  ActorType = "external"   // inbound webhook
	ActorOperator  ActorType = "operator"   // human out-of-band (psql, kubectl)
)

// OutcomeStatus is the result of the audited action.
type OutcomeStatus string

// Outcome statuses record how the action ended.
const (
	OutcomeSuccess OutcomeStatus = "success"
	OutcomeDenied  OutcomeStatus = "denied"
	OutcomeError   OutcomeStatus = "error"
)

// Actor is who performed the action.
type Actor struct {
	Type        ActorType `json:"type"`
	ID          string    `json:"id,omitempty"`
	Sub         string    `json:"sub,omitempty"`
	WorkspaceID string    `json:"workspaceId,omitempty"`
	OrgID       string    `json:"orgId,omitempty"`
	Client      string    `json:"client,omitempty"`
	IP          string    `json:"ip,omitempty"`
	UserAgent   string    `json:"userAgent,omitempty"`
	// Service is set by the consumer from the topic, not by the producer; kept
	// here so the shape matches the stored row.
	Service string `json:"service,omitempty"`
	// OnBehalfOf is set for agents/impersonation: who the actor acts for.
	OnBehalfOf string `json:"onBehalfOf,omitempty"`
}

// Target is the resource the action affected.
type Target struct {
	Type        string `json:"type,omitempty"`
	ID          string `json:"id,omitempty"`
	WorkspaceID string `json:"workspaceId,omitempty"`
	Version     string `json:"version,omitempty"`
}

// Outcome is the result plus an optional reason (e.g. why denied).
type Outcome struct {
	Status OutcomeStatus `json:"status"`
	Reason string        `json:"reason,omitempty"`
}

// Event is one audit record (schema v2). Field names match the BigQuery
// audit.events table.
type Event struct {
	ID         string         `json:"id"`
	Schema     int            `json:"schema"`
	OccurredAt string         `json:"occurredAt"`
	Action     string         `json:"action"` // "<service>.<resource>.<verb>"
	Outcome    Outcome        `json:"outcome"`
	Actor      Actor          `json:"actor"`
	Target     *Target        `json:"target,omitempty"`
	Subject    string         `json:"subject,omitempty"` // HMAC of a CPF/CNPJ, never the raw value
	Changes    map[string]any `json:"changes,omitempty"` // before/after of changed fields
	Data       map[string]any `json:"data,omitempty"`    // read receipt (categories, counts) — never a payload
	RequestID  string         `json:"requestId,omitempty"`
	CausedBy   string         `json:"causedBy,omitempty"`
	Trigger    string         `json:"trigger,omitempty"` // schedule:<job> | queue:<name> | webhook:<source>
	RunID      string         `json:"runId,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}
