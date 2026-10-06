-- 0006_outbox_trace: carry distributed trace context through the outbox so a
-- checkout and the workers that react to it are stitched into one trace.

ALTER TABLE outbox_events ADD COLUMN IF NOT EXISTS trace_parent VARCHAR(128) NOT NULL DEFAULT '';
