-- 0023_product_faqs: per-product frequently asked questions.
CREATE TABLE IF NOT EXISTS product_faqs (
    id         UUID PRIMARY KEY,
    product_id UUID        NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    question   TEXT        NOT NULL,
    answer     TEXT        NOT NULL,
    sort       INT         NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_product_faqs_product ON product_faqs (product_id, sort);
