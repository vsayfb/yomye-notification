BEGIN;

ALTER TABLE notifications
    ADD COLUMN source_event_id TEXT;

ALTER TABLE notifications
    DROP CONSTRAINT notifications_unique;

CREATE UNIQUE INDEX notifications_legacy_entity_unique
    ON notifications (user_id, entity_type, entity_id, type)
    WHERE source_event_id IS NULL;

CREATE UNIQUE INDEX notifications_source_event_unique
    ON notifications (type, source_event_id)
    WHERE source_event_id IS NOT NULL;

COMMIT;
