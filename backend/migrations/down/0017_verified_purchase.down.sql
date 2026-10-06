-- Reverts 0017_verified_purchase.
ALTER TABLE reviews DROP COLUMN IF EXISTS verified_purchase;
