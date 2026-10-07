-- 0018_returns: partial refunds and return requests (RMA).
--
-- Refunds are tracked separately from the payment so an order can be refunded in
-- parts; orders.refunded_cents accumulates them. Restocking is opt-in per refund
-- (a refund does not imply the goods came back). Return requests let a customer
-- ask to send a delivered order back; ops approves or rejects.

ALTER TABLE orders ADD COLUMN IF NOT EXISTS refunded_cents BIGINT NOT NULL DEFAULT 0
    CHECK (refunded_cents >= 0);

CREATE TABLE IF NOT EXISTS refunds (
    id           UUID PRIMARY KEY,
    order_id     UUID NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    payment_id   UUID,
    amount_cents BIGINT NOT NULL CHECK (amount_cents > 0),
    reason       VARCHAR(512) NOT NULL DEFAULT '',
    restock      BOOLEAN NOT NULL DEFAULT false,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_refunds_order ON refunds (order_id, created_at DESC);

CREATE TABLE IF NOT EXISTS return_requests (
    id         UUID PRIMARY KEY,
    order_id   UUID NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    user_id    UUID,
    reason     VARCHAR(512) NOT NULL DEFAULT '',
    status     VARCHAR(32) NOT NULL DEFAULT 'requested',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT return_status_check CHECK (status IN ('requested', 'approved', 'rejected'))
);
CREATE INDEX IF NOT EXISTS idx_return_requests_order ON return_requests (order_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_return_requests_status ON return_requests (status, created_at DESC);
