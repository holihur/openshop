-- Reverts 0018_returns.
DROP TABLE IF EXISTS return_requests;
DROP TABLE IF EXISTS refunds;
ALTER TABLE orders DROP COLUMN IF EXISTS refunded_cents;
