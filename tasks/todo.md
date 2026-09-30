# Tasks — `pkg/ingestion`

Plan and rationale in [`plan.md`](plan.md). Ordered; dependencies noted.
Sizes: XS 1 file · S 1–2 · M 3–5 · L 5–8.

Every task: `make ci` (lint, test, build) green before its commit.

---

## Phase 1 — the package

### Task 1: configuration and `New`

**Description:** `pkg/ingestion` with `Config{URL, Key, HTTPClient, Logger,
MaxAttempts}` and `New(cfg) (*Client, error)`. URL and key default to
`ZARV_INGESTION_URL` and `ZARV_INGESTION_KEY`; a `Config` field overrides each.

**Acceptance criteria:**
- [ ] `New` fails when the URL is unset, naming `ZARV_INGESTION_URL`
- [ ] `New` fails when the URL is not an absolute `http` or `https` URL, and when
      the key is unset, naming the variable and never the value
- [ ] Defaults: an `http.Client` with a timeout, `slog.Default()`, a bounded
      number of attempts

**Verification:**
- [ ] `go test ./pkg/ingestion/...` with `t.Setenv` for each case

**Dependencies:** None
**Files:** `pkg/ingestion/client.go`, `pkg/ingestion/client_test.go`, `pkg/ingestion/doc.go`
**Scope:** S

### Task 2: the envelope and `TableName`

**Description:** `Event{Domain, Service, Operation, Description, Data, Metadata}`
and the envelope it becomes: `{table_name, operation, description, data}`, with
`data` = the event's fields plus `metadata`. `TableName(domain, service,
metadata)` is exported and is the one implementation of the naming rule.

**Acceptance criteria:**
- [ ] `<domain>_<service>` normalised: lowercase, anything outside `[a-z0-9_]`
      to `_`, repeats collapsed, trimmed (`id` + `Verification failed` →
      `id_verification_failed`)
- [ ] `id` + `providers` → `id_providers_<metadata.provider>_<metadata.path>`,
      normalised; a providers event without `provider` or `path` is an error naming
      the field
- [ ] A table-driven test over the 94 historical `domain_service` and the 12
      `provider|path` pairs: no two onto one name, each within the gateway's
      `naming.pattern`

**Verification:**
- [ ] `go test ./pkg/ingestion/...`

**Dependencies:** Task 1
**Files:** `pkg/ingestion/event.go`, `pkg/ingestion/table.go`, their tests, a fixture of the historical names
**Scope:** M

### Task 3: `Send`, its errors, and its retries

**Description:** `Send(ctx, Event) error` posts the envelope with the Bearer
key. A 503, 5xx, timeout or connection error is retried with backoff, honouring
`Retry-After`, within the context and up to `MaxAttempts`. A 400/401/413 is a
typed error carrying the gateway's reason; a 202 is parsed, and a `rejected`
entry is an error too.

**Acceptance criteria:**
- [ ] Against an `httptest` gateway: 202 with nothing rejected → nil; 202 with a
      rejection → an error naming the reason; 400, 401 and 413 → not retried
- [ ] 503 with `Retry-After` → retried after the delay, then succeeds; context
      cancelled → stops and returns the context's error
- [ ] The same bytes on every attempt (the record is never rebuilt between
      retries), and the key never appears in an error or a log line

**Verification:**
- [ ] `go test -race ./pkg/ingestion/...`

**Dependencies:** Task 2
**Files:** `pkg/ingestion/send.go`, `pkg/ingestion/errors.go`, their tests
**Scope:** M

### Task 4: observability and an example

**Description:** One structured log line per failed send (table, status, attempt,
reason — never the payload or the key), and a runnable `Example` in
`example_test.go` that is also the README's snippet.

**Acceptance criteria:**
- [ ] Logs go to `Config.Logger`; a successful send logs nothing at info level
- [ ] `go test` runs the example

**Dependencies:** Task 3
**Files:** `pkg/ingestion/send.go`, `pkg/ingestion/example_test.go`
**Scope:** S

### Checkpoint: the package
- [ ] `make ci` green
- [ ] The operator has read the API and the README draft

---

## Phase 2 — ship it

### Task 5: docs, lint, release `v0.1.0`

**Description:** `pkg/ingestion/README.md` in Portuguese (what it does, the
variables, the naming rule, what each error means, the retry guarantee),
`CHANGELOG.md`, and the tag. The tag is released by the operator: it is what
every consumer then pins.

**Acceptance criteria:**
- [ ] README and godoc complete; `golangci-lint run` clean
- [ ] `CHANGELOG.md` has the `pkg/ingestion` entry
- [ ] Tag `v0.1.0` pushed (`make release VERSION=0.1.0`), after the operator's go

**Dependencies:** Task 4
**Files:** `pkg/ingestion/README.md`, `CHANGELOG.md`
**Scope:** S

---

## Phase 3 — use it

### Task 6: the data-api bridge on top of it

**Description:** In `zarv-data-api`, `internal/usecase/ingestion_usecase.go`
becomes a thin adapter: it maps the request to `ingestion.Event` and calls
`Client.Send`, returning the gateway's refusal to the producer. This is T8 of
`zarv-data-pipeline/tasks/todo.md`, now implemented with this package instead of
its own client and naming code.

**Acceptance criteria:**
- [ ] The use case depends on the ingestion client only; the Pub/Sub publisher,
      size check, GCS fallback and field strip are gone from it
- [ ] `ZARV_INGESTION_URL` and `ZARV_INGESTION_KEY` in the data-api's deployment,
      dev and prod, the key sealed
- [ ] In dev, a real event lands in its bronze table with `data.metadata`

**Verification:**
- [ ] `go test ./...` in `zarv-data-api`; the dev check above

**Dependencies:** Task 5; `zarv-data-pipeline` T7 (the gateway in the target environment)
**Files:** `zarv-data-api: internal/usecase/ingestion_usecase.go`, its test, wiring in `cmd/`, `zarv-applications` for the variables
**Scope:** M

### Task 7: one team sends directly

**Description:** One Go service that produces through the data-api today moves to
`pkg/ingestion` directly — the pilot for `zarv-data-pipeline` T22. Its events must
land in the same table, with the same columns, as through the data-api.

**Acceptance criteria:**
- [ ] The service sends with `Client.Send`; its events in dev land in the table
      the data-api route used
- [ ] A week without a refused event, or every refusal explained

**Dependencies:** Task 6
**Files:** the pilot service's repository
**Scope:** S

### Checkpoint: complete
- [ ] `v0.1.0` released; the data-api and one producer on it
- [ ] `zarv-data-pipeline` T8 and T22 point here
