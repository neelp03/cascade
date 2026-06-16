// ── Event schema (PRD §6.1) ──────────────────────────────────────────────────
// This is the canonical ingest payload. The Go counterpart lives in
// services/ingest/internal/schema/event.go.

export type EventProperties = Record<string, string | number | boolean>;

export interface CaptureEvent {
  event: string;
  timestamp: string; // ISO 8601
  distinct_id: string;
  properties?: EventProperties;
}

// Server-injected fields added by the ingest service before writing to ClickHouse.
export interface StoredEvent extends CaptureEvent {
  event_id: string; // UUIDv7
  tenant_id: string;
  received_at: string; // ISO 8601
}

// ── Tenancy ──────────────────────────────────────────────────────────────────

export interface TenantContext {
  tenant_id: string;
  org_id: string;
  user_id?: string;
}

// ── Query API types (PRD §6.3) ───────────────────────────────────────────────

export interface TimeSeriesRequest {
  tenant_id: string;
  event: string;
  from: string; // ISO 8601
  to: string; // ISO 8601
  interval: 'minute' | 'hour' | 'day';
  breakdown?: string; // property key
}

export interface TimeSeriesPoint {
  timestamp: string;
  count: number;
  breakdown_value?: string;
}

export interface TimeSeriesResponse {
  series: TimeSeriesPoint[];
}

export interface CountRequest {
  tenant_id: string;
  event: string;
  from: string;
  to: string;
}

export interface CountResponse {
  count: number;
}

export interface FunnelStep {
  event: string;
}

export interface FunnelRequest {
  tenant_id: string;
  steps: FunnelStep[];
  from: string;
  to: string;
  window_hours?: number;
}

export interface FunnelStepResult {
  step: number;
  event: string;
  count: number;
  conversion_rate: number;
}

export interface FunnelResponse {
  steps: FunnelStepResult[];
}

// ── Dashboard / Widget types (PRD §6.2) ──────────────────────────────────────

export type WidgetType = 'number' | 'timeseries' | 'funnel' | 'table';

export interface WidgetConfig {
  event: string;
  from?: string;
  to?: string;
  interval?: 'minute' | 'hour' | 'day';
  breakdown?: string;
  funnel_steps?: FunnelStep[];
}

export interface Widget {
  id: string;
  dashboard_id: string;
  title: string;
  type: WidgetType;
  config: WidgetConfig;
  position: { x: number; y: number; w: number; h: number };
  created_at: string;
  updated_at: string;
}

export interface Dashboard {
  id: string;
  org_id: string;
  name: string;
  description?: string;
  widgets: Widget[];
  created_at: string;
  updated_at: string;
}

// ── Org / User types (PRD §6.2) ──────────────────────────────────────────────

export type MemberRole = 'owner' | 'admin' | 'member';

export interface Org {
  id: string;
  name: string;
  slug: string;
  created_at: string;
}

export interface User {
  id: string;
  email: string;
  name: string;
  created_at: string;
}

export interface Membership {
  org_id: string;
  user_id: string;
  role: MemberRole;
}

// ── Realtime message types (PRD §6.2 rt:* channels) ─────────────────────────

export type RealtimeMessageType = 'event_count' | 'cursor_move' | 'dashboard_patch';

export interface RealtimeMessage<T = unknown> {
  type: RealtimeMessageType;
  tenant_id: string;
  dashboard_id?: string;
  payload: T;
}

export interface EventCountPayload {
  event: string;
  count: number;
  window_seconds: number;
}

export interface CursorPayload {
  user_id: string;
  x: number;
  y: number;
}
