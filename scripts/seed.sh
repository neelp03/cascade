#!/usr/bin/env bash
# Seed dev data: send a burst of sample events for all three widget types.
set -euo pipefail

TENANT_ID="${TENANT_ID:-00000000-0000-0000-0000-000000000001}"
INGEST_URL="${INGEST_URL:-http://localhost:8080}"
N="${N:-500}"

echo "Seeding $N events per type to tenant $TENANT_ID..."

send_events() {
  local event=$1
  python3 -c "
import json, random, datetime, urllib.request, sys

events = []
now = datetime.datetime.utcnow()
for i in range($N):
    offset = datetime.timedelta(seconds=random.randint(0, 3600))
    events.append({
        'event': '$event',
        'timestamp': (now - offset).strftime('%Y-%m-%dT%H:%M:%S.000Z'),
        'distinct_id': f'seed-user-{i % 50}',
        'properties': {'source': 'seed', 'index': i},
    })

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
    body = json.loads(resp.read())
    print(f'  {body[\"accepted\"]} $event events accepted')
"
}

send_events "page_view"
send_events "signup"
send_events "click"

echo "Done. Refresh the dashboard at http://localhost:3000"
