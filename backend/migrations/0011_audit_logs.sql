-- 0011_audit_logs: append-only record of security- and admin-relevant actions.

CREATE TABLE IF NOT EXISTS audit_logs (
    id            UUID PRIMARY KEY,
    actor_id      UUID,
    actor_role    VARCHAR(32) NOT NULL DEFAULT '',
    action        VARCHAR(64) NOT NULL,
    resource_type VARCHAR(64) NOT NULL DEFAULT '',
    resource_id   VARCHAR(64) NOT NULL DEFAULT '',
    metadata      JSONB NOT NULL DEFAULT '{}'::jsonb,
    ip            VARCHAR(64) NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_audit_created ON audit_logs (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_actor ON audit_logs (actor_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_logs (action, created_at DESC);
