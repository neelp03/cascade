# A3 — Dashboard Agent Brief

**Owns:** `apps/dashboard` (React), `services/app-api` (Node/TS, not yet scaffolded)

## Current state (M1 complete)

A minimal React dashboard exists and is served via nginx on port 3000. It shows a hardcoded count widget that queries the query service directly from the browser.

### Dashboard app (`apps/dashboard`, port 3000 in Docker)
- **Tech:** React 18 + TypeScript + Vite + Tailwind + Recharts
- **Structure:**
  - `src/App.tsx` — root component, hardcoded demo layout
  - `src/api/` — API client wrappers for ingest and query
  - `src/components/` — shared UI components
  - `src/hooks/` — data-fetching hooks
- **Build:** `pnpm build` → `dist/` → served by nginx in Docker
- **Dev:** `pnpm dev` → Vite HMR on port 5173

### What the M1 dashboard does
- Renders a single numeric "count" widget polling `/v1/count`.
- Hardcoded tenant ID, event name, and time range.
- No persistence (no app-api, no Postgres dashboards/widgets).

## Next tasks for A3

### 1. Scaffold `services/app-api` (Node 22 + Fastify + Postgres)
This service does not exist yet. Create:
```
services/app-api/
  package.json        # @cascade/app-api, fastify, pg, @cascade/types
  tsconfig.json
  src/
    index.ts          # Fastify app + routes
    db/
      pool.ts         # pg Pool (DATABASE_URL env)
    routes/
      dashboards.ts   # CRUD for dashboards
      widgets.ts      # CRUD for widgets
      queries.ts      # saved query configs
```
Postgres DDL is in `infra/migrations/postgres/001_init.sql` (tables: `orgs`, `users`, `memberships`, `dashboards`, `widgets`, `saved_queries`).

### 2. Dashboard CRUD UI
Replace the hardcoded demo with real data from app-api:
- List dashboards (`GET /api/dashboards`)
- Create dashboard (`POST /api/dashboards`)
- Add widgets to a dashboard (count, timeseries, funnel)
- Delete widget / dashboard

### 3. Dynamic count widget
- Widget config: `{type: "count", event: string, from: string, to: string}`
- Query: `GET /v1/count?event=&from=&to=` via query service
- Auto-refresh: poll every 30s

### 4. Time-series chart widget
- Widget config: `{type: "timeseries", event: string, interval: "hour"|"day", lookback_days: number}`
- Uses Recharts `<LineChart>` or `<AreaChart>`
- Query: `GET /v1/timeseries`

### 5. Funnel widget (stub)
- Widget config: `{type: "funnel", steps: string[]}`
- Renders a simple bar chart of conversion rates per step
- Query: `GET /v1/funnel` (A2 must implement first)

### 6. Real-time numbers via WebSocket
Once A4 (realtime) is ready:
- Connect to `ws://localhost:8082` on dashboard load
- Subscribe to `{type: "subscribe", tenant_id, event}` 
- On `{type: "count_update", count}` message, update widget value without polling

### 7. Collaborative editing (future, A4)
Dashboard layout changes broadcast via realtime service; Last-Write-Wins (LWW) per DECISIONS.md.

## Key files

| File | Purpose |
|---|---|
| `apps/dashboard/src/App.tsx` | Root component |
| `apps/dashboard/src/api/` | HTTP client wrappers |
| `apps/dashboard/src/components/` | UI components |
| `apps/dashboard/Dockerfile` | Multi-stage: pnpm build → nginx |
| `apps/dashboard/nginx.conf` | Static file serving config |
| `infra/migrations/postgres/001_init.sql` | Dashboard/widget schema |
| `contracts/app/openapi.yaml` | app-api OpenAPI spec (stub) |

## Contracts (frozen)

- `Widget`, `Dashboard`, `Org`, `User`, `Membership` in `packages/types/src/index.ts`
- OpenAPI: `contracts/app/openapi.yaml`

## Environment variables (dashboard Dockerfile)

| Var | Purpose |
|---|---|
| `VITE_INGEST_URL` | Ingest service URL (browser-visible) |
| `VITE_QUERY_URL` | Query service URL (browser-visible) |
| `VITE_APP_URL` | app-api URL (browser-visible) |

## Running locally

```bash
make up
# Dashboard at http://localhost:3000

# Dev mode (hot reload, no Docker):
cd apps/dashboard
pnpm dev   # → http://localhost:5173
```

## Postgres DDL reference

See `infra/migrations/postgres/001_init.sql`. Key tables:
- `orgs(id UUID, name TEXT, slug TEXT UNIQUE, created_at)`
- `users(id UUID, email TEXT UNIQUE, password_hash TEXT, created_at)`
- `memberships(org_id, user_id, role TEXT)` — role: owner | admin | member | viewer
- `dashboards(id UUID, org_id, name TEXT, layout JSONB, created_by, created_at)`
- `widgets(id UUID, dashboard_id, type TEXT, config JSONB, position JSONB)`
- `saved_queries(id UUID, org_id, name TEXT, query_config JSONB)`
