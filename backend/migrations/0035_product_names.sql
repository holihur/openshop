-- 0035_product_names: localized product titles.
--
-- Categories already carry per-locale names; products only had a single title,
-- so a bilingual storefront could not show a product in the shopper's language.
-- names maps a locale to its title; an absent locale falls back to title.

ALTER TABLE products ADD COLUMN IF NOT EXISTS names JSONB NOT NULL DEFAULT '{}'::jsonb;
