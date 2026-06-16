import type { CountResponse, TimeSeriesResponse } from '@cascade/types';

const QUERY_BASE = import.meta.env.VITE_QUERY_URL ?? '/api/query';

function headers(tenantId: string): HeadersInit {
  return { 'Content-Type': 'application/json', 'X-Tenant-ID': tenantId };
}

export async function fetchCount(
  tenantId: string,
  event: string,
  from: string,
  to: string
): Promise<CountResponse> {
  const params = new URLSearchParams({ event, from, to });
  const res = await fetch(`${QUERY_BASE}/v1/count?${params}`, {
    headers: headers(tenantId),
  });
  if (!res.ok) throw new Error(`count query failed: ${res.status}`);
  return res.json();
}

export async function fetchTimeSeries(
  tenantId: string,
  event: string,
  from: string,
  to: string,
  interval: 'minute' | 'hour' | 'day' = 'hour'
): Promise<TimeSeriesResponse> {
  const params = new URLSearchParams({ event, from, to, interval });
  const res = await fetch(`${QUERY_BASE}/v1/timeseries?${params}`, {
    headers: headers(tenantId),
  });
  if (!res.ok) throw new Error(`timeseries query failed: ${res.status}`);
  return res.json();
}
