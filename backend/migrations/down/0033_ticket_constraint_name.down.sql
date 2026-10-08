DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'uni_tickets_number') THEN
        ALTER TABLE tickets RENAME CONSTRAINT uni_tickets_number TO tickets_number_key;
    END IF;
END $$;
