BEGIN;

-- This is the current baseline for the Core-owned tables used by Lambda.
-- The referenced users table must already exist.

CREATE TABLE notifications (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_event_id TEXT,

    user_id UUID NOT NULL
        REFERENCES users(id) ON DELETE CASCADE,
    actor_id UUID
        REFERENCES users(id) ON DELETE SET NULL,

    type        TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id   TEXT NOT NULL,

    -- Compatibility display text. Semantic localization data is authoritative.
    title TEXT NOT NULL,
    body  TEXT NOT NULL,

    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    read_at   TIMESTAMPTZ,
    pushed_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT notifications_metadata_object
        CHECK (jsonb_typeof(metadata) = 'object')
);

COMMENT ON TABLE notifications IS 'User notification inbox.';
COMMENT ON COLUMN notifications.source_event_id IS
'Producer occurrence identity; entity_id remains the navigation target.';
COMMENT ON COLUMN notifications.pushed_at IS
'Push processing completion timestamp, not proof of device display.';

CREATE UNIQUE INDEX notifications_legacy_entity_unique
    ON notifications (user_id, entity_type, entity_id, type)
    WHERE source_event_id IS NULL;

CREATE UNIQUE INDEX notifications_source_event_unique
    ON notifications (type, source_event_id)
    WHERE source_event_id IS NOT NULL;

CREATE INDEX idx_notifications_user_created
    ON notifications (user_id, created_at DESC);

CREATE INDEX idx_notifications_user_unread
    ON notifications (user_id)
    WHERE read_at IS NULL;

CREATE INDEX idx_notifications_entity
    ON notifications (entity_type, entity_id);

CREATE INDEX idx_notifications_actor
    ON notifications (actor_id);

CREATE TABLE fcm_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    user_id UUID NOT NULL
        REFERENCES users(id) ON DELETE CASCADE,

    token        TEXT NOT NULL UNIQUE,
    platform     TEXT NOT NULL DEFAULT 'unknown',
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fcm_tokens_platform_check
        CHECK (platform IN ('unknown', 'web', 'android', 'ios'))
);

COMMENT ON COLUMN fcm_tokens.platform IS
'Client-reported operational metadata; unknown is reserved for legacy rows.';

CREATE INDEX idx_fcm_tokens_user
    ON fcm_tokens (user_id);

CREATE INDEX fcm_tokens_last_seen_at_idx
    ON fcm_tokens (last_seen_at);

-- Domain-event idempotency for at-least-once producer delivery.
-- A pending row is a renewable claim; processed_at marks durable completion.
CREATE TABLE notification_processed_events (
    event_id      UUID PRIMARY KEY,
    event_type    TEXT NOT NULL,
    event_version INTEGER NOT NULL
        CHECK (event_version > 0),
    claim_token  UUID NOT NULL,
    claimed_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ
);

CREATE INDEX idx_notification_processed_events_claimed
    ON notification_processed_events (claimed_at)
    WHERE processed_at IS NULL;

COMMIT;
