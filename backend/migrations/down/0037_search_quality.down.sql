DROP INDEX IF EXISTS idx_variants_attributes;
DROP INDEX IF EXISTS idx_products_price;
DROP INDEX IF EXISTS idx_products_title_trgm;

DROP INDEX IF EXISTS idx_products_search;
ALTER TABLE products DROP COLUMN IF EXISTS search_vector;
ALTER TABLE products ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        to_tsvector('simple', coalesce(title, '') || ' ' || coalesce(description, ''))
    ) STORED;
CREATE INDEX IF NOT EXISTS idx_products_search ON products USING GIN (search_vector);
