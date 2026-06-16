# Cascade — Agent Conventions & Context

> Every Claude Code session in this repo should read this file first.
> The PRD (`PRD-cascade-analytics.md` at repo root) is the product source of truth.
> `DECISIONS.md` logs architecture decisions and contract changes.

## What this is

Cascade is a self-hostable, open-source product analytics platform (PostHog/Amplitude alternative). Events flow in via a JS SDK, are stored in ClickHouse, and surface in live-updating collaborative dashboards.

## Repo layout

```
cascade/
├── apps/
│   └── dashboard/        # React + TypeScript + Tailwind + Recharts (A3)
├── services/
│   ├── ingest/           # Go — HTTP ingest API, enqueues to Redis Stream (A1)
│   ├── writer/           # Go — Redis Stream consumer, batch-writes to ClickHouse (A1)
│   ├── query/            # Go — ClickHouse query API, Redis result cache (A2)
│   ├── app-api/          # Node/TS — Dashboard CRUD, Postgres (A3)
│   ├── realtime/         # Node/TS — WebSocket server, Redis pub/sub (A4)
│   └── auth/             # Node/TS — Sessions/JWT, org/team model, tenancy (A5)
├── packages/
│   └── types/            # @cascade/types — shared TS types (canonical schema)
├── contracts/
│   ├── ingest/           # OpenAPI spec for the ingest API
│   ├── query/            # OpenAPI spec for the query API (highest-traffic)
│   └── app/              # OpenAPI spec for the app/dashboard API
├── infra/
│   ├── docker-compose.yml
│   ├── clickhouse/       # ClickHouse server config
│   └── migrations/
│       ├── clickhouse/   # DDL run at ClickHouse startup
│       └── postgres/     # DDL run at Postgres startup
├── briefs/               # Per-agent working briefs (read yours before starting)
├── scripts/              # smoke-test.sh, seed.sh
├── Makefile              # make help — lists all targets
└── DECISIONS.md          # architecture log
```

## Tech stack

| Concern          | Choice                                                      |
| ---------------- | ----------------------------------------------------------- |
| Go services      | Go 1.23; chi v5 router; zap logging; OTEL tracing           |
| Node services    | Node 22 LTS; TypeScript; Fastify                            |
| Frontend         | React 18 + TypeScript + Tailwind CSS + Recharts             |
| Analytical store | ClickHouse 24.8                                             |
| Metadata store   | PostgreSQL 16                                               |
| Queue / pub-sub  | Redis Streams (`ingest:stream`) + pub/sub (`rt:*` channels) |
| Package manager  | pnpm 9 workspaces                                           |
| Containers       | Docker Compose (infra/docker-compose.yml)                   |

## Running locally

```bash
# Prerequisites: Docker, Node 22, pnpm 9, Go 1.23
make install      # pnpm install
make up           # docker compose up --build (all deps + services)
make smoke        # run the M1 smoke test
make down         # stop everything
make logs         # tail logs
```

## Frozen contracts — DO NOT change without updating PRD §6 + DECISIONS.md

1. **Event schema** — `packages/types/src/index.ts` (`CaptureEvent` / `StoredEvent`).
   Go mirror: `services/ingest/internal/schema/event.go`.
2. **ClickHouse `events` table** — `infra/migrations/clickhouse/001_events.sql`.
3. **Postgres schemas** — `infra/migrations/postgres/001_init.sql`.
4. **OpenAPI stubs** — `contracts/{ingest,query,app}/openapi.yaml`.
5. **Tenancy rule** — every DB/CH query MUST be scoped by `tenant_id`. Use the shared helper from `services/auth` (Node) or `services/ingest/internal/tenant` (Go).

If you need to change a contract, update PRD §6 first, log in `DECISIONS.md`, then update the artifact.

## Code conventions

### Go

- `internal/` for non-exported packages; `cmd/` for `main.go`.
- Structured logging via `go.uber.org/zap`; never `fmt.Print` in production paths.
- All errors wrapped with context: `fmt.Errorf("doing X: %w", err)`.
- HTTP handlers return JSON; errors as `{"error": "message"}` with appropriate status.
- Every handler is traced with OTEL spans.

### TypeScript / Node

- Strict mode (`"strict": true` in tsconfig).
- No `any` without a comment explaining why.
- Prefer `async/await` over callback chains.
- Fastify for HTTP in Node services; Vite for the frontend.

### React

- Functional components + hooks only.
- Tailwind for all styling — no inline styles, no CSS modules.
- Recharts for charts.
- `@cascade/types` is the single source of truth for shared types.

## Tenancy enforcement (critical)

Every Postgres query: `WHERE tenant_id = $1` (parameterized).
Every ClickHouse query: `AND tenant_id = {tenant_id:String}`.
The middleware that injects `tenant_id` into request context lives in `services/auth`.
In M1, the ingest API reads `X-Tenant-ID` header and rejects requests without one.

## Agent ownership (do not reach into another agent's tables/packages)

| Agent        | Owns                                                              |
| ------------ | ----------------------------------------------------------------- |
| A1 Ingestion | SDK, ingest service, writer service, CH `events` write path       |
| A2 Query     | query service, CH read layer                                      |
| A3 Dashboard | React app, app-api service, PG `dashboards`/`widgets`/`queries`   |
| A4 Realtime  | realtime service, `rt:*` channels, client WS layer                |
| A5 Auth      | auth service, PG `orgs`/`users`/`memberships`, tenancy middleware |
| A6 Platform  | Docker builds, observability, CI/CD, load-test harness            |

## Commit style

```
type(scope): short imperative description

Types: feat | fix | refactor | test | chore | docs | infra
Scope: ingest | writer | query | dashboard | realtime | auth | platform | types | contracts
```

Examples:

```
feat(ingest): add batch endpoint for JS SDK
fix(writer): handle Redis XACK failure on writer crash
chore(infra): add ClickHouse healthcheck to docker-compose
```

No co-author trailers. Keep commits small and reviewable.

## Adding a new service

1. Create `services/<name>/` with its own `go.mod` (Go) or `package.json` (Node).
2. Add it to `pnpm-workspace.yaml` (Node only).
3. Add a build + test step to `.github/workflows/ci.yml`.
4. Add the service to `infra/docker-compose.yml` with a healthcheck.
5. Log the decision in `DECISIONS.md`.
