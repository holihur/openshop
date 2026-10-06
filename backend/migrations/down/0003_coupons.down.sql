-- Reverts 0003_coupons.
DROP TABLE IF EXISTS coupon_redemptions;
DROP TABLE IF EXISTS coupons;
ALTER TABLE orders DROP COLUMN IF EXISTS subtotal_cents;
ALTER TABLE orders DROP COLUMN IF EXISTS discount_cents;
ALTER TABLE orders DROP COLUMN IF EXISTS coupon_id;
ALTER TABLE orders DROP COLUMN IF EXISTS coupon_code;
