# Architecture Decisions

> Log every contract change and significant architecture decision here.
> Format: `## [YYYY-MM-DD] Title` — rationale and any impacted contracts.

---

## [2026-06-15] Redis Streams over Kafka for v1

**Decision:** Use Redis Streams as the ingestion buffer and pub/sub backbone for v1.

**Rationale:** Reduces operational complexity (one less dependency to run/configure). Redis is already required for query caching. At ~10k events/sec, a single Redis node is well within capacity. Kafka is the documented upgrade path when horizontal scale demands it.

**Impacted contracts:** §6.2 ownership map (`ingest:stream`, `rt:*` channels in Redis).

---

## [2026-06-15] Go for hot paths, Node for app/realtime

**Decision:** Go handles ingest and query (CPU/IO hot paths); Node/TypeScript handles app API, realtime, and auth (richer ecosystem for WebSockets, JWT, ORM).

**Rationale:** Go's goroutines and low-overhead HTTP server are better suited for sustained 10k events/sec with sub-millisecond handler latency. Node is adequate for dashboard CRUD and WebSocket fan-out where throughput is lower.

---

## [2026-06-15] pnpm workspaces — TS services only

**Decision:** Go services use independent `go.mod` files (not a Go workspace) to keep modules self-contained. Node/TS services and packages use pnpm workspaces.

**Rationale:** Go workspaces add complexity without benefit when services share no Go packages. Shared types are TS-only (`@cascade/types`); Go services each define their own local event struct that mirrors the canonical TS type.

---

## [2026-06-15] Tenancy via X-Tenant-ID header in M1, hardened by A5

**Decision:** In M1, the ingest and query APIs read `X-Tenant-ID` from the request header and reject requests that omit it. A5 (Auth) will replace this with a JWT-derived tenant context in M4.

**Rationale:** Unblocks M1 without requiring the full auth service. The header-based stub is explicit and easy to search/replace when A5 lands.

---

## [2026-06-15] Last-write-wins for collaborative editing in v1

**Decision:** Collaborative dashboard editing uses last-write-wins semantics in v1. The upgrade path to CRDT is documented but not implemented.

**Rationale:** LWW is sufficient for the demo scope and dramatically simpler to implement. CRDT (e.g., Yjs) is the natural upgrade when simultaneous conflicting edits become a real user problem.

---

## [2026-06-15] UUIDv7 for event_id

**Decision:** `event_id` is a UUIDv7 (time-ordered UUID). Generated server-side by the ingest service.

**Rationale:** Time-ordered UUIDs sort naturally by insertion time, which improves ClickHouse range scan performance on the primary key. Client-generated IDs are not trusted to avoid injection of crafted timestamps.
