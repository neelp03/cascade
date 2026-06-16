#!/usr/bin/env bash
# M1 smoke test — proves the full pipeline end to end.
# Usage: bash scripts/smoke-test.sh
# Requires: curl, jq; all services running (make up)

set -euo pipefail

TENANT_ID="${TENANT_ID:-00000000-0000-0000-0000-000000000001}"
INGEST_URL="${INGEST_URL:-http://localhost:8080}"
QUERY_URL="${QUERY_URL:-http://localhost:8081}"
EVENT_NAME="smoke_test"
N_EVENTS=10
MAX_WAIT=30  # seconds to wait for writer to flush to ClickHouse

GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

pass() { echo -e "${GREEN}✓${NC} $1"; }
fail() { echo -e "${RED}✗${NC} $1"; exit 1; }

echo "==> Cascade M1 smoke test"
echo "    Ingest: $INGEST_URL"
echo "    Query:  $QUERY_URL"
echo "    Tenant: $TENANT_ID"
echo ""

# ── 1. Health checks ────────────────────────────────────────────────────────
echo "--- Health checks"

INGEST_HEALTH=$(curl -sf "$INGEST_URL/health" | jq -r '.status')
[ "$INGEST_HEALTH" = "ok" ] && pass "ingest healthy" || fail "ingest unhealthy"

QUERY_HEALTH=$(curl -sf "$QUERY_URL/health" | jq -r '.status')
[ "$QUERY_HEALTH" = "ok" ] && pass "query healthy" || fail "query unhealthy"

# ── 2. Get baseline count ────────────────────────────────────────────────────
echo ""
echo "--- Baseline count"
FROM=$(date -u -d "1 hour ago" +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null \
     || date -u -v-1H +"%Y-%m-%dT%H:%M:%SZ")
TO=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

BASELINE=$(curl -sf \
  -H "X-Tenant-ID: $TENANT_ID" \
  "$QUERY_URL/v1/count?event=$EVENT_NAME&from=$FROM&to=$TO" \
  | jq -r '.count')
pass "baseline count = $BASELINE"

# ── 3. Send batch of events ──────────────────────────────────────────────────
echo ""
echo "--- Sending $N_EVENTS events"
TS=$(date -u +"%Y-%m-%dT%H:%M:%S.000Z")

EVENTS_JSON=$(python3 -c "
import json, sys
events = [
    {
        'event': '$EVENT_NAME',
        'timestamp': '$TS',
        'distinct_id': f'smoke-user-{i}',
        'properties': {'run': 'smoke', 'index': i}
    }
    for i in range($N_EVENTS)
]
print(json.dumps({'events': events}))
")

BATCH_RESP=$(curl -sf -X POST \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d "$EVENTS_JSON" \
  "$INGEST_URL/v1/batch")

ACCEPTED=$(echo "$BATCH_RESP" | jq -r '.accepted')
[ "$ACCEPTED" = "$N_EVENTS" ] && pass "batch accepted $ACCEPTED/$N_EVENTS" \
  || fail "batch only accepted $ACCEPTED/$N_EVENTS"

# ── 4. Wait for writer to flush to ClickHouse ───────────────────────────────
echo ""
echo "--- Waiting for writer flush (max ${MAX_WAIT}s)"
EXPECTED=$((BASELINE + N_EVENTS))
ACTUAL=$BASELINE
ELAPSED=0

while [ "$ACTUAL" -lt "$EXPECTED" ] && [ "$ELAPSED" -lt "$MAX_WAIT" ]; do
  sleep 2
  ELAPSED=$((ELAPSED + 2))
  ACTUAL=$(curl -sf \
    -H "X-Tenant-ID: $TENANT_ID" \
    "$QUERY_URL/v1/count?event=$EVENT_NAME&from=$FROM&to=$TO" \
    | jq -r '.count')
  echo "    count = $ACTUAL / $EXPECTED (${ELAPSED}s elapsed)"
done

[ "$ACTUAL" -ge "$EXPECTED" ] \
  && pass "count reached $ACTUAL (expected >= $EXPECTED) in ${ELAPSED}s" \
  || fail "count $ACTUAL < $EXPECTED after ${ELAPSED}s — writer may not be flushing"

# ── 5. Verify single event capture ──────────────────────────────────────────
echo ""
echo "--- Single event capture"
SINGLE_RESP=$(curl -sf -X POST \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d "{\"event\":\"${EVENT_NAME}_single\",\"timestamp\":\"$TS\",\"distinct_id\":\"smoke-single\"}" \
  "$INGEST_URL/v1/capture")

EVENT_ID=$(echo "$SINGLE_RESP" | jq -r '.event_id')
[ -n "$EVENT_ID" ] && [ "$EVENT_ID" != "null" ] \
  && pass "single capture returned event_id: $EVENT_ID" \
  || fail "single capture did not return event_id"

echo ""
echo -e "${GREEN}==> M1 smoke test PASSED${NC}"
