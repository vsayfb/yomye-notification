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
	// Create persists n. Rows carrying SourceEventID use (type,
	// source_event_id); legacy rows use (user_id, entity_type, entity_id,
	// type). On conflict the existing row is returned so callers can use
	// PushedAt to decide whether delivery still needs to be attempted.
	Create(ctx context.Context, n *Notification) (*Notification, error)

	// MarkPushed records that a push was successfully delivered for the
	// notification with the given id, so a future redelivery of the same
	// event won't push again.
	MarkPushed(ctx context.Context, id uuid.UUID, at time.Time) error
}
