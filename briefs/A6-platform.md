# A6 — Platform Agent Brief

**Owns:** Docker builds, observability (OTEL), CI/CD, load-test harness, `Makefile` targets, `infra/`

## Current state (M1 complete)

All Docker images build and run. Core infra (ClickHouse, Postgres, Redis) is healthy. CI does not exist yet. No OTEL export configured. No load test harness.

## Key infra facts

### docker-compose (infra/docker-compose.yml)
- No `version:` field (removed — deprecated warning in Docker Compose v2).
- Healthchecks use `curl -sf` for Go services (not `wget --spider` — chi v5.3 returns 405 for HEAD on GET routes).
- Go service images: `golang:1.24-alpine` builder + `alpine:3.20` runtime. Both ingest and query runtime images include `curl` for healthchecks.
- All Go services need `RUN apk add --no-cache git` in the builder stage (go mod download requires git for private modules).

### ClickHouse config
- Server config: `infra/clickhouse/config.xml` — sets `listen_host: 0.0.0.0` and `skip_check_for_incorrect_settings: 1`.
- User-level settings (async_insert, max_memory_usage) are in `infra/clickhouse/users.d/cascade_profile.xml` mounted as a single file (NOT a directory — ClickHouse entrypoint writes `default-user.xml` into `users.d/`).
- Migration files in `infra/migrations/clickhouse/001_events.sql`, `002_events_counts.sql`, `003_mv.sql` — processed in lexicographic order by the ClickHouse entrypoint.

### Go module layout
Each Go service has its own `go.mod` (no Go workspace). Modules are independent.

### Known version constraints
- `go-redis/v9 v9.20.1` requires Go ≥ 1.24
- `clickhouse-go/v2 v2.46.0` requires Go ≥ 1.24.1
- All Go images and `go.mod` files must specify `go 1.24` or higher.
- `.tool-versions`: `golang 1.24.4`

## Tasks

### 1. GitHub Actions CI workflow

Create `.github/workflows/ci.yml`:

```yaml
name: CI
on: [push, pull_request]

jobs:
  build-go:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        service: [ingest, writer, query]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: {go-version: '1.24'}
      - run: go build ./...
        working-directory: services/${{ matrix.service }}
      - run: go test ./...
        working-directory: services/${{ matrix.service }}

  build-node:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: pnpm/action-setup@v3
        with: {version: '9'}
      - uses: actions/setup-node@v4
        with: {node-version: '22', cache: 'pnpm'}
      - run: pnpm install --frozen-lockfile
      - run: pnpm typecheck
      - run: pnpm build

  docker-build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: docker compose -f infra/docker-compose.yml build

  smoke:
    runs-on: ubuntu-latest
    needs: docker-build
    steps:
      - uses: actions/checkout@v4
      - run: docker compose -f infra/docker-compose.yml up -d
      - run: sleep 30  # wait for ClickHouse startup
      - run: bash scripts/smoke-test.sh
      - run: docker compose -f infra/docker-compose.yml down -v
```

### 2. OTEL tracing

A1/A2 add spans; A6 wires the exporter:

Add to each Go service's `main.go`:
```go
import "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
// In dev: stdout exporter. In prod: OTLP exporter pointed at Jaeger/Tempo.
```

Add `OTEL_EXPORTER_OTLP_ENDPOINT` to docker-compose (optional).

Optional: add Jaeger to docker-compose for local tracing UI.

### 3. Structured logging

Ensure all services (Go + Node) log to stdout as JSON. Add log aggregation to docker-compose (e.g., Loki + Grafana, or just rely on `docker compose logs`).

### 4. Load-test harness

Create `scripts/load-test.sh` using `k6` or `wrk`:

```bash
# k6 load test: 100 VUs, 30s, batch POST
k6 run scripts/k6-ingest.js
```

Or create `scripts/k6-ingest.js`:
```javascript
import http from 'k6/http';
export const options = { vus: 100, duration: '30s' };
export default function() {
  http.post('http://localhost:8080/v1/batch', JSON.stringify({
    events: [{event: 'load_test', distinct_id: `user-${__VU}`, timestamp: new Date().toISOString()}]
  }), { headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': 'load-test-org' }});
}
```

### 5. Makefile targets

Current targets: `install`, `up`, `down`, `logs`, `smoke`, `build`.

Add:
```makefile
.PHONY: test lint load-test

test:
    @for svc in ingest writer query; do \
        echo "Testing $$svc..."; \
        docker run --rm -v $(PWD)/services/$$svc:/app -w /app golang:1.24-alpine go test ./...; \
    done
    pnpm test

lint:
    @for svc in ingest writer query; do \
        docker run --rm -v $(PWD)/services/$$svc:/app -w /app golang:1.24-alpine golangci-lint run; \
    done
    pnpm lint

load-test:
    k6 run scripts/k6-ingest.js

reset-data:
    docker exec cascade-clickhouse clickhouse-client \
        --user cascade --password cascade --database cascade \
        --query "TRUNCATE TABLE events"
    docker exec cascade-redis redis-cli FLUSHDB
```

### 6. Secrets management

For production: document in `infra/README.md` that these env vars must be overridden:
- `CLICKHOUSE_PASSWORD`, `POSTGRES_PASSWORD`, `JWT_SECRET`
- Never commit production secrets. Use `.env.production` (in `.gitignore`) or a secrets manager.

### 7. Docker image optimisation

Current Go images are multi-stage but not layer-cached efficiently. After each agent ships their service, ensure:
- `go.mod` + `go.sum` are copied before source (layer cache for `go mod download`).
- Build args for version tags: `ARG VERSION=dev; -ldflags="-X main.Version=${VERSION}"`.

## Running infra locally

```bash
make up           # start everything
make logs         # tail all logs
make down         # stop everything
docker compose -f infra/docker-compose.yml ps  # check health

# Force rebuild a single service:
cd infra && docker compose build <service> && docker compose up -d --no-deps <service>
```

## Current docker-compose services and ports

| Service | Port | Health endpoint |
|---|---|---|
| clickhouse | 8123 (HTTP), 9000 (native) | `GET /ping` |
| postgres | 5432 | `pg_isready` |
| redis | 6379 | `redis-cli ping` |
| ingest | 8080 | `GET /health` |
| writer | — (no HTTP) | — |
| query | 8081 | `GET /health` |
| dashboard | 3000 | — (nginx) |
