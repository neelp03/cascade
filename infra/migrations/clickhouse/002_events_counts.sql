-- Per-minute event counts for fast time-series queries
CREATE TABLE IF NOT EXISTS events_counts_1m
(
    tenant_id   String,
    event       LowCardinality(String),
    minute      DateTime,
    count       UInt64
)
ENGINE = SummingMergeTree
ORDER BY (tenant_id, event, minute);
