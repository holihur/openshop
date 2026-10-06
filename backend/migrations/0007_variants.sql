-- 0007_variants: product variants (SKUs) with variant-level inventory.
--
-- A product may have zero variants (simple product: stock lives on products) or
-- many (multi-variant: stock lives on product_variants). Order items record the
-- chosen variant so fulfilment and refunds are precise.

CREATE TABLE IF NOT EXISTS product_variants (
    id          UUID PRIMARY KEY,
    product_id  UUID NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    sku         VARCHAR(64) NOT NULL,
    name        VARCHAR(160) NOT NULL,
    price_cents BIGINT NOT NULL DEFAULT 0 CHECK (price_cents >= 0),
    stock       INTEGER NOT NULL DEFAULT 0 CHECK (stock >= 0),
    attributes  JSONB NOT NULL DEFAULT '{}'::jsonb,
    sort        INTEGER NOT NULL DEFAULT 0,
    active      BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_variants_sku ON product_variants (sku);
CREATE INDEX IF NOT EXISTS idx_variants_product ON product_variants (product_id);
CREATE INDEX IF NOT EXISTS idx_variants_product_active ON product_variants (product_id, active);

ALTER TABLE order_items ADD COLUMN IF NOT EXISTS variant_id UUID;
ALTER TABLE order_items ADD COLUMN IF NOT EXISTS variant_name VARCHAR(160) NOT NULL DEFAULT '';
ALTER TABLE order_items ADD COLUMN IF NOT EXISTS sku VARCHAR(64) NOT NULL DEFAULT '';
