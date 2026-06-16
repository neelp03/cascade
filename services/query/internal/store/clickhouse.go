package store

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
)

// ClickHouseStore runs analytical queries against ClickHouse.
// Every query is scoped by tenant_id — callers MUST supply it.
type ClickHouseStore struct {
	conn clickhouse.Conn
}

func NewClickHouseStore(dsn string) (*ClickHouseStore, error) {
	opts, err := clickhouse.ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse ClickHouse DSN: %w", err)
	}
	conn, err := clickhouse.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("open ClickHouse: %w", err)
	}
	return &ClickHouseStore{conn: conn}, nil
}

func (s *ClickHouseStore) Ping(ctx context.Context) error {
	return s.conn.Ping(ctx)
}

// CountEvents returns the total number of events matching the filter.
func (s *ClickHouseStore) CountEvents(
	ctx context.Context,
	tenantID, event string,
	from, to time.Time,
) (int64, error) {
	var count uint64
	err := s.conn.QueryRow(ctx, `
		SELECT count()
		FROM events
		WHERE tenant_id = {tenant_id:String}
		  AND event      = {event:String}
		  AND timestamp >= toDateTime64({from:Int64}, 3, 'UTC')
		  AND timestamp <  toDateTime64({to:Int64}, 3, 'UTC')
	`, clickhouse.Named("tenant_id", tenantID),
		clickhouse.Named("event", event),
		clickhouse.Named("from", from.Unix()),
		clickhouse.Named("to", to.Unix()),
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count query: %w", err)
	}
	return int64(count), nil
}

// TimeSeriesPoint is one bucket in a time-series result.
type TimeSeriesPoint struct {
	Timestamp time.Time
	Count     uint64
	Breakdown string
}

// TimeSeries returns event counts grouped by the given interval.
func (s *ClickHouseStore) TimeSeries(
	ctx context.Context,
	tenantID, event string,
	from, to time.Time,
	interval string, // "minute" | "hour" | "day"
	breakdown string, // property key; empty = no breakdown
) ([]TimeSeriesPoint, error) {
	truncFn := map[string]string{
		"minute": "toStartOfMinute",
		"hour":   "toStartOfHour",
		"day":    "toStartOfDay",
	}[interval]
	if truncFn == "" {
		return nil, fmt.Errorf("unsupported interval: %s", interval)
	}

	query := fmt.Sprintf(`
		SELECT %s(timestamp) AS ts, count() AS cnt
		FROM events
		WHERE tenant_id = {tenant_id:String}
		  AND event      = {event:String}
		  AND timestamp >= toDateTime64({from:Int64}, 3, 'UTC')
		  AND timestamp <  toDateTime64({to:Int64}, 3, 'UTC')
		GROUP BY ts
		ORDER BY ts
	`, truncFn)

	rows, err := s.conn.Query(ctx, query,
		clickhouse.Named("tenant_id", tenantID),
		clickhouse.Named("event", event),
		clickhouse.Named("from", from.Unix()),
		clickhouse.Named("to", to.Unix()),
	)
	if err != nil {
		return nil, fmt.Errorf("timeseries query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var pts []TimeSeriesPoint
	for rows.Next() {
		var p TimeSeriesPoint
		if err := rows.Scan(&p.Timestamp, &p.Count); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		pts = append(pts, p)
	}
	return pts, rows.Err()
}
