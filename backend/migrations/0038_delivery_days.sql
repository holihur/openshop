-- 0038_delivery_days: express delivery expectations to shoppers.
--
-- Methods carry a default delivery window in business days; a zone rate may
-- override it (a remote province takes longer). 0 on a rate means "inherit the
-- method", so existing rows keep working.

ALTER TABLE shipping_methods ADD COLUMN IF NOT EXISTS min_days INT NOT NULL DEFAULT 3;
ALTER TABLE shipping_methods ADD COLUMN IF NOT EXISTS max_days INT NOT NULL DEFAULT 7;
ALTER TABLE shipping_rates ADD COLUMN IF NOT EXISTS min_days INT NOT NULL DEFAULT 0;
ALTER TABLE shipping_rates ADD COLUMN IF NOT EXISTS max_days INT NOT NULL DEFAULT 0;

ALTER TABLE shipping_methods DROP CONSTRAINT IF EXISTS shipping_methods_days_check;
ALTER TABLE shipping_methods ADD CONSTRAINT shipping_methods_days_check
    CHECK (min_days >= 0 AND max_days >= min_days);
ALTER TABLE shipping_rates DROP CONSTRAINT IF EXISTS shipping_rates_days_check;
ALTER TABLE shipping_rates ADD CONSTRAINT shipping_rates_days_check
    CHECK (min_days >= 0 AND max_days >= min_days);
