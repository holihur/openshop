-- 0027_notifications: in-app notifications.
--
-- A notification is a short message addressed to one user, optionally linking
-- back into the storefront or console. Read state is stored per notification
-- (read_at), so an unread badge is a single indexed count.

CREATE TABLE IF NOT EXISTS notifications (
    id         UUID PRIMARY KEY,
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    type       VARCHAR(32) NOT NULL DEFAULT 'system',
    title      VARCHAR(200) NOT NULL,
    body       VARCHAR(1000) NOT NULL DEFAULT '',
    link       VARCHAR(300) NOT NULL DEFAULT '',
    -- data carries the template variables (order no, amount, …) so the client
    -- can render the message in the reader's own language.
    data       JSONB NOT NULL DEFAULT '{}'::jsonb,
    read_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_unread ON notifications (user_id) WHERE read_at IS NULL;
