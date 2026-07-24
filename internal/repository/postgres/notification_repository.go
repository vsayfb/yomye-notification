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
// Dedup guard: notifications_unique on (user_id, entity_type, entity_id,
// type). Known limitation: this key isn't fine-grained enough to tell
// "the same event redelivered" apart from "a genuinely new occurrence
// sharing the same entity" — e.g. two different messages in the same
// thread collide on this key just like a redelivery of the same message
// would. We deliberately resolve that ambiguity by treating a conflict as
// "already have a row" and never re-pushing once pushed_at is set, since
// that's the safer failure mode (a missed follow-up notification is far
// better than spamming a push on every SQS redelivery). If per-message
// granularity turns out to matter, this needs a dedicated idempotency key
// (e.g. a source_event_id column) rather than reusing entity_id for it.
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
	const query = `
		INSERT INTO notifications
			(id, user_id, actor_id, type, entity_type, entity_id, title, body, metadata, read_at, created_at)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (user_id, entity_type, entity_id, type)
		DO UPDATE SET id = notifications.id
		RETURNING id, user_id, actor_id, type, entity_type, entity_id, title, body, metadata, read_at, pushed_at, created_at
	`

	var persisted notification.Notification
	var rawMetadata []byte

	err = r.pool.QueryRow(ctx, query,
		n.ID,
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
