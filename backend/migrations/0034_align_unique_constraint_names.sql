-- 0034_align_unique_constraint_names: give inline UNIQUE constraints the names
-- GORM's AutoMigrate manages.
--
-- Tables declared with an inline UNIQUE get a PostgreSQL-generated name
-- (<table>_<column>_key), while the models declare uniqueIndex, which expects
-- uni_<table>_<column>. AutoMigrate then tries to drop a constraint that does
-- not exist and fails, so a database created by these migrations could not be
-- used with the development AutoMigrate path (integration tests included).
-- Renaming makes both paths converge.

DO $$
DECLARE
    target record;
BEGIN
    FOR target IN
        SELECT * FROM (VALUES
            ('commissions', 'order_id'),
            ('personal_access_tokens', 'token_hash'),
            ('points_accounts', 'user_id'),
            ('referral_codes', 'code'),
            ('referrals', 'referee_id'),
            ('wallets', 'user_id')
        ) AS t(tbl, col)
    LOOP
        IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = target.tbl || '_' || target.col || '_key') THEN
            EXECUTE format('ALTER TABLE %I RENAME CONSTRAINT %I TO %I',
                target.tbl,
                target.tbl || '_' || target.col || '_key',
                'uni_' || target.tbl || '_' || target.col);
        END IF;
    END LOOP;
END $$;
