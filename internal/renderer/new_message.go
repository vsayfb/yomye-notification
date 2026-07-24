package renderer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/vsayfb/gig-platform-notification-lambda/internal/event"
	"github.com/vsayfb/gig-platform-notification-lambda/internal/notification"
)

// NewMessageRenderer renders NewMessageNotificationEvent into a persisted
// Notification and a PushNotification.
type NewMessageRenderer struct{}

func NewNewMessageRenderer() *NewMessageRenderer {
	return &NewMessageRenderer{}
}

func (r *NewMessageRenderer) Render(
	ctx context.Context,
	payload json.RawMessage,
) (*notification.Notification, *notification.PushNotification, error) {
	var evt event.NewMessageNotificationEvent

	if err := json.Unmarshal(payload, &evt); err != nil {
		return nil, nil, fmt.Errorf("new_message: unmarshal payload: %w", err)
	}

	recipientID, err := uuid.Parse(evt.RecipientID)

	if err != nil {
		return nil, nil, fmt.Errorf("new_message: invalid recipient_id: %w", err)
	}

	senderID, err := uuid.Parse(evt.SenderID)

	if err != nil {
		return nil, nil, fmt.Errorf("new_message: invalid sender_id: %w", err)
	}

	n := notification.NewNotification(
		recipientID,
		&senderID,
		notification.TypeMessageReceived,
		notification.EntityTypeThread,
		evt.ThreadID,
		evt.SenderName,
		evt.MessagePreview,
		map[string]any{
			"gig_id":     evt.GigID,
			"message_id": evt.MessageID,
		},
	)

	push := &notification.PushNotification{
		RecipientID: evt.RecipientID,
		Title:       evt.SenderName,
		Body:        evt.MessagePreview,
		Data: map[string]string{
			"type":       notification.TypeMessageReceived,
			"thread_id":  evt.ThreadID,
			"gig_id":     evt.GigID,
			"message_id": evt.MessageID,
		},
	}

	return n, push, nil
}
