-- 0002_outbox: transactional outbox for reliable event publication.
--
-- Events are inserted in the same transaction as the state change that produced
-- them. A relay (any number of replicas) leases pending rows with FOR UPDATE
-- SKIP LOCKED and publishes them to NATS, guaranteeing at-least-once delivery
-- even if the process crashes between commit and publish.

CREATE TABLE IF NOT EXISTS outbox_events (
    id           UUID PRIMARY KEY,
    subject      VARCHAR(128) NOT NULL,
    payload      JSONB NOT NULL,
    status       VARCHAR(16) NOT NULL DEFAULT 'pending',
    attempts     INTEGER NOT NULL DEFAULT 0,
    available_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    locked_until TIMESTAMPTZ,
    last_error   TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ,
    CONSTRAINT outbox_status_check CHECK (status IN ('pending', 'processing', 'published', 'failed'))
);

-- Partial indexes keep the relay's working set tiny even as history grows.
CREATE INDEX IF NOT EXISTS idx_outbox_pending
    ON outbox_events (available_at)
    WHERE status = 'pending';

CREATE INDEX IF NOT EXISTS idx_outbox_processing
    ON outbox_events (locked_until)
    WHERE status = 'processing';
