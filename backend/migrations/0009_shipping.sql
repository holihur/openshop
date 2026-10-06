-- 0009_shipping: selectable shipping methods and order shipping/tax totals.

CREATE TABLE IF NOT EXISTS shipping_methods (
    id                   UUID PRIMARY KEY,
    code                 VARCHAR(64) NOT NULL,
    name                 VARCHAR(128) NOT NULL,
    flat_rate_cents      BIGINT NOT NULL DEFAULT 0 CHECK (flat_rate_cents >= 0),
    free_threshold_cents BIGINT NOT NULL DEFAULT 0 CHECK (free_threshold_cents >= 0),
    active               BOOLEAN NOT NULL DEFAULT true,
    sort                 INTEGER NOT NULL DEFAULT 0,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_shipping_methods_code ON shipping_methods (code);

ALTER TABLE orders ADD COLUMN IF NOT EXISTS shipping_cents BIGINT NOT NULL DEFAULT 0;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS tax_cents BIGINT NOT NULL DEFAULT 0;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS shipping_method_id UUID;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS shipping_method_name VARCHAR(128) NOT NULL DEFAULT '';
