-- 0028_personal_access_tokens: programmatic API access.
--
-- A personal access token (PAT) authenticates a user without a browser session.
-- Only the SHA-256 hash of the secret is stored; the raw token is shown once at
-- creation. Tokens carry an explicit scope list and an optional CIDR allow-list,
-- so a token can be limited to a resource set and to a set of source networks.
-- Realm keeps storefront tokens out of the operations API and vice versa.

CREATE TABLE IF NOT EXISTS personal_access_tokens (
    id           UUID PRIMARY KEY,
    user_id      UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name         VARCHAR(100) NOT NULL,
    prefix       VARCHAR(16) NOT NULL,
    token_hash   CHAR(64) NOT NULL UNIQUE,
    realm        VARCHAR(16) NOT NULL DEFAULT 'front',
    scopes       TEXT NOT NULL DEFAULT '',
    cidrs        TEXT NOT NULL DEFAULT '',
    expires_at   TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    revoked_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pat_realm_check CHECK (realm IN ('front', 'ops'))
);
CREATE INDEX IF NOT EXISTS idx_pat_user ON personal_access_tokens (user_id, created_at DESC);
