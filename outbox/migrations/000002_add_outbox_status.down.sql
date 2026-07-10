DROP INDEX IF EXISTS idx_outbox_messages_queued;

ALTER TABLE outbox_messages RENAME COLUMN payload_type TO name;

ALTER TABLE outbox_messages
    ADD COLUMN IF NOT EXISTS published_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_error   TEXT,
    ADD COLUMN IF NOT EXISTS version      TEXT NOT NULL DEFAULT 'v1';

UPDATE outbox_messages
   SET published_at = completed_at
 WHERE status = 3;

ALTER TABLE outbox_messages
    DROP COLUMN IF EXISTS status,
    DROP COLUMN IF EXISTS completed_at;

CREATE INDEX IF NOT EXISTS idx_outbox_messages_unpublished
    ON outbox_messages (occurred_at)
    WHERE published_at IS NULL;
