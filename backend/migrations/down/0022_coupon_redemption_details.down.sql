-- Reverts 0022_coupon_redemption_details.
ALTER TABLE coupon_redemptions DROP COLUMN IF EXISTS discount_cents;
ALTER TABLE coupon_redemptions DROP COLUMN IF EXISTS order_no;
