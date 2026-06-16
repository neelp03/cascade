# A2 — Query Agent Brief

**Owns:** `services/query`, ClickHouse read layer, Redis result cache

## Current state (M1 complete)

Query service compiles, passes healthcheck, and is proven by `make smoke`.

### Query service (`services/query`, port 8081)
- `GET /v1/count?event=&from=&to=` — returns `{"count": N}` for an event in a time window.
- `GET /v1/timeseries?event=&from=&to=&interval=hour|minute|day` — returns `{"series": [{timestamp, count}]}`.
- `GET /health` — returns `{"status":"ok"}`.
- Tenant: read from `X-Tenant-ID` header; inject into all ClickHouse queries.

### ClickHouse binding fix (critical knowledge)
`clickhouse-go/v2` serialises `time.Time` in named parameters as `toDateTime('...')`. ClickHouse cannot parse this string as a `DateTime` or `DateTime64` literal. **Always bind time parameters as `Int64` Unix seconds and cast in SQL:**
```go
clickhouse.Named("from", from.Unix())
// SQL: timestamp >= toDateTime64({from:Int64}, 3, 'UTC')
```

### Redis result cache
Currently **not implemented** — the `QUERY_CACHE_TTL_SECONDS` env var is wired but the handler doesn't cache yet. This is the first M2 task.

## Next tasks for A2

### 1. Redis result cache
Add a cache layer in `services/query/internal/cache/redis.go`:
- Key: `query:count:<tenant_id>:<event>:<from_unix>:<to_unix>`
- TTL: `QUERY_CACHE_TTL_SECONDS` (default 30).
- On hit: return cached JSON directly.
- On miss: run ClickHouse query, store result, return.

Use `github.com/redis/go-redis/v9` (already in `go.mod`).

### 2. Structured logging
Add `go.uber.org/zap`. Log tenant_id, event name, and query latency on each request.

### 3. OTEL tracing
Add spans for HTTP handler and ClickHouse query.

### 4. Funnel query endpoint
`GET /v1/funnel?steps[]=<event>&from=&to=` — sequential funnel conversion. Uses ClickHouse `windowFunnel()`.
Type contract: `FunnelRequest` / `FunnelResponse` in `packages/types/src/index.ts`.

### 5. Property filter support
Extend count and timeseries to accept `filter[prop]=value` query params. Map to `has(properties, 'key')` / `properties['key'] = 'value'` in ClickHouse.

### 6. Pagination for large timeseries
Add `limit` and `offset` params to `/v1/timeseries`.

### 7. Query timeout
Add per-query deadline: `ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)`.

## Key files

| File | Purpose |
|---|---|
| `services/query/cmd/main.go` | Entry point, env config |
| `services/query/internal/handler/query.go` | HTTP handlers |
| `services/query/internal/store/clickhouse.go` | ClickHouse query layer |
| `services/query/internal/tenant/tenant.go` | Tenant context extraction |

## Contracts (frozen)

- OpenAPI spec: `contracts/query/openapi.yaml`
- `TimeSeriesRequest/Response`, `CountRequest/Response`, `FunnelRequest/Response` in `packages/types/src/index.ts`
- ClickHouse `events` table schema (read-only for A2)

## Environment variables

| Var | Default | Purpose |
|---|---|---|
| `PORT` | `8081` | HTTP listen port |
| `CLICKHOUSE_DSN` | `clickhouse://cascade:cascade@clickhouse:9000/cascade` | Native TCP DSN |
| `REDIS_URL` | `redis://redis:6379` | Cache backend |
| `QUERY_CACHE_TTL_SECONDS` | `30` | Result cache TTL |

## Running locally

```bash
make up
docker logs -f cascade-query
curl -H "X-Tenant-ID: test" "http://localhost:8081/v1/count?event=page_view&from=2026-01-01T00:00:00Z&to=2026-12-31T23:59:59Z"
```
