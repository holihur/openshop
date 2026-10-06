-- Reverts 0006_outbox_trace.
ALTER TABLE outbox_events DROP COLUMN IF EXISTS trace_parent;
