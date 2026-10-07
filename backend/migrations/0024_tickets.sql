-- 0024_tickets: customer support tickets (pre-sale and post-sale).
--
-- A ticket is a threaded conversation between a customer (or a guest through the
-- contact form) and the support team. Messages may be internal notes, which are
-- never returned to the customer. Tickets can be linked to an order (post-sale)
-- or a product (pre-sale) and assigned to a staff member.

CREATE SEQUENCE IF NOT EXISTS ticket_number_seq START 1000;

CREATE TABLE IF NOT EXISTS tickets (
    id          UUID PRIMARY KEY,
    number      BIGINT NOT NULL DEFAULT nextval('ticket_number_seq') UNIQUE,
    user_id     UUID,
    email       VARCHAR(320) NOT NULL DEFAULT '',
    name        VARCHAR(200) NOT NULL DEFAULT '',
    subject     VARCHAR(300) NOT NULL,
    kind        VARCHAR(16) NOT NULL DEFAULT 'other',
    priority    VARCHAR(16) NOT NULL DEFAULT 'normal',
    status      VARCHAR(16) NOT NULL DEFAULT 'open',
    order_id    UUID,
    product_id  UUID,
    assignee_id UUID,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT ticket_kind_check CHECK (kind IN ('presale', 'postsale', 'other')),
    CONSTRAINT ticket_status_check CHECK (status IN ('open', 'pending', 'resolved', 'closed')),
    CONSTRAINT ticket_priority_check CHECK (priority IN ('low', 'normal', 'high', 'urgent'))
);
CREATE INDEX IF NOT EXISTS idx_tickets_status ON tickets (status, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_tickets_user ON tickets (user_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_tickets_assignee ON tickets (assignee_id, updated_at DESC);

CREATE TABLE IF NOT EXISTS ticket_messages (
    id          UUID PRIMARY KEY,
    ticket_id   UUID NOT NULL REFERENCES tickets (id) ON DELETE CASCADE,
    author_id   UUID,
    author_role VARCHAR(16) NOT NULL DEFAULT 'customer',
    body        TEXT NOT NULL,
    internal    BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT ticket_author_role_check CHECK (author_role IN ('customer', 'staff', 'system'))
);
CREATE INDEX IF NOT EXISTS idx_ticket_messages_ticket ON ticket_messages (ticket_id, created_at);
