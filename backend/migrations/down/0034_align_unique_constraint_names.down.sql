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
        IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'uni_' || target.tbl || '_' || target.col) THEN
            EXECUTE format('ALTER TABLE %I RENAME CONSTRAINT %I TO %I',
                target.tbl,
                'uni_' || target.tbl || '_' || target.col,
                target.tbl || '_' || target.col || '_key');
        END IF;
    END LOOP;
END $$;
