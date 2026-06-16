# A5 — Auth Agent Brief

**Owns:** `services/auth` (Node/TS), Postgres `orgs`/`users`/`memberships` tables, JWT issuance, tenancy middleware for all services

## Current state (M1 complete)

`services/auth` **does not exist yet.** Tenancy is currently enforced via a raw `X-Tenant-ID` header (trusted, no auth). This agent replaces that with JWT-based auth.

## Architecture

```
Browser / SDK
    → POST /auth/login  →  services/auth  →  {token: <JWT>}
    → All API requests  →  Authorization: Bearer <JWT>

services/ingest  (Go)     → validate JWT, extract tenant_id
services/query   (Go)     → validate JWT, extract tenant_id
services/app-api (Node)   → validate JWT, extract tenant_id
services/realtime (Node)  → validate JWT on WS upgrade
```

JWT payload:

```json
{
  "sub": "<user_id>",
  "org_id": "<org_id>",   // = tenant_id
  "role": "admin",
  "exp": <unix_ts>
}
```

## Tasks

### 1. Scaffold `services/auth`

```
services/auth/
  package.json     # @cascade/auth, fastify, @fastify/jwt, pg, bcrypt
  tsconfig.json
  src/
    index.ts       # Fastify server
    db/
      pool.ts      # pg Pool (DATABASE_URL env)
    routes/
      auth.ts      # POST /auth/register, POST /auth/login, POST /auth/refresh
      orgs.ts      # POST /api/orgs, GET /api/orgs/:id
      users.ts     # GET /api/users/me, PATCH /api/users/me
    middleware/
      jwt.ts       # verifyJWT fastify preHandler (exported for other services)
```

**`package.json` dependencies:**

```json
{
  "fastify": "^4",
  "@fastify/jwt": "^8",
  "pg": "^8",
  "bcryptjs": "^2",
  "zod": "^3"
}
```

### 2. Auth endpoints

**`POST /auth/register`**

```json
// Request
{"email": "user@example.com", "password": "...", "name": "Alice", "org_name": "Acme"}
// Response
{"user": {...}, "org": {...}, "token": "<JWT>"}
```

- Creates org + user + owner membership in a single transaction.
- Hash password with `bcryptjs` (rounds: 12).
- Return signed JWT (exp: 7 days).

**`POST /auth/login`**

```json
// Request
{"email": "user@example.com", "password": "..."}
// Response
{"token": "<JWT>", "user": {...}}
```

**`POST /auth/refresh`** — accepts old valid token, returns new token (shorter TTL on old, new exp reset).

### 3. JWT secret

Read from `JWT_SECRET` environment variable. Must be ≥ 32 chars. Fail fast on startup if missing.

### 4. Go tenancy middleware

Add a shared JWT verifier for the Go services. Create `services/ingest/internal/tenant/jwt.go` (A5 may add this directly — it's A1's package but A5 owns the middleware):

```go
// JWTMiddleware validates Bearer token, extracts org_id → tenant_id.
// Falls back to X-Tenant-ID header only if ALLOW_HEADER_TENANT=true (dev only).
func JWTMiddleware(secret []byte) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // parse Authorization: Bearer <token>
            // extract org_id claim as tenant_id
            // store in context via tenant.WithTenantID(ctx, tenantID)
            // reject 401 if invalid
            next.ServeHTTP(w, r)
        })
    }
}
```

Both ingest and query services' `main.go` currently use `chi.Use(tenant.Middleware)` — replace with `chi.Use(JWTMiddleware(secret))`.

### 5. Node tenancy middleware (Fastify preHandler)

Export from `services/auth/src/middleware/jwt.ts` so app-api and realtime can import it:

```typescript
// packages/auth-client (or just local import path)
export async function requireAuth(request: FastifyRequest, reply: FastifyReply) {
  try {
    await request.jwtVerify();
    // request.user = { sub, org_id, role }
  } catch {
    reply.status(401).send({ error: 'unauthorized' });
  }
}
```

### 6. Add to docker-compose

```yaml
auth:
  build:
    context: ../services/auth
    dockerfile: Dockerfile
  container_name: cascade-auth
  ports:
    - '8083:8083'
  environment:
    PORT: '8083'
    DATABASE_URL: 'postgresql://cascade:cascade@postgres:5432/cascade'
    JWT_SECRET: 'cascade-dev-secret-change-in-production-min32chars'
  depends_on:
    postgres:
      condition: service_healthy
  healthcheck:
    test: ['CMD', 'curl', '-sf', 'http://localhost:8083/health']
    interval: 5s
    timeout: 5s
    retries: 10
    start_period: 10s
```

## Postgres tables (already migrated)

See `infra/migrations/postgres/001_init.sql`:

- `orgs` — `(id UUID, name, slug UNIQUE, created_at, updated_at)`
- `users` — `(id UUID, email UNIQUE, name, password_hash, created_at, updated_at)`
- `memberships` — `(org_id FK orgs, user_id FK users, role CHECK IN ('owner','admin','member'), created_at)` PK: `(org_id, user_id)`
- Index: `idx_memberships_user_id ON memberships(user_id)`

## Key constraints

- `tenant_id` === `org_id` throughout the system. Every API must scope data to `org_id` from the JWT — never trust a client-supplied tenant_id in M2+.
- The `X-Tenant-ID` bypass (`ALLOW_HEADER_TENANT=true`) must only be enabled in dev/test. Default: disabled.
- Password hashing MUST use bcrypt with ≥ 10 rounds (12 recommended).
- JWT secret must come from env, never hardcoded.

## Running locally (after implementation)

```bash
make up

# Register
curl -X POST http://localhost:8083/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@test.com","password":"test1234","name":"Test","org_name":"TestOrg"}'

# Login
TOKEN=$(curl -s -X POST http://localhost:8083/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@test.com","password":"test1234"}' | python3 -c "import sys,json;print(json.load(sys.stdin)['token'])")

# Use token
curl -H "Authorization: Bearer $TOKEN" "http://localhost:8081/v1/count?event=page_view&from=2026-01-01T00:00:00Z&to=2026-12-31T23:59:59Z"
```
