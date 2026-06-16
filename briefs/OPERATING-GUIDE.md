# Cascade Operating Guide

How to run development sessions on this repo.

## Repository layout

```
cascade/
├── apps/dashboard/        # React + Vite + Tailwind (A3)
├── services/
│   ├── ingest/            # Go 1.24, chi v5 (A1)
│   ├── writer/            # Go 1.24, clickhouse-go/v2 (A1)
│   ├── query/             # Go 1.24, chi v5 (A2)
│   ├── app-api/           # Node 22, Fastify — not yet scaffolded (A3)
│   ├── realtime/          # Node 22, Fastify + WS — not yet scaffolded (A4)
│   └── auth/              # Node 22, Fastify + JWT — not yet scaffolded (A5)
├── packages/
│   ├── types/             # @cascade/types (canonical TypeScript types)
│   └── sdk/               # @cascade/sdk (JS SDK)
├── infra/
│   ├── docker-compose.yml
│   ├── clickhouse/        # config.xml, users.d/cascade_profile.xml
│   └── migrations/
│       ├── clickhouse/    # 001_events, 002_events_counts, 003_mv
│       └── postgres/      # 001_init
├── contracts/
│   ├── ingest/openapi.yaml
│   ├── query/openapi.yaml
│   └── app/openapi.yaml
├── briefs/                # This directory — per-agent working briefs
├── scripts/
│   └── smoke-test.sh      # M1 end-to-end test
├── Makefile
├── CLAUDE.md              # Master conventions (read every session)
└── DECISIONS.md           # Architecture log
```

## Daily workflow

```bash
# Start everything
make up

# Verify M1 pipeline
make smoke

# Tail all logs
make logs

# Stop
make down

# Rebuild a single Go service after code changes
cd infra
docker compose build <service>
docker compose up -d --no-deps <service>

# Rebuild a Node/TS service
pnpm --filter @cascade/<package> build
cd infra && docker compose build <service> && docker compose up -d --no-deps <service>
```

## Starting a new agent session

Every Claude Code session should:
1. Read `CLAUDE.md` (conventions, ownership map, commit style).
2. Read your agent brief in `briefs/A<N>-<name>.md`.
3. Read `DECISIONS.md` (architecture log).
4. Run `make up` to confirm all services are healthy before making changes.
5. Run `make smoke` to confirm M1 is green before changing anything in the ingest/query path.

## Go services

Each Go service has its own `go.mod`. There is no Go workspace file.

```bash
# Run go mod tidy (requires Docker since Go is not installed locally):
docker run --rm -v $(PWD)/services/<name>:/app -w /app golang:1.24-alpine \
    sh -c "apk add git && go mod tidy"

# Build a service locally (no Docker):
cd services/<name>
go build ./...
go test ./...

# Add a dependency:
docker run --rm -v $(PWD)/services/<name>:/app -w /app golang:1.24-alpine \
    sh -c "apk add git && go get <module>@<version> && go mod tidy"
```

**Version constraints:**
- Use `golang:1.24-alpine` Docker image
- `go.mod` must say `go 1.24` (go-redis/v9 and clickhouse-go/v2 require it)
- Always `RUN apk add --no-cache git` in Dockerfile builder stage

**ClickHouse named parameter binding:**
```go
// WRONG — driver wraps time.Time in toDateTime(...) which CH rejects as a named param
clickhouse.Named("from", from)  // type hint {from:DateTime} or {from:DateTime64}

// CORRECT — bind as Unix int64, cast in SQL
clickhouse.Named("from", from.Unix())
// SQL: timestamp >= toDateTime64({from:Int64}, 3, 'UTC')
```

## Node services

All Node services use `pnpm` workspaces. Never use `npm` or `yarn`.

```bash
# Install all deps
pnpm install

# Add a dep to a specific service
pnpm --filter @cascade/<name> add <package>

# Build all packages
pnpm build

# Run tests
pnpm test

# Type-check all
pnpm typecheck
```

**Workspace packages:** `@cascade/types`, `@cascade/sdk`, `@cascade/dashboard`, and any new services added to `pnpm-workspace.yaml`.

## ClickHouse

```bash
# Connect to ClickHouse
docker exec -it cascade-clickhouse clickhouse-client \
    --user cascade --password cascade --database cascade

# Check events
SELECT count(), min(timestamp), max(timestamp) FROM events;

# Truncate for testing
TRUNCATE TABLE events;

# Check migration status
SHOW TABLES;
```

**Important:** `async_insert` is enabled (see `infra/clickhouse/users.d/cascade_profile.xml`). Data inserted by the writer may take up to 500ms to be visible in queries.

## Redis

```bash
# Connect
docker exec -it cascade-redis redis-cli

# Stream status
XLEN ingest:stream
XINFO GROUPS ingest:stream

# If stream/group was deleted accidentally:
XGROUP CREATE ingest:stream writer-group 0 MKSTREAM

# Flush stream (only for testing, NOT in production):
DEL ingest:stream
XGROUP CREATE ingest:stream writer-group 0 MKSTREAM
docker compose restart writer  # writer may need restart after group recreation
```

## Postgres

```bash
# Connect
docker exec -it cascade-postgres psql -U cascade -d cascade

# List tables
\dt

# Check orgs/users (M2)
SELECT * FROM orgs;
SELECT * FROM users;
```

## Smoke test

```bash
bash scripts/smoke-test.sh

# Custom tenant
TENANT_ID=my-tenant-id bash scripts/smoke-test.sh

# Custom endpoints (staging)
INGEST_URL=https://ingest.example.com QUERY_URL=https://query.example.com bash scripts/smoke-test.sh
```

The smoke test:
1. Checks health of ingest + query
2. Gets baseline count for `smoke_test` event
3. Sends a batch of 10 events
4. Polls count until it reaches baseline+10 (max 30s)
5. Sends a single capture and verifies event_id is returned

All timestamps are anchored to `NOW_EPOCH` at script start so events always land inside the query window.

## Troubleshooting

### ClickHouse won't start
Check for config errors: `docker logs cascade-clickhouse 2>&1 | grep -i error`
- Code 137 = user-level settings (`max_memory_usage`, `async_insert*`) placed in server config. Move them to `users.d/`.
- Code 450 = TTL on DateTime64 without `toDateTime()` cast. Use `TTL toDateTime(timestamp) + INTERVAL 1 YEAR`.

### Writer keeps crashing with NOGROUP
The Redis stream consumer group was deleted. Recreate it:
```bash
docker exec cascade-redis redis-cli XGROUP CREATE ingest:stream writer-group 0 MKSTREAM
docker compose restart writer
```

### Query returns 500
Check the store-level error: add `fmt.Printf("[DEBUG] error: %v\n", err)` to `CountEvents` temporarily.
Most likely cause: datetime binding issue — ensure time parameters are bound as `Int64` Unix seconds with `toDateTime64()` cast in SQL.

### Health check fails with curl exit 22 (405)
`wget --spider` sends a HEAD request; chi v5.3 returns 405 on GET-only routes. Use `curl -sf` in healthchecks.

### go mod tidy fails: "missing go.sum"
Build the container; `go mod tidy` runs inside Docker:
```bash
docker run --rm -v $(PWD)/services/<name>:/app -w /app golang:1.24-alpine \
    sh -c "apk add git && go mod tidy"
```

### Events in ClickHouse but count returns 0
Time window mismatch. Verify with a wide window:
```bash
curl -H "X-Tenant-ID: <id>" \
  "http://localhost:8081/v1/count?event=<name>&from=2026-01-01T00:00:00Z&to=2027-01-01T00:00:00Z"
```

## Commit style

```
type(scope): short imperative description
```
- Types: `feat | fix | refactor | test | chore | docs | infra`
- Scopes: `ingest | writer | query | dashboard | realtime | auth | platform | types | contracts`
- No co-author trailers. No ticket references in the subject line.

## Contract change process

1. Update the PRD section (§6 for schemas, §7 for API).
2. Add an entry to `DECISIONS.md`.
3. Update `packages/types/src/index.ts`.
4. Update the relevant OpenAPI spec in `contracts/`.
5. Update any Go mirror structs.
6. Notify agents that consume the changed contract.

Contracts are: `CaptureEvent`/`StoredEvent`, ClickHouse `events` DDL, Postgres schema, OpenAPI specs, tenancy rule.

## Agent ownership

| Agent | Owns | Brief |
|---|---|---|
| A1 | SDK, ingest, writer, CH write path | `briefs/A1-ingestion.md` |
| A2 | query service, CH read layer | `briefs/A2-query.md` |
| A3 | React dashboard, app-api, PG dashboards/widgets | `briefs/A3-dashboard.md` |
| A4 | realtime service, `rt:*` channels, WS client | `briefs/A4-realtime.md` |
| A5 | auth service, PG orgs/users/memberships, JWT | `briefs/A5-auth.md` |
| A6 | Docker, CI, OTEL, load tests, Makefile | `briefs/A6-platform.md` |
