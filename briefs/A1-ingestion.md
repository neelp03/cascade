# A1 — Ingestion Agent Brief

**Owns:** `services/ingest`, `services/writer`, `packages/sdk`, ClickHouse write path

## Current state (M1 complete)

Both services compile, pass healthchecks, and are proven by `make smoke`.

### Ingest service (`services/ingest`, port 8080)
- `POST /v1/batch` — accepts `{events: CaptureEvent[]}`, validates, generates UUIDv7 `event_id`, pushes to Redis Stream `ingest:stream` as JSON `{data: <StoredEvent JSON>}`.
- `POST /v1/capture` — single-event convenience endpoint; same pipeline.
- `GET /health` — returns `{"status":"ok"}`.
- Tenant: read from `X-Tenant-ID` header; reject (400) if missing.
- No authentication in M1 — A5 will add JWT middleware.

### Writer service (`services/writer`)
- Reads from `ingest:stream` consumer group `writer-group` via `XREADGROUP`.
- Batches up to `BATCH_SIZE` events or `FLUSH_INTERVAL_MS` ms, whichever comes first.
- Writes to ClickHouse table `events` using `clickhouse-go/v2` async insert.
- `XACK`s only after successful write (at-least-once delivery).
- Recovers from `NOGROUP` errors by calling `EnsureGroup` and retrying.

### SDK (`packages/sdk`)
- `CascadeSDK.init(config)` — sets endpoint, tenant, flushInterval, maxBatchSize.
- `capture(event, properties)` — buffers events.
- `flush()` — POSTs to `/v1/batch`.
- Auto-flush on interval and on `beforeunload`.
- `distinct_id` persisted to `localStorage`.

## Next tasks for A1

### 1. Structured logging (replace `fmt.Println`)
Both services use bare `fmt.Println`/`fmt.Fprintf`. Add `go.uber.org/zap` per CLAUDE.md conventions:
```
go get go.uber.org/zap
```
Replace all `fmt.Print*` with `zap.Logger` calls (JSON in production, console in dev based on `LOG_LEVEL`).

### 2. OTEL tracing stubs
Add `go.opentelemetry.io/otel` spans to ingest handler and writer consumer loop. Export to stdout (OTLP later, handled by A6).

### 3. Schema validation
The ingest handler currently accepts any JSON. Add validation:
- `event` field required, non-empty string.
- `timestamp` must parse as RFC3339; if absent, default to server time.
- `distinct_id` required.
- `properties` values: stringify non-string types rather than rejecting.

### 4. Dead-letter handling in writer
If a message fails to parse (malformed JSON), currently it is silently skipped. Log it and write to a `ingest:dlq` stream instead.

### 5. SDK improvements
- Add `identify(distinctId, traits)` — captures `$identify` event.
- Add `page()` / `screen()` helpers.
- Add retry logic on batch POST failure (exponential backoff, max 3).
- Ship as an npm package (`packages/sdk/package.json` already scaffolded).

## Key files

| File | Purpose |
|---|---|
| `services/ingest/cmd/main.go` | Entry point, env config |
| `services/ingest/internal/handler/ingest.go` | HTTP handlers |
| `services/ingest/internal/schema/event.go` | StoredEvent struct |
| `services/ingest/internal/tenant/tenant.go` | X-Tenant-ID extraction |
| `services/writer/cmd/main.go` | Entry point |
| `services/writer/internal/consumer/stream.go` | Redis consumer loop |
| `services/writer/internal/writer/clickhouse.go` | ClickHouse batch writer |
| `packages/sdk/src/index.ts` | JS SDK |
| `infra/migrations/clickhouse/001_events.sql` | events table DDL |

## Contracts (frozen — do not change without updating PRD §6 + DECISIONS.md)

- `CaptureEvent` / `StoredEvent` in `packages/types/src/index.ts`
- ClickHouse `events` table schema in `infra/migrations/clickhouse/001_events.sql`
- OpenAPI spec: `contracts/ingest/openapi.yaml`

## Running locally

```bash
make up          # starts all services
docker logs -f cascade-ingest
docker logs -f cascade-writer
make smoke       # end-to-end test
```

## ClickHouse write notes

- clickhouse-go/v2 named parameters: pass `time.Time` as `Int64` Unix seconds and cast in SQL with `toDateTime64({ts:Int64}, 3, 'UTC')` — the driver wraps `time.Time` in `toDateTime(...)` which CH cannot parse as a named-parameter literal.
- Async insert is enabled in `infra/clickhouse/users.d/cascade_profile.xml`. Data is visible within ~500ms of insert.
