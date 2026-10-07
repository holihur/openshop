-- 0032_audit_hash_varchar: CHAR pads with spaces, which broke the chain.
--
-- prev_hash and hash were declared CHAR(64). CHAR blank-pads to the fixed width,
-- so the genesis entry's empty prev_hash was stored and read back as 64 spaces
-- and never matched the value used to compute its hash. VARCHAR keeps the value
-- exactly as written.

ALTER TABLE audit_logs ALTER COLUMN prev_hash TYPE VARCHAR(64);
ALTER TABLE audit_logs ALTER COLUMN hash TYPE VARCHAR(64);
UPDATE audit_logs SET prev_hash = rtrim(prev_hash), hash = rtrim(hash);
