-- The relay no longer records publication as a nullable timestamp. A message
-- now moves through an explicit status: QUEUED -> COMPLETED, or -> EXCEEDED
-- once it has burned its retries, or -> POISONED when no processor claims it.
--
-- published_at could only distinguish "sent" from "not yet sent". It had no way
-- to say "will never send", so a message with a malformed payload was retried
-- forever and the partial index it lived in grew without bound.

ALTER TABLE outbox_messages
    ADD COLUMN IF NOT EXISTS status       SMALLINT    NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS completed_at TIMESTAMPTZ;

-- Rows published under the old schema are already done; carry them over rather
-- than re-publishing them on the first poll after this migration.
UPDATE outbox_messages
   SET status = 3, completed_at = published_at
 WHERE published_at IS NOT NULL;

ALTER TABLE outbox_messages DROP COLUMN IF EXISTS published_at;

-- Events are no longer versioned in their type; the pipeline is versioned
-- instead. Nothing dispatches on this column.
ALTER TABLE outbox_messages DROP COLUMN IF EXISTS version;

-- Nothing reads last_error. The failure is logged by the relay at the moment it
-- happens, with the stack of wrapped causes intact; a truncated copy in the row
-- was a second, worse record of the same thing.
ALTER TABLE outbox_messages DROP COLUMN IF EXISTS last_error;

-- The relay dispatches on the payload's declared type, so name it that. The
-- value is still the event's declared name, never a Go type obtained by
-- reflection: a reflected name would stop matching rows already queued the
-- moment someone renamed the struct.
ALTER TABLE outbox_messages RENAME COLUMN name TO payload_type;

DROP INDEX IF EXISTS idx_outbox_messages_unpublished;

-- The relay only ever claims QUEUED rows. A partial index keeps that scan
-- proportional to the backlog rather than to the table, which otherwise grows
-- without bound as completed rows accumulate.
CREATE INDEX IF NOT EXISTS idx_outbox_messages_queued
    ON outbox_messages (occurred_at)
    WHERE status = 1;
