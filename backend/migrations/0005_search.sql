-- 0005_search: full-text search over products.
--
-- A stored generated tsvector column plus a GIN index gives indexed full-text
-- search for latin text; the query layer additionally falls back to a substring
-- match so CJK and partial words still work.

ALTER TABLE products ADD COLUMN IF NOT EXISTS search_vector tsvector
    GENERATED ALWAYS AS (
        to_tsvector('simple', coalesce(title, '') || ' ' || coalesce(description, ''))
    ) STORED;

CREATE INDEX IF NOT EXISTS idx_products_search ON products USING GIN (search_vector);
