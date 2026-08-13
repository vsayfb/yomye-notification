BEGIN;

-- Upgrade databases that applied the original Lambda baseline before it was
-- refreshed. Core remains authoritative for these shared tables.

ALTER TABLE notifications
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conrelid = 'notifications'::regclass
          AND conname = 'notifications_metadata_object'
    ) THEN
        ALTER TABLE notifications
            ADD CONSTRAINT notifications_metadata_object
            CHECK (jsonb_typeof(metadata) = 'object');
    END IF;
END;
$$;

ALTER TABLE fcm_tokens
    ADD COLUMN IF NOT EXISTS platform TEXT NOT NULL DEFAULT 'unknown',
    ADD COLUMN IF NOT EXISTS last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conrelid = 'fcm_tokens'::regclass
          AND conname = 'fcm_tokens_platform_check'
    ) THEN
        ALTER TABLE fcm_tokens
            ADD CONSTRAINT fcm_tokens_platform_check
            CHECK (platform IN ('unknown', 'web', 'android', 'ios'));
    END IF;
END;
$$;

ALTER TABLE fcm_tokens
    DROP CONSTRAINT IF EXISTS fcm_tokens_user_id_token_key;

-- This deliberately fails if historical rows contain one token assigned to
-- multiple users. Ownership must be resolved explicitly; a migration must not
-- silently discard credential rows.
CREATE UNIQUE INDEX IF NOT EXISTS fcm_tokens_token_unique
    ON fcm_tokens (token);

CREATE INDEX IF NOT EXISTS idx_fcm_tokens_user
    ON fcm_tokens (user_id);

CREATE INDEX IF NOT EXISTS fcm_tokens_last_seen_at_idx
    ON fcm_tokens (last_seen_at);

COMMIT;
