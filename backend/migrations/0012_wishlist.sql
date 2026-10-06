-- 0012_wishlist: saved products per user.

CREATE TABLE IF NOT EXISTS wishlists (
    user_id    UUID NOT NULL,
    product_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, product_id)
);
CREATE INDEX IF NOT EXISTS idx_wishlists_user ON wishlists (user_id, created_at DESC);
