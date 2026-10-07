-- 0029_login_lockout: throttle credential stuffing per account.
--
-- The per-IP rate limiter alone cannot stop a distributed attack. Counting
-- consecutive failures per account and locking it for a cooling-off period
-- closes that gap; a successful sign-in clears the counter.

ALTER TABLE users ADD COLUMN IF NOT EXISTS failed_attempts INT NOT NULL DEFAULT 0
    CHECK (failed_attempts >= 0);
ALTER TABLE users ADD COLUMN IF NOT EXISTS locked_until TIMESTAMPTZ;
