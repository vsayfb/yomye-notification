package renderer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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
	if recipientID == senderID {
		return nil, nil, fmt.Errorf("new_message: recipient_id must differ from sender_id")
	}

	evt.SenderName = strings.TrimSpace(evt.SenderName)
	evt.ThreadID = strings.TrimSpace(evt.ThreadID)
	evt.MessageID = strings.TrimSpace(evt.MessageID)
	if evt.SenderName == "" {
		return nil, nil, fmt.Errorf("new_message: sender_name is required")
	}
	if !event.IsMongoObjectID(evt.ThreadID) {
		return nil, nil, fmt.Errorf("new_message: thread_id must be a MongoDB ObjectID")
	}
	if !event.IsMongoObjectID(evt.MessageID) {
		return nil, nil, fmt.Errorf("new_message: message_id must be a MongoDB ObjectID")
	}
	if evt.OccurredAt.IsZero() {
		return nil, nil, fmt.Errorf("new_message: occurred_at is required")
	}
	if evt.GigID != "" {
		gigID, err := uuid.Parse(evt.GigID)
		if err != nil || gigID == uuid.Nil {
			return nil, nil, fmt.Errorf("new_message: invalid gig_id")
		}
		evt.GigID = gigID.String()
	}

	localizationArgs := map[string]string{
		"name": evt.SenderName,
	}

	n := notification.NewNotification(
		recipientID,
		&senderID,
		notification.TypeMessageReceived,
		notification.EntityTypeThread,
		evt.ThreadID,
		evt.SenderName,
		evt.MessagePreview,
		notification.AddLocalizationMetadata(map[string]any{
			"thread_id":  evt.ThreadID,
			"gig_id":     evt.GigID,
			"message_id": evt.MessageID,
		}, notification.LocalizationKeyMessageReceived, localizationArgs),
	)
	n.SourceEventID = &evt.MessageID

	push, err := notification.NewSemanticPush(
		evt.RecipientID,
		n,
		notification.LocalizationKeyMessageReceived,
		localizationArgs,
		evt.SenderName,
		evt.MessagePreview,
		map[string]string{
			"thread_id":  evt.ThreadID,
			"gig_id":     evt.GigID,
			"message_id": evt.MessageID,
		},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("new_message: build semantic push: %w", err)
	}

	return n, push, nil
}
