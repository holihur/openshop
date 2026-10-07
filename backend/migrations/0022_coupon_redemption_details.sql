-- 0022_coupon_redemption_details: record how much a coupon saved and the order
-- number, so the ops console can show a coupon's usage history.
ALTER TABLE coupon_redemptions ADD COLUMN IF NOT EXISTS discount_cents BIGINT NOT NULL DEFAULT 0;
ALTER TABLE coupon_redemptions ADD COLUMN IF NOT EXISTS order_no VARCHAR(64) NOT NULL DEFAULT '';
