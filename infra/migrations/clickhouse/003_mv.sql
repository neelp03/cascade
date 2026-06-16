-- Materialized view: populates events_counts_1m on insert
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_events_counts_1m
TO events_counts_1m
AS
SELECT
    tenant_id,
    event,
    toStartOfMinute(timestamp) AS minute,
    count() AS count
FROM events
GROUP BY tenant_id, event, minute;
