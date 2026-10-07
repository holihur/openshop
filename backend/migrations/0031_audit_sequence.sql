-- 0031_audit_sequence: a monotonic append order for the audit chain.
--
-- Timestamps are stored with microsecond precision and ids are random UUIDs, so
-- neither can order entries written within the same microsecond. A sequence
-- column gives the chain an unambiguous append order.

ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS seq BIGSERIAL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_audit_logs_seq ON audit_logs (seq);
