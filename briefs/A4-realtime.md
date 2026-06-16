# A4 — Realtime Agent Brief

**Owns:** `services/realtime` (Node/TS WebSocket server), `rt:*` Redis pub/sub channels, client WebSocket layer in dashboard

## Current state (M1 complete)

`services/realtime` **does not exist yet.** The M1 slice uses polling for count queries. This agent builds the realtime layer.

## Architecture

```
Browser WebSocket client
    ↕ ws://localhost:8082
services/realtime (Node 22 + Fastify + @fastify/websocket)
    ↕ Redis pub/sub (SUBSCRIBE rt:<tenant_id>:<event>)
services/writer (publishes on every ClickHouse flush)
```

Each time the writer successfully flushes a batch to ClickHouse, it publishes a `{event, tenant_id, delta}` message to channel `rt:<tenant_id>:<event>`. The realtime service fans this out to all subscribed WebSocket clients for that tenant+event.

## Tasks

### 1. Scaffold `services/realtime`

```
services/realtime/
  package.json     # @cascade/realtime, fastify, @fastify/websocket, ioredis
  tsconfig.json
  src/
    index.ts       # Fastify server with /ws WebSocket route
    pubsub.ts      # Redis subscriber (one connection per tenant+event pair)
    session.ts     # WebSocket session management
```

**`package.json` dependencies:**
```json
{
  "fastify": "^4",
  "@fastify/websocket": "^8",
  "ioredis": "^5"
}
```

### 2. WebSocket protocol

Client → server (JSON over WS):
```jsonc
{"type": "subscribe",   "tenant_id": "...", "event": "page_view"}
{"type": "unsubscribe", "tenant_id": "...", "event": "page_view"}
{"type": "ping"}
```

Server → client (JSON over WS):
```jsonc
{"type": "count_update", "tenant_id": "...", "event": "page_view", "delta": 5}
{"type": "pong"}
{"type": "error", "message": "..."}
```

### 3. Redis pub/sub channels

Channel name pattern: `rt:<tenant_id>:<event>`

Message payload (JSON string):
```json
{"tenant_id": "...", "event": "page_view", "delta": 10}
```

The realtime service subscribes to channels dynamically as clients subscribe. Use one Redis connection per subscribed channel (or a pattern subscription `rt:*` and route by channel).

### 4. Writer integration

Modify `services/writer/internal/writer/clickhouse.go` to publish a pub/sub message after each successful batch write. This is a **cross-agent dependency** — coordinate with A1 or make the change yourself (A4 owns the `rt:*` channels per CLAUDE.md).

In writer, after `batch.Send()`:
```go
// After successful ClickHouse write:
for event, count := range batchCountByEvent {
    payload, _ := json.Marshal(map[string]any{
        "tenant_id": tenantID,
        "event":     event,
        "delta":     count,
    })
    rdb.Publish(ctx, fmt.Sprintf("rt:%s:%s", tenantID, event), payload)
}
```

### 5. Dashboard WebSocket hook

In `apps/dashboard/src/hooks/useRealtimeCount.ts`:
```typescript
export function useRealtimeCount(tenantId: string, event: string, initial: number) {
  const [count, setCount] = useState(initial);
  useEffect(() => {
    const ws = new WebSocket(import.meta.env.VITE_REALTIME_URL ?? 'ws://localhost:8082/ws');
    ws.onopen = () => ws.send(JSON.stringify({type:'subscribe', tenant_id: tenantId, event}));
    ws.onmessage = (e) => {
      const msg = JSON.parse(e.data);
      if (msg.type === 'count_update') setCount(c => c + msg.delta);
    };
    return () => ws.close();
  }, [tenantId, event]);
  return count;
}
```

### 6. Add to docker-compose

```yaml
realtime:
  build:
    context: ../services/realtime
    dockerfile: Dockerfile
  container_name: cascade-realtime
  ports:
    - "8082:8082"
  environment:
    PORT: "8082"
    REDIS_URL: "redis://redis:6379"
  depends_on:
    redis:
      condition: service_healthy
  healthcheck:
    test: ["CMD", "curl", "-sf", "http://localhost:8082/health"]
    interval: 5s
    timeout: 5s
    retries: 10
    start_period: 10s
```

## Key constraints

- Every WebSocket message MUST include `tenant_id`. The realtime service must verify the client only subscribes to channels matching their tenant.
- In M1 there is no auth — tenant_id is trusted from the client. A5 (auth) will add JWT validation; design the session object to accept a `tenantId` extracted from JWT later.
- Use `pnpm` to manage dependencies — `pnpm-workspace.yaml` lists all Node services.

## Running locally

```bash
make up   # after A4 is complete, realtime will be part of the stack
docker logs -f cascade-realtime

# Test with wscat:
npm install -g wscat
wscat -c ws://localhost:8082/ws
> {"type":"subscribe","tenant_id":"00000000-0000-0000-0000-000000000001","event":"smoke_test"}
# Should receive count_update messages as events are ingested
```
