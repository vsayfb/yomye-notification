package notification

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// NotificationRepository persists notifications. Implementations should be
// persistence-only; all notification-specific business logic belongs in
// the renderers.
type NotificationRepository interface {
	// Create persists n, keyed by (user_id, entity_type, entity_id,
	// type) — notifications_unique. If a row for that key already
	// exists, the existing row is returned instead of erroring or
	// silently no-op'ing — callers decide what to do next based on the
	// returned Notification's PushedAt, not based on whether a new row
	// was inserted.
	Create(ctx context.Context, n *Notification) (*Notification, error)

	// MarkPushed records that a push was successfully delivered for the
	// notification with the given id, so a future redelivery of the same
	// event won't push again.
	MarkPushed(ctx context.Context, id uuid.UUID, at time.Time) error
}
