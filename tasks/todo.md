# Tasks — `pkg/ingestion`

Plan and rationale in [`plan.md`](plan.md). Ordered; dependencies noted.
Sizes: XS 1 file · S 1–2 · M 3–5 · L 5–8.

Every task: `make ci` (lint, test, build) green before its commit.

---

## Phase 1 — the package

### Task 1: configuration and `New`

**Done:** 2026-09-30 -- 10 tests, `-race`, `golangci-lint` v2.8.0 clean.

**Description:** `pkg/ingestion` with `Config{URL, Key, HTTPClient, Logger,
MaxAttempts}` and `New(cfg) (*Client, error)`. URL and key default to
`ZARV_INGESTION_URL` and `ZARV_INGESTION_KEY`; a `Config` field overrides each.

**Acceptance criteria:**
- [x] `New` fails when the URL is unset, naming `ZARV_INGESTION_URL`
- [x] `New` fails when the URL is not an absolute `http` or `https` URL, and when
      the key is unset, naming the variable and never the value
- [x] Defaults: an `http.Client` with a timeout, `slog.Default()`, a bounded
      number of attempts

**Verification:**
- [ ] `go test ./pkg/ingestion/...` with `t.Setenv` for each case

**Dependencies:** None
**Files:** `pkg/ingestion/client.go`, `pkg/ingestion/client_test.go`, `pkg/ingestion/doc.go`
**Scope:** S

### Task 2: the Event, as the gateway reads it

**Done:** 2026-09-30. A first version carried the legacy conversion (domain and
service to table_name, the id/providers rule, event_metadata, the operation's
synonyms); the operator moved all of it to the data-api (`74d40b8` → the next
commit), so zarv-go carries nothing legacy.

**Description:** `Event{TableName, Data, Operation, Description, UniqueKey}` is
the gateway's body, field for field. The package adds nothing and translates
nothing; an empty optional field is left out so the gateway's default applies.

**Acceptance criteria:**
- [x] The envelope is the Event as it is: `{table_name, data, operation?,
      description?, unique_key?}`
- [x] Only a missing table name or data is refused (`ErrInvalidEvent`): the rest
      is the gateway's to judge
- [x] The caller's `Data` is never modified

**Verification:**
- [x] `go test -race ./pkg/ingestion/...`; `golangci-lint` clean

**Dependencies:** Task 1
**Files:** `pkg/ingestion/event.go`, `pkg/ingestion/event_test.go`
**Scope:** S

### Task 3: `Send`, its errors, and its retries

**Description:** `Send(ctx, Event) error` posts the envelope with the Bearer
key. A 503, 5xx, timeout or connection error is retried with backoff, honouring
`Retry-After`, within the context and up to `MaxAttempts`. A 400/401/413 is a
typed error carrying the gateway's reason; a 202 is parsed, and a `rejected`
entry is an error too.

**Acceptance criteria:**
- [x] Against an `httptest` gateway: 202 with nothing rejected → nil; 202 with a
      rejection → an error naming the reason; 400, 401 and 413 → not retried
- [x] 503 with `Retry-After` → retried after the delay, then succeeds; context
      cancelled → stops and returns the context's error
- [x] The same bytes on every attempt (the record is never rebuilt between
      retries), and the key never appears in an error or a log line

**Verification:**
- [x] `go test -race ./pkg/ingestion/...`

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
converts the legacy body into an `ingestion.Event` -- the table name from domain
and service with the `id/providers` rule, the event's metadata under
`event_metadata`, CREATE/REPLACE/UPSERT translated -- and calls `Client.Send`,
returning the gateway's refusal to the producer. The conversion and its history
test live in the data-api; this package is only the transport. This is T8 of
`zarv-data-pipeline/tasks/todo.md`.

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
