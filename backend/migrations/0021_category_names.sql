-- 0021_category_names: per-locale category names. The base `name` remains the
-- fallback; `names` maps a locale code to its translation.
ALTER TABLE categories ADD COLUMN IF NOT EXISTS names JSONB NOT NULL DEFAULT '{}'::jsonb;
