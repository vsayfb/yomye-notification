package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vsayfb/gig-platform-notification-lambda/internal/notification"
)

// NotificationRepository is the PostgreSQL implementation of
// notification.NotificationRepository. It is persistence-only: no
// notification-specific business logic lives here.
//
// Rows with source_event_id use the producer occurrence as their dedup key;
// rows without it retain the legacy entity-based key. EntityID is always a
// navigation target and is never repurposed for message idempotency.
type NotificationRepository struct {
	pool *pgxpool.Pool
}

func NewNotificationRepository(pool *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{pool: pool}
}

var _ notification.NotificationRepository = (*NotificationRepository)(nil)

func (r *NotificationRepository) Create(ctx context.Context, n *notification.Notification) (*notification.Notification, error) {
	metadata, err := json.Marshal(n.Metadata)

	if err != nil {
		return nil, fmt.Errorf("notification_repository: marshal metadata: %w", err)
	}

	// ON CONFLICT ... DO UPDATE (a harmless no-op self-assignment) rather
	// than DO NOTHING: DO NOTHING can't RETURNING the pre-existing row,
	// and the caller needs that row (specifically pushed_at) regardless
	// of whether this call inserted or hit the conflict.
	const legacyQuery = `
		INSERT INTO notifications
			(id, source_event_id, user_id, actor_id, type, entity_type, entity_id, title, body, metadata, read_at, created_at)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (user_id, entity_type, entity_id, type) WHERE source_event_id IS NULL
		DO UPDATE SET id = notifications.id
		RETURNING id, source_event_id, user_id, actor_id, type, entity_type, entity_id, title, body, metadata, read_at, pushed_at, created_at
	`
	const sourceEventQuery = `
		INSERT INTO notifications
			(id, source_event_id, user_id, actor_id, type, entity_type, entity_id, title, body, metadata, read_at, created_at)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (type, source_event_id) WHERE source_event_id IS NOT NULL
		DO UPDATE SET id = notifications.id
		RETURNING id, source_event_id, user_id, actor_id, type, entity_type, entity_id, title, body, metadata, read_at, pushed_at, created_at
	`
	query := legacyQuery
	if n.SourceEventID != nil {
		query = sourceEventQuery
	}

	var persisted notification.Notification
	var rawMetadata []byte

	err = r.pool.QueryRow(ctx, query,
		n.ID,
		n.SourceEventID,
		n.UserID,
		n.ActorID,
		n.Type,
		n.EntityType,
		n.EntityID,
		n.Title,
		n.Body,
		metadata,
		n.ReadAt,
		n.CreatedAt,
	).Scan(
		&persisted.ID,
		&persisted.SourceEventID,
		&persisted.UserID,
		&persisted.ActorID,
		&persisted.Type,
		&persisted.EntityType,
		&persisted.EntityID,
		&persisted.Title,
		&persisted.Body,
		&rawMetadata,
		&persisted.ReadAt,
		&persisted.PushedAt,
		&persisted.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("notification_repository: insert: %w", err)
	}

	if err := json.Unmarshal(rawMetadata, &persisted.Metadata); err != nil {
		return nil, fmt.Errorf("notification_repository: unmarshal metadata: %w", err)
	}

	return &persisted, nil
}

func (r *NotificationRepository) MarkPushed(ctx context.Context, id uuid.UUID, at time.Time) error {
	const query = `UPDATE notifications SET pushed_at = $2 WHERE id = $1`

	if _, err := r.pool.Exec(ctx, query, id, at); err != nil {
		return fmt.Errorf("notification_repository: mark pushed: %w", err)
	}

	return nil
}
