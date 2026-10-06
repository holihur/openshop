-- 0010_email_verification: track whether an account's email is verified.

ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified_at TIMESTAMPTZ;
-- Existing accounts are considered verified so nobody is locked out.
UPDATE users SET email_verified = true, email_verified_at = now() WHERE email_verified = false;
