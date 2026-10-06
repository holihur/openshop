-- Reverts 0009_shipping.
DROP TABLE IF EXISTS shipping_methods;
ALTER TABLE orders DROP COLUMN IF EXISTS shipping_cents;
ALTER TABLE orders DROP COLUMN IF EXISTS tax_cents;
ALTER TABLE orders DROP COLUMN IF EXISTS shipping_method_id;
ALTER TABLE orders DROP COLUMN IF EXISTS shipping_method_name;
