-- 0025_loyalty: wallet, points and referral commissions.
--
-- Wallet: a per-user stored-value account with an append-only ledger. Money is
-- debited when an order is placed and credited back if it is cancelled. Points
-- are a separate loyalty balance earned on completion and redeemed at checkout.
-- Commissions are created when a referred customer's order completes and are
-- held for a configurable period (default 15 days) before being credited to the
-- referrer's wallet, so refunds can reverse them.

ALTER TABLE orders ADD COLUMN IF NOT EXISTS wallet_cents BIGINT NOT NULL DEFAULT 0
    CHECK (wallet_cents >= 0);
ALTER TABLE orders ADD COLUMN IF NOT EXISTS points_used BIGINT NOT NULL DEFAULT 0
    CHECK (points_used >= 0);
ALTER TABLE orders ADD COLUMN IF NOT EXISTS points_discount_cents BIGINT NOT NULL DEFAULT 0
    CHECK (points_discount_cents >= 0);

CREATE TABLE IF NOT EXISTS wallets (
    id            UUID PRIMARY KEY,
    user_id       UUID NOT NULL UNIQUE REFERENCES users (id) ON DELETE CASCADE,
    currency      VARCHAR(8) NOT NULL DEFAULT 'USD',
    balance_cents BIGINT NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS wallet_transactions (
    id             UUID PRIMARY KEY,
    wallet_id      UUID NOT NULL REFERENCES wallets (id) ON DELETE CASCADE,
    user_id        UUID NOT NULL,
    type           VARCHAR(32) NOT NULL,
    amount_cents   BIGINT NOT NULL,
    balance_after  BIGINT NOT NULL,
    reference_type VARCHAR(32) NOT NULL DEFAULT '',
    reference_id   VARCHAR(64) NOT NULL DEFAULT '',
    description    VARCHAR(512) NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_wallet_tx_wallet ON wallet_transactions (wallet_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_wallet_tx_user ON wallet_transactions (user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS points_accounts (
    id              UUID PRIMARY KEY,
    user_id         UUID NOT NULL UNIQUE REFERENCES users (id) ON DELETE CASCADE,
    balance         BIGINT NOT NULL DEFAULT 0 CHECK (balance >= 0),
    lifetime_earned BIGINT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS points_transactions (
    id             UUID PRIMARY KEY,
    user_id        UUID NOT NULL,
    type           VARCHAR(32) NOT NULL,
    points         BIGINT NOT NULL,
    balance_after  BIGINT NOT NULL,
    reference_type VARCHAR(32) NOT NULL DEFAULT '',
    reference_id   VARCHAR(64) NOT NULL DEFAULT '',
    description    VARCHAR(512) NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_points_tx_user ON points_transactions (user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS referral_codes (
    user_id    UUID PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    code       VARCHAR(32) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS referrals (
    id          UUID PRIMARY KEY,
    referrer_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    referee_id  UUID NOT NULL UNIQUE REFERENCES users (id) ON DELETE CASCADE,
    code        VARCHAR(32) NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_referrals_referrer ON referrals (referrer_id, created_at DESC);

CREATE TABLE IF NOT EXISTS commissions (
    id           UUID PRIMARY KEY,
    referrer_id  UUID NOT NULL,
    referee_id   UUID NOT NULL,
    order_id     UUID NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    base_cents   BIGINT NOT NULL,
    rate_bps     INT NOT NULL,
    amount_cents BIGINT NOT NULL CHECK (amount_cents >= 0),
    status       VARCHAR(16) NOT NULL DEFAULT 'pending',
    hold_until   TIMESTAMPTZ NOT NULL,
    approved_at  TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT commission_status_check CHECK (status IN ('pending', 'approved', 'reversed')),
    UNIQUE (order_id)
);
CREATE INDEX IF NOT EXISTS idx_commissions_due ON commissions (status, hold_until);
CREATE INDEX IF NOT EXISTS idx_commissions_referrer ON commissions (referrer_id, created_at DESC);
