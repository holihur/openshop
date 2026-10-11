-- 0040_custom_roles: operator-defined roles.
--
-- The four built-in roles stay in code; this table holds the ones an operator
-- creates. A role stores the permission list directly, so adding a permission to
-- a role is a data change and never a deploy. money is not involved, but the same
-- rules apply as elsewhere: a role that still has members cannot be deleted, and
-- no one may grant a permission they do not hold themselves.

CREATE TABLE IF NOT EXISTS roles (
    key         VARCHAR(64) PRIMARY KEY,
    name        VARCHAR(128) NOT NULL,
    description VARCHAR(512) NOT NULL DEFAULT '',
    permissions JSONB        NOT NULL DEFAULT '[]'::jsonb,
    created_at  TIMESTAMPTZ  NOT NULL,
    updated_at  TIMESTAMPTZ  NOT NULL
);
