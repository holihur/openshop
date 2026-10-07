-- 0026_withdrawals: wallet withdrawal requests and their approval.
--
-- A customer asks to withdraw part of their wallet balance. The funds are
-- debited from the wallet immediately (held) so they cannot be spent while the
-- request is open; a rejection or customer cancellation credits them back.
-- Approval is a review step, not a payment: the actual payout happens offline,
-- and staff record the bank/transfer reference when marking the request paid.

CREATE TABLE IF NOT EXISTS withdrawals (
    id             UUID PRIMARY KEY,
    user_id        UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    amount_cents   BIGINT NOT NULL CHECK (amount_cents > 0),
    currency       VARCHAR(8) NOT NULL DEFAULT '',
    method         VARCHAR(32) NOT NULL DEFAULT 'bank',
    account_name   VARCHAR(200) NOT NULL DEFAULT '',
    account_no     VARCHAR(200) NOT NULL DEFAULT '',
    note           VARCHAR(512) NOT NULL DEFAULT '',
    status         VARCHAR(16) NOT NULL DEFAULT 'requested',
    reject_reason  VARCHAR(512) NOT NULL DEFAULT '',
    paid_reference VARCHAR(200) NOT NULL DEFAULT '',
    reviewed_by    UUID,
    reviewed_at    TIMESTAMPTZ,
    paid_at        TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT withdrawal_status_check CHECK (status IN ('requested', 'approved', 'paid', 'rejected', 'cancelled'))
);
CREATE INDEX IF NOT EXISTS idx_withdrawals_user ON withdrawals (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_withdrawals_status ON withdrawals (status, created_at DESC);
