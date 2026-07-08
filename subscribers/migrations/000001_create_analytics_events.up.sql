CREATE TABLE IF NOT EXISTS analytics_events (
    message_id   UUID PRIMARY KEY,
    event_id     UUID        NOT NULL,
    name         TEXT        NOT NULL,
    type         TEXT        NOT NULL,
    country_code TEXT        NOT NULL,
    created_by   TEXT        NOT NULL,
    occurred_at  TIMESTAMPTZ NOT NULL,
    ingested_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- message_id is the primary key precisely so that redelivery from the outbox
-- relay collides here and is absorbed by ON CONFLICT DO NOTHING.

CREATE INDEX IF NOT EXISTS idx_analytics_events_occurred_at
    ON analytics_events (occurred_at DESC);

CREATE INDEX IF NOT EXISTS idx_analytics_events_country_code
    ON analytics_events (country_code);
