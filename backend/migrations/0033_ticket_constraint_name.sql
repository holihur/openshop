-- 0033_ticket_constraint_name: align the tickets unique constraint with the
-- name GORM's AutoMigrate manages.
--
-- The constraint was declared inline in 0024, so PostgreSQL named it
-- tickets_number_key. AutoMigrate expects uni_tickets_number and tries to drop
-- the other one, which fails on a database created by these migrations. Renaming
-- it lets a migrated database and the development AutoMigrate path converge.

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'tickets_number_key') THEN
        ALTER TABLE tickets RENAME CONSTRAINT tickets_number_key TO uni_tickets_number;
    END IF;
END $$;
