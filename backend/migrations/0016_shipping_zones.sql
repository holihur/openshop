-- 0016_shipping_zones: per-zone shipping rates and product weights.

ALTER TABLE products ADD COLUMN IF NOT EXISTS weight_grams INTEGER NOT NULL DEFAULT 0 CHECK (weight_grams >= 0);
ALTER TABLE product_variants ADD COLUMN IF NOT EXISTS weight_grams INTEGER NOT NULL DEFAULT 0 CHECK (weight_grams >= 0);

CREATE TABLE IF NOT EXISTS shipping_zones (
    id         UUID PRIMARY KEY,
    name       VARCHAR(128) NOT NULL,
    provinces  JSONB NOT NULL DEFAULT '[]'::jsonb,
    active     BOOLEAN NOT NULL DEFAULT true,
    sort       INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_shipping_zones_active ON shipping_zones (active, sort);

CREATE TABLE IF NOT EXISTS shipping_rates (
    id                   UUID PRIMARY KEY,
    zone_id              UUID NOT NULL REFERENCES shipping_zones (id) ON DELETE CASCADE,
    method_id            UUID NOT NULL REFERENCES shipping_methods (id) ON DELETE CASCADE,
    flat_rate_cents      BIGINT NOT NULL DEFAULT 0 CHECK (flat_rate_cents >= 0),
    free_threshold_cents BIGINT NOT NULL DEFAULT 0 CHECK (free_threshold_cents >= 0),
    per_kg_cents         BIGINT NOT NULL DEFAULT 0 CHECK (per_kg_cents >= 0)
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_shipping_rates_zone_method ON shipping_rates (zone_id, method_id);
