-- Materialized view: populates events_counts_1m on insert
-- Each docker-entrypoint-initdb.d script is a separate client invocation,
-- so USE must be repeated per file (see 001_events.sql for why).
USE cascade;

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
