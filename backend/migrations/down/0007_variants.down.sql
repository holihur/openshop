-- Reverts 0007_variants.
DROP TABLE IF EXISTS product_variants;
ALTER TABLE order_items DROP COLUMN IF EXISTS variant_id;
ALTER TABLE order_items DROP COLUMN IF EXISTS variant_name;
ALTER TABLE order_items DROP COLUMN IF EXISTS sku;
