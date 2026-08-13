BEGIN;

-- Domain-event idempotency for at-least-once producer delivery.
-- A row with processed_at IS NULL is only a renewable processing claim;
-- the event is complete only after processed_at is set.
CREATE TABLE IF NOT EXISTS notification_processed_events (
    event_id       UUID PRIMARY KEY,
    event_type     TEXT NOT NULL,
    event_version  INTEGER NOT NULL CHECK (event_version > 0),
    claim_token    UUID NOT NULL,
    claimed_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at   TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_notification_processed_events_claimed
ON notification_processed_events (claimed_at)
WHERE processed_at IS NULL;

COMMIT;
