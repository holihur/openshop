-- 0030_audit_hash_chain: make the audit trail tamper-evident.
--
-- Each entry stores the hash of the previous entry and its own hash, computed
-- over a canonical serialisation of the entry. Editing or deleting a row breaks
-- every hash after it, so tampering is detectable. Appends take a transaction
-- advisory lock so the chain has exactly one writer at a time.
--
-- Verification covers the retained window: pruning removes the oldest entries,
-- so the boundary entry's prev_hash is not checked.

ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS prev_hash CHAR(64) NOT NULL DEFAULT '';
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS hash CHAR(64) NOT NULL DEFAULT '';
