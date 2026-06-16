package writer

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/cascade-analytics/cascade/services/writer/internal/schema"
)

// ClickHouseWriter batch-inserts StoredEvents into the events table.
type ClickHouseWriter struct {
	conn clickhouse.Conn
}

func NewClickHouseWriter(dsn string) (*ClickHouseWriter, error) {
	opts, err := clickhouse.ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse ClickHouse DSN: %w", err)
	}
	conn, err := clickhouse.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("open ClickHouse: %w", err)
	}
	return &ClickHouseWriter{conn: conn}, nil
}

// Ping checks ClickHouse connectivity.
func (w *ClickHouseWriter) Ping(ctx context.Context) error {
	return w.conn.Ping(ctx)
}

// WriteBatch inserts a slice of StoredEvents in a single batch operation.
// Properties values are stored as a Map(String, String) in ClickHouse.
func (w *ClickHouseWriter) WriteBatch(ctx context.Context, events []schema.StoredEvent) error {
	if len(events) == 0 {
		return nil
	}

	batch, err := w.conn.PrepareBatch(ctx, `
		INSERT INTO events
		(event_id, tenant_id, event, distinct_id, timestamp, received_at, properties)
	`)
	if err != nil {
		return fmt.Errorf("prepare batch: %w", err)
	}

	for _, ev := range events {
		if err := batch.Append(
			ev.EventID,
			ev.TenantID,
			ev.Event,
			ev.DistinctID,
			ev.Timestamp,
			ev.ReceivedAt,
			ev.Properties,
		); err != nil {
			return fmt.Errorf("append row: %w", err)
		}
	}

	if err := batch.Send(); err != nil {
		return fmt.Errorf("send batch: %w", err)
	}
	return nil
}
