-- 0013_guest_checkout: allow orders without an account.

ALTER TABLE orders ALTER COLUMN user_id DROP NOT NULL;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS guest_email VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE orders ADD COLUMN IF NOT EXISTS guest_phone VARCHAR(32) NOT NULL DEFAULT '';
ALTER TABLE orders ADD COLUMN IF NOT EXISTS access_token VARCHAR(128) NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_orders_access_token
    ON orders (access_token) WHERE access_token <> '';
