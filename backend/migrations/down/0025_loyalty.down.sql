DROP TABLE IF EXISTS commissions;
DROP TABLE IF EXISTS referrals;
DROP TABLE IF EXISTS referral_codes;
DROP TABLE IF EXISTS points_transactions;
DROP TABLE IF EXISTS points_accounts;
DROP TABLE IF EXISTS wallet_transactions;
DROP TABLE IF EXISTS wallets;

ALTER TABLE orders DROP COLUMN IF EXISTS points_discount_cents;
ALTER TABLE orders DROP COLUMN IF EXISTS points_used;
ALTER TABLE orders DROP COLUMN IF EXISTS wallet_cents;
