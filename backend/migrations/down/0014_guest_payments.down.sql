-- Reverts 0014_guest_payments.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM payments WHERE user_id IS NULL) THEN
        ALTER TABLE payments ALTER COLUMN user_id SET NOT NULL;
    END IF;
END $$;
