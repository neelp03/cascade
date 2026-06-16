-- Per-minute event counts for fast time-series queries
-- Each docker-entrypoint-initdb.d script is a separate client invocation,
-- so USE must be repeated per file (see 001_events.sql for why).
USE cascade;

CREATE TABLE IF NOT EXISTS events_counts_1m
(
    tenant_id   String,
    event       LowCardinality(String),
    minute      DateTime,
    count       UInt64
)
ENGINE = SummingMergeTree
ORDER BY (tenant_id, event, minute);
