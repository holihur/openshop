-- Mark reviews written by customers who have a paid order for the product.
ALTER TABLE reviews ADD COLUMN IF NOT EXISTS verified_purchase boolean NOT NULL DEFAULT false;
