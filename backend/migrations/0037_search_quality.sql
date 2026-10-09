-- 0037_search_quality: stemming, typo tolerance and filtered search.
--
-- The original vector used the 'simple' configuration, which does not stem, so
-- searching "lamps" never matched "lamp". A stored generated column cannot be
-- altered in place, so it is rebuilt with the 'english' configuration. CJK text
-- and partial words keep working through the substring fallback in the query
-- layer, and pg_trgm adds tolerance for misspellings.

CREATE EXTENSION IF NOT EXISTS pg_trgm;

DROP INDEX IF EXISTS idx_products_search;
ALTER TABLE products DROP COLUMN IF EXISTS search_vector;
ALTER TABLE products ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        to_tsvector('english', coalesce(title, '') || ' ' || coalesce(description, ''))
    ) STORED;
CREATE INDEX IF NOT EXISTS idx_products_search ON products USING GIN (search_vector);

-- Trigram indexes power fuzzy title matching ("desk lampp" still finds the lamp).
CREATE INDEX IF NOT EXISTS idx_products_title_trgm ON products USING GIN (title gin_trgm_ops);

-- Price range filters and variant attribute filters.
CREATE INDEX IF NOT EXISTS idx_products_price ON products (price_cents);
CREATE INDEX IF NOT EXISTS idx_variants_attributes ON product_variants USING GIN (attributes);
