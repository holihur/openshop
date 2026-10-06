-- Reverts 0008_addresses.
DROP TABLE IF EXISTS addresses;
ALTER TABLE orders DROP COLUMN IF EXISTS shipping_address;
ALTER TABLE orders DROP COLUMN IF EXISTS tracking_no;
ALTER TABLE orders DROP COLUMN IF EXISTS shipped_at;
ALTER TABLE orders DROP COLUMN IF EXISTS completed_at;
