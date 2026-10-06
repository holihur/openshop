-- 0003_coupons: discount coupons and their redemptions.
--
-- Usage limits are enforced by an atomic conditional UPDATE on used_count, so
-- the global cap holds across any number of application replicas.

CREATE TABLE IF NOT EXISTS coupons (
    id                 UUID PRIMARY KEY,
    code               VARCHAR(64) NOT NULL,
    description        TEXT NOT NULL DEFAULT '',
    discount_type      VARCHAR(16) NOT NULL,
    discount_value     BIGINT NOT NULL CHECK (discount_value >= 0),
    min_subtotal_cents BIGINT NOT NULL DEFAULT 0 CHECK (min_subtotal_cents >= 0),
    max_discount_cents BIGINT NOT NULL DEFAULT 0 CHECK (max_discount_cents >= 0),
    usage_limit        INTEGER NOT NULL DEFAULT 0 CHECK (usage_limit >= 0),
    used_count         INTEGER NOT NULL DEFAULT 0 CHECK (used_count >= 0),
    per_user_limit     INTEGER NOT NULL DEFAULT 1 CHECK (per_user_limit >= 0),
    starts_at          TIMESTAMPTZ,
    ends_at            TIMESTAMPTZ,
    active             BOOLEAN NOT NULL DEFAULT true,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT coupons_discount_type_check CHECK (discount_type IN ('percent', 'fixed'))
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_coupons_code ON coupons (upper(code));
CREATE INDEX IF NOT EXISTS idx_coupons_active ON coupons (active);

CREATE TABLE IF NOT EXISTS coupon_redemptions (
    id         UUID PRIMARY KEY,
    coupon_id  UUID NOT NULL REFERENCES coupons (id) ON DELETE CASCADE,
    user_id    UUID NOT NULL,
    order_id   UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- One redemption per order, and a fast per-user count for per-user limits.
CREATE UNIQUE INDEX IF NOT EXISTS idx_coupon_redemptions_order ON coupon_redemptions (coupon_id, order_id);
CREATE INDEX IF NOT EXISTS idx_coupon_redemptions_user ON coupon_redemptions (coupon_id, user_id);

-- Orders gain discount bookkeeping.
ALTER TABLE orders ADD COLUMN IF NOT EXISTS subtotal_cents BIGINT NOT NULL DEFAULT 0;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS discount_cents BIGINT NOT NULL DEFAULT 0;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS coupon_id UUID;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS coupon_code VARCHAR(64) NOT NULL DEFAULT '';
UPDATE orders SET subtotal_cents = total_cents WHERE subtotal_cents = 0;
