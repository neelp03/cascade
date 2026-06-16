#!/usr/bin/env bash
# M1 smoke test — proves the full pipeline end to end.
# Requires: curl, python3 (no jq dependency)

set -euo pipefail

TENANT_ID="${TENANT_ID:-00000000-0000-0000-0000-000000000001}"
INGEST_URL="${INGEST_URL:-http://localhost:8080}"
QUERY_URL="${QUERY_URL:-http://localhost:8081}"
EVENT_NAME="smoke_test"
N_EVENTS=10
MAX_WAIT=30

GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

pass() { printf "${GREEN}✓${NC} %s\n" "$1"; }
fail() { printf "${RED}✗${NC} %s\n" "$1"; exit 1; }

# Extract a JSON field without jq
json_field() {
  python3 -c "import sys,json; print(json.loads(sys.stdin.read()).get('$1',''))"
}

echo "==> Cascade M1 smoke test"
echo "    Ingest: $INGEST_URL"
echo "    Query:  $QUERY_URL"
echo "    Tenant: $TENANT_ID"
echo ""

# ── 1. Health checks ────────────────────────────────────────────────────────
echo "--- Health checks"

INGEST_HEALTH=$(curl -sf "$INGEST_URL/health" | json_field status)
[ "$INGEST_HEALTH" = "ok" ] && pass "ingest healthy" || fail "ingest unhealthy (got: '$INGEST_HEALTH')"

QUERY_HEALTH=$(curl -sf "$QUERY_URL/health" | json_field status)
[ "$QUERY_HEALTH" = "ok" ] && pass "query healthy" || fail "query unhealthy (got: '$QUERY_HEALTH')"

# ── 2. Baseline count ────────────────────────────────────────────────────────
echo ""
echo "--- Baseline count"
# Anchor all timestamps to a single captured second so events always land within the query window.
NOW_EPOCH=$(date -u +%s)
TS=$(date -u -d "@${NOW_EPOCH}" +"%Y-%m-%dT%H:%M:%S.000Z" 2>/dev/null \
   || date -u -r "${NOW_EPOCH}" +"%Y-%m-%dT%H:%M:%S.000Z")
FROM=$(date -u -d "@$((NOW_EPOCH - 3600))" +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null \
     || date -u -r "$((NOW_EPOCH - 3600))" +"%Y-%m-%dT%H:%M:%SZ")
TO=$(date -u -d "@$((NOW_EPOCH + 300))" +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null \
   || date -u -r "$((NOW_EPOCH + 300))" +"%Y-%m-%dT%H:%M:%SZ")

BASELINE=$(curl -sf \
  -H "X-Tenant-ID: $TENANT_ID" \
  "$QUERY_URL/v1/count?event=$EVENT_NAME&from=$FROM&to=$TO" \
  | json_field count)
BASELINE=${BASELINE:-0}
pass "baseline count = $BASELINE"

# ── 3. Send batch of events ──────────────────────────────────────────────────
echo ""
echo "--- Sending $N_EVENTS events"

BATCH_RESPONSE=$(python3 -c "
import json, urllib.request

events = [
    {
        'event': '$EVENT_NAME',
        'timestamp': '$TS',
        'distinct_id': f'smoke-user-{i}',
        'properties': {'run': 'smoke', 'index': i},
    }
    for i in range($N_EVENTS)
]
payload = json.dumps({'events': events}).encode()
req = urllib.request.Request(
    '$INGEST_URL/v1/batch',
    data=payload,
    headers={
        'Content-Type': 'application/json',
        'X-Tenant-ID': '$TENANT_ID',
    },
    method='POST',
)
with urllib.request.urlopen(req) as resp:
    print(resp.read().decode())
")

ACCEPTED=$(echo "$BATCH_RESPONSE" | json_field accepted)
[ "$ACCEPTED" = "$N_EVENTS" ] \
  && pass "batch accepted $ACCEPTED/$N_EVENTS" \
  || fail "batch only accepted $ACCEPTED/$N_EVENTS (response: $BATCH_RESPONSE)"

# ── 4. Wait for writer flush ─────────────────────────────────────────────────
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
    | json_field count)
  ACTUAL=${ACTUAL:-0}
  echo "    count = $ACTUAL / $EXPECTED (${ELAPSED}s elapsed)"
done

[ "$ACTUAL" -ge "$EXPECTED" ] \
  && pass "count reached $ACTUAL (expected >= $EXPECTED) in ${ELAPSED}s" \
  || fail "count $ACTUAL < $EXPECTED after ${ELAPSED}s — writer may not be flushing"

# ── 5. Single event capture ──────────────────────────────────────────────────
echo ""
echo "--- Single event capture"
SINGLE_RESP=$(python3 -c "
import json, urllib.request

payload = json.dumps({
    'event': '${EVENT_NAME}_single',
    'timestamp': '$TS',
    'distinct_id': 'smoke-single',
}).encode()
req = urllib.request.Request(
    '$INGEST_URL/v1/capture',
    data=payload,
    headers={
        'Content-Type': 'application/json',
        'X-Tenant-ID': '$TENANT_ID',
    },
    method='POST',
)
with urllib.request.urlopen(req) as resp:
    print(resp.read().decode())
")

EVENT_ID=$(echo "$SINGLE_RESP" | json_field event_id)
[ -n "$EVENT_ID" ] && [ "$EVENT_ID" != "None" ] \
  && pass "single capture returned event_id: $EVENT_ID" \
  || fail "single capture did not return event_id (response: $SINGLE_RESP)"

echo ""
printf "${GREEN}==> M1 smoke test PASSED${NC}\n"
