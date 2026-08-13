BEGIN;

ALTER TABLE notifications
    ADD COLUMN IF NOT EXISTS source_event_id TEXT;

ALTER TABLE notifications
    DROP CONSTRAINT IF EXISTS notifications_unique;

CREATE UNIQUE INDEX IF NOT EXISTS notifications_legacy_entity_unique
    ON notifications (user_id, entity_type, entity_id, type)
    WHERE source_event_id IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS notifications_source_event_unique
    ON notifications (type, source_event_id)
    WHERE source_event_id IS NOT NULL;

COMMIT;
