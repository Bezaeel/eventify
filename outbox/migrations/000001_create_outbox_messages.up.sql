CREATE TABLE IF NOT EXISTS outbox_messages (
    id           UUID PRIMARY KEY,
    message_id   UUID        NOT NULL,
    name         TEXT        NOT NULL,
    version      TEXT        NOT NULL,
    payload      JSONB       NOT NULL,
    occurred_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ,
    attempts     INT         NOT NULL DEFAULT 0,
    last_error   TEXT
);

-- The relay only ever scans for unpublished rows. A partial index keeps that
-- scan proportional to the backlog rather than to the table, which otherwise
-- grows without bound as published rows accumulate.
CREATE INDEX IF NOT EXISTS idx_outbox_messages_unpublished
    ON outbox_messages (occurred_at)
    WHERE published_at IS NULL;

-- Subscribers deduplicate on message_id; the relay must never mint two rows
-- with the same one.
CREATE UNIQUE INDEX IF NOT EXISTS idx_outbox_messages_message_id
    ON outbox_messages (message_id);
