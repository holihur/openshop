-- 0020_settings: runtime configuration edited from the ops console. Only
-- overrides are stored; defaults come from the environment at startup.
CREATE TABLE IF NOT EXISTS settings (
    key        VARCHAR(128) PRIMARY KEY,
    value      TEXT        NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
