-- =============================================================================
-- Notifications
-- =============================================================================

CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Recipient
    user_id UUID NOT NULL
        REFERENCES users(id) ON DELETE CASCADE,

    -- User who triggered the notification (nullable for system notifications)
    actor_id UUID
        REFERENCES users(id) ON DELETE SET NULL,

    -- Notification type
    type TEXT NOT NULL,

    -- Target entity
    entity_type TEXT NOT NULL,
    entity_id TEXT NOT NULL,

    -- Display content
    title TEXT NOT NULL,
    body TEXT NOT NULL,

    -- Optional payload
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,

    -- Read state
    read_at TIMESTAMPTZ,

    -- Push delivery state. NULL = not yet (successfully) pushed. Kept
    -- separate from row existence: a redelivered event that already
    -- created this row but whose push previously failed must still be
    -- able to retry the push, without depending on entity dedup alone.
    pushed_at TIMESTAMPTZ,

    -- Audit
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT notifications_unique
        UNIQUE (user_id, entity_type, entity_id, type)
);

CREATE INDEX idx_notifications_user_created
    ON notifications (user_id, created_at DESC);

CREATE INDEX idx_notifications_user_unread
    ON notifications (user_id)
    WHERE read_at IS NULL;

CREATE INDEX idx_notifications_entity
    ON notifications (entity_type, entity_id);

CREATE INDEX idx_notifications_actor
    ON notifications (actor_id);

-- =============================================================================
-- FCM Tokens
-- =============================================================================

CREATE TABLE fcm_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    token       TEXT NOT NULL,

    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (user_id, token)
);

CREATE INDEX idx_fcm_tokens_user
ON fcm_tokens(user_id);

COMMIT;
