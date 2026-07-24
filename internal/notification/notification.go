package notification

import (
	"time"

	"github.com/google/uuid"
)

// NewNotification builds a Notification ready to persist. Renderers call
// this after translating an inbound event into notification-domain terms.
func NewNotification(
	userID uuid.UUID,
	actorID *uuid.UUID,
	notificationType string,
	entityType string,
	entityID string,
	title string,
	body string,
	metadata map[string]any,
) *Notification {
	if metadata == nil {
		metadata = map[string]any{}
	}

	return &Notification{
		ID:         uuid.New(),
		UserID:     userID,
		ActorID:    actorID,
		Type:       notificationType,
		EntityType: entityType,
		EntityID:   entityID,
		Title:      title,
		Body:       body,
		Metadata:   metadata,
		CreatedAt:  time.Now().UTC(),
	}
}

// MarkRead marks the notification as read at the given time.
func (n *Notification) MarkRead(at time.Time) {
	n.ReadAt = &at
}

// IsRead reports whether the notification has been read.
func (n *Notification) IsRead() bool {
	return n.ReadAt != nil
}
