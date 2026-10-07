-- Reverts 0019_retention.
DROP INDEX IF EXISTS idx_audit_created_brin;
DROP INDEX IF EXISTS idx_outbox_created_brin;
