# audit

Build and publish platform audit events (schema v2). Generic and self-contained:
the service name, the HMAC key and the transport are supplied by the caller, so
nothing environment-specific lives here.

```go
import (
    "github.com/zarvhq/zarv-go/pkg/audit"
    "github.com/zarvhq/zarv-go/pkg/gcp/pubsub"
)

pub, _ := psClient.NewPublisher("audit_events_zarv_id", pubsub.WithoutTopicExistsCheck())
em, _ := audit.New(audit.Config{
    Service:   "zarv-id",
    Publisher: pub,          // any type with PublishAsync satisfies audit.Publisher
    HMACKey:   hmacKey,      // from Secret Manager; only needed for Subject
})

em.Emit(ctx, audit.Event{
    Action:  "zarv-id.verification.read",
    Actor:   audit.Actor{Type: audit.ActorUser, ID: userID, WorkspaceID: wsID, IP: ip},
    Target:  &audit.Target{Type: "verification", ID: id},
    Subject: em.Subject(cpf),                       // hmac:v1:… never the raw CPF
    Data:    map[string]any{"categories": []string{"identity", "address"}, "records": 1},
})
```

- **Emit never blocks or fails the audited operation.** It stamps `id`,
  `occurred_at` and `schema`, validates (actor type + action required), redacts
  sensitive keys in `changes`/`metadata`, and publishes asynchronously. Transport
  failures are logged, not returned.
- **Reads are receipts, not payloads.** Put categories/counts in `Data`; never a
  response body.
- **`Subject`** returns a keyed HMAC (`hmac:v1:<hex>`) of a CPF/CNPJ so the trail
  can answer "who accessed document X" without storing the number.
- Publish attributes carry `action` and `service` for routing/filtering.
