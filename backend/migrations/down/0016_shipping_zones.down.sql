-- Reverts 0016_shipping_zones.
DROP TABLE IF EXISTS shipping_rates;
DROP TABLE IF EXISTS shipping_zones;
ALTER TABLE products DROP COLUMN IF EXISTS weight_grams;
ALTER TABLE product_variants DROP COLUMN IF EXISTS weight_grams;
