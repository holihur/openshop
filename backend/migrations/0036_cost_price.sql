-- 0036_cost_price: unit cost and gross margin.
--
-- Products and variants gain a cost price so a merchant can see what they earn,
-- not just what they take. Order items snapshot the cost at purchase time, so a
-- later cost change never rewrites the margin of a historical order.

ALTER TABLE products ADD COLUMN IF NOT EXISTS cost_cents BIGINT NOT NULL DEFAULT 0
    CHECK (cost_cents >= 0);
ALTER TABLE product_variants ADD COLUMN IF NOT EXISTS cost_cents BIGINT NOT NULL DEFAULT 0
    CHECK (cost_cents >= 0);
ALTER TABLE order_items ADD COLUMN IF NOT EXISTS cost_cents BIGINT NOT NULL DEFAULT 0
    CHECK (cost_cents >= 0);
