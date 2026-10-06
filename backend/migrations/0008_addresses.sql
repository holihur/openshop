-- 0008_addresses: shipping address book and order fulfilment fields.

CREATE TABLE IF NOT EXISTS addresses (
    id          UUID PRIMARY KEY,
    user_id     UUID NOT NULL,
    recipient   VARCHAR(128) NOT NULL,
    phone       VARCHAR(32) NOT NULL DEFAULT '',
    province    VARCHAR(64) NOT NULL DEFAULT '',
    city        VARCHAR(64) NOT NULL DEFAULT '',
    district    VARCHAR(64) NOT NULL DEFAULT '',
    line1       VARCHAR(255) NOT NULL,
    postal_code VARCHAR(16) NOT NULL DEFAULT '',
    is_default  BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_addresses_user ON addresses (user_id);
-- At most one default address per user.
CREATE UNIQUE INDEX IF NOT EXISTS idx_addresses_user_default
    ON addresses (user_id) WHERE is_default;

-- Orders snapshot the shipping address and track fulfilment.
ALTER TABLE orders ADD COLUMN IF NOT EXISTS shipping_address JSONB;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS tracking_no VARCHAR(128) NOT NULL DEFAULT '';
ALTER TABLE orders ADD COLUMN IF NOT EXISTS shipped_at TIMESTAMPTZ;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS completed_at TIMESTAMPTZ;
