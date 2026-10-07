-- Reverts 0021_category_names.
ALTER TABLE categories DROP COLUMN IF EXISTS names;
