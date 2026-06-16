// Package schema defines the canonical event types for the ingest pipeline.
// The TypeScript mirror lives in packages/types/src/index.ts.
// Any change here MUST be reflected there and logged in DECISIONS.md.
package schema

import "time"

// CaptureEvent is the ingest payload sent by the JS SDK (PRD §6.1).
type CaptureEvent struct {
	Event      string            `json:"event"`
	Timestamp  string            `json:"timestamp"`  // ISO 8601
	DistinctID string            `json:"distinct_id"`
	Properties map[string]any    `json:"properties,omitempty"`
}

// StoredEvent is written to ClickHouse by the writer service.
// Fields beyond CaptureEvent are injected server-side.
type StoredEvent struct {
	EventID    string            `json:"event_id"`    // UUIDv7
	TenantID   string            `json:"tenant_id"`
	Event      string            `json:"event"`
	DistinctID string            `json:"distinct_id"`
	Timestamp  time.Time         `json:"timestamp"`
	ReceivedAt time.Time         `json:"received_at"`
	Properties map[string]string `json:"properties"`  // string-serialised
}
