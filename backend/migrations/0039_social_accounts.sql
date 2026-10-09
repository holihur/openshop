-- 0039_social_accounts: link external identities (WeChat, Alipay, ...) to a user.
--
-- WeChat and Alipay do not return an email address, so an account cannot be
-- matched by email. The provider's own subject (openid/unionid, Alipay user id)
-- is the link, with a uniqueness constraint so one external account always maps
-- to exactly one shop account.

CREATE TABLE IF NOT EXISTS social_accounts (
    id         UUID PRIMARY KEY,
    provider   VARCHAR(40)  NOT NULL,
    subject    VARCHAR(191) NOT NULL,
    user_id    UUID         NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    email      VARCHAR(191) NOT NULL DEFAULT '',
    name       VARCHAR(191) NOT NULL DEFAULT '',
    avatar_url TEXT         NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ  NOT NULL,
    updated_at TIMESTAMPTZ  NOT NULL
);

-- One external account belongs to one shop account. GORM names the constraint
-- the same way, so AutoMigrate and the SQL migrations agree.
ALTER TABLE social_accounts DROP CONSTRAINT IF EXISTS uni_social_accounts_provider_subject;
ALTER TABLE social_accounts ADD CONSTRAINT uni_social_accounts_provider_subject
    UNIQUE (provider, subject);

CREATE INDEX IF NOT EXISTS idx_social_accounts_user ON social_accounts (user_id);
