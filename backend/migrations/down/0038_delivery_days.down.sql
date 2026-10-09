ALTER TABLE shipping_rates DROP CONSTRAINT IF EXISTS shipping_rates_days_check;
ALTER TABLE shipping_methods DROP CONSTRAINT IF EXISTS shipping_methods_days_check;
ALTER TABLE shipping_rates DROP COLUMN IF EXISTS max_days;
ALTER TABLE shipping_rates DROP COLUMN IF EXISTS min_days;
ALTER TABLE shipping_methods DROP COLUMN IF EXISTS max_days;
ALTER TABLE shipping_methods DROP COLUMN IF EXISTS min_days;
