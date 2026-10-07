-- 0019_retention: indexes that keep time-range scans cheap as the append-only
-- tables grow. A BRIN index is tiny and ideal for the created_at ordering used
-- by the retention pruner and audit queries.
CREATE INDEX IF NOT EXISTS idx_audit_created_brin ON audit_logs USING BRIN (created_at);
CREATE INDEX IF NOT EXISTS idx_outbox_created_brin ON outbox_events USING BRIN (created_at);
