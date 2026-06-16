package schema

import "time"

// StoredEvent mirrors services/ingest/internal/schema.StoredEvent.
// Duplicated intentionally — Go services do not share modules.
type StoredEvent struct {
	EventID    string            `json:"event_id"`
	TenantID   string            `json:"tenant_id"`
	Event      string            `json:"event"`
	DistinctID string            `json:"distinct_id"`
	Timestamp  time.Time         `json:"timestamp"`
	ReceivedAt time.Time         `json:"received_at"`
	Properties map[string]string `json:"properties"`
}
