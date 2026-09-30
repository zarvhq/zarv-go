# Implementation Plan — `pkg/ingestion`, the producers' client for the gateway

## Overview

A Go package that lets any team send an event to the Brevis ingestion gateway in
a few lines, the right way: the gateway's own body, Bearer auth, retries that
cannot duplicate, and a clear error when the gateway refuses. **It speaks only
the gateway's current body** -- `table_name`, `data`, `operation`,
`description`, `unique_key`. Nothing of the legacy `{domain, service, ...}`
shape comes into zarv-go: converting it is the data-api's job. It fails fast when its configuration is missing — the gateway URL has to
be in the environment.

It is one of the two roads producers take into `bronze` (the migration plan in
`zarv-data-pipeline/tasks/plan.md`):

```
legacy producer ─► data-api /v1/ingestion ─┐
                                           ├─► pkg/ingestion ─► gateway ─► bronze.<table>
new Go producer ───────────────────────────┘
```

**The data-api uses this package too**, as its transport. Its ingestion use case
(T8 there) converts the legacy body -- `<domain>_<service>`, the `id/providers`
rule, `event_metadata`, the operation's synonyms -- into an `ingestion.Event`,
and calls `Client.Send`. The conversion lives there and nowhere else.

## What the gateway expects (read from its code, gateway 0.15.0)

| | |
|---|---|
| request | `POST <url>/v1/ingestion`, one JSON object per request, `Authorization: Bearer <key>` |
| envelope | `{"table_name", "data", "operation"?, "description"?, "unique_key"?}` — `table_name` and `data` required |
| 202 | `{"accepted": n, "rejected": [...], "archived"?: n}` — a 202 can still carry a refusal |
| 400 | malformed, or no event · **401** no or wrong key · **413** body above `max_body` (30 MiB) |
| 503 | buffer full, with `Retry-After: 1` — safe to retry: the id is the content, so a retry is the same row |
| naming | the stream's `naming.pattern` refuses a table name outside it, per event, inside the 202 |

## Architecture decisions

- **Same shape as the rest of zarv-go.** `Config` + `New(cfg) (*Client, error)`
  validating what is required, `slog` defaulting to `slog.Default()`, godoc in
  English, README in Portuguese, `golangci-lint` clean, a semver tag to release.
  `pkg/audit` is the reference.
- **Configuration from the environment, validated at `New`.** `ZARV_INGESTION_URL`
  must be set and be an absolute `http(s)` URL; `ZARV_INGESTION_KEY` must be set.
  A `Config` field overrides each. A missing or malformed value is an error from
  `New` naming the variable, never a failure on the first send.
- **No legacy in zarv-go.** The package takes the gateway's body as it is: no
  domain or service, no naming rule, no translation of the operation. The
  data-api owns the conversion for the producers still on its route.
- **Synchronous `Send`, with bounded retries.** A 503, a 5xx, a timeout or a
  connection error is retried with backoff, honouring `Retry-After`, up to a
  limit and within the caller's context. A 400/401/413, or a refusal inside a
  202, is returned at once as a typed error with the gateway's reason. Retrying is
  safe because the gateway's id is the record's content.
- **No validation the gateway already does.** The client does not re-check size,
  table names or operations: the gateway's answer is the truth, and the client's
  job is to surface it. The only checks are the ones that stop a request being
  built at all (no URL, no key, no table name, no data).

## Dependency graph

```
T1 config + New ──► T2 the Event, as the gateway reads it ──► T3 Send, errors, retries ──► T4 observability + example
                                                                                   │
                                                                                   ▼
                                                                         T5 docs, lint, release v0.1.0
                                                                                   │
                                                  ┌────────────────────────────────┴────────────┐
                                                  ▼                                             ▼
                                   T6 data-api uses it (zarv-data-api)          T7 one team's producer, direct
```

## Task list

Tasks are in [`todo.md`](todo.md).

- **Phase 1 — the package** (T1–T4), each slice test-first against an
  `httptest` gateway.
- **Phase 2 — ship it** (T5): docs, lint, `v0.1.0`.
- **Phase 3 — use it** (T6–T7): the data-api bridge on top of it, then one team
  sending directly.

## Risks and mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| A direct producer names a table differently from the data-api's rule for the same data | Medium — the same fact in two tables | The data-api's rule is written down for teams moving off its route (T22 there); a new producer picks its `table_name` in review |
| A retry lands a second row | Medium | Only if the record changes between attempts; the gateway's id is content-addressed. The client never mutates the record it retries |
| A producer treats a 202 as success when it carried a refusal | **High** — silent loss | `Send` parses the 202 and returns an error for any `rejected` entry |
| The gateway is reachable only inside the cluster | Medium | Documented; a producer outside it needs network access agreed first |
| The key leaks into logs | High | The key is never logged; errors name the variable, not the value |

## Open questions

1. **Variable names.** Proposed `ZARV_INGESTION_URL` and `ZARV_INGESTION_KEY`
   (the gateway reads its list of keys from `ZARV_INGEST_KEYS`). Keep, or align
   with an existing convention?
2. **Async as well?** `pkg/audit` fires and forgets. For data, a synchronous
   `Send` that reports the outcome is safer; an async variant with a bounded queue
   can come later if a producer needs it.
3. **Batching.** The gateway's stream is `format: json`, one object per request.
   An array format exists in the engine; switching the stream to it would let the
   client send many per request. Not needed at 1.45 events/s.
