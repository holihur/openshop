-- Reverts 0013_guest_checkout. Restores NOT NULL only when no guest orders
-- remain, so the rollback never silently deletes financial records.
DROP INDEX IF EXISTS idx_orders_access_token;
ALTER TABLE orders DROP COLUMN IF EXISTS guest_email;
ALTER TABLE orders DROP COLUMN IF EXISTS guest_phone;
ALTER TABLE orders DROP COLUMN IF EXISTS access_token;
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM orders WHERE user_id IS NULL) THEN
        ALTER TABLE orders ALTER COLUMN user_id SET NOT NULL;
    END IF;
END $$;
