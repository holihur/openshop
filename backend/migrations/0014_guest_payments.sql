-- 0014_guest_payments: allow payments without a user for guest checkout.

ALTER TABLE payments ALTER COLUMN user_id DROP NOT NULL;
