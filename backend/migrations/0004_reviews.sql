-- 0004_reviews: product reviews. A user may review a product once.

CREATE TABLE IF NOT EXISTS reviews (
    id         UUID PRIMARY KEY,
    product_id UUID NOT NULL,
    user_id    UUID NOT NULL,
    rating     SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    title      VARCHAR(160) NOT NULL DEFAULT '',
    body       TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_reviews_product_user ON reviews (product_id, user_id);
CREATE INDEX IF NOT EXISTS idx_reviews_product ON reviews (product_id, created_at DESC);
