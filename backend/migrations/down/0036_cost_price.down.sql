ALTER TABLE order_items DROP COLUMN IF EXISTS cost_cents;
ALTER TABLE product_variants DROP COLUMN IF EXISTS cost_cents;
ALTER TABLE products DROP COLUMN IF EXISTS cost_cents;
