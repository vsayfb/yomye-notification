package renderer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/vsayfb/gig-platform-notification-lambda/internal/event"
	"github.com/vsayfb/gig-platform-notification-lambda/internal/notification"
)

// GigApprovedRenderer renders GigApprovedNotificationEvent into a persisted
// Notification and a PushNotification. Wording here is a starting point —
// adjust copy as needed, since presentation is owned by this service.
type GigApprovedRenderer struct{}

func NewGigApprovedRenderer() *GigApprovedRenderer {
	return &GigApprovedRenderer{}
}

func (r *GigApprovedRenderer) Render(
	ctx context.Context,
	payload json.RawMessage,
) (*notification.Notification, *notification.PushNotification, error) {
	var evt event.GigApprovedNotificationEvent

	if err := json.Unmarshal(payload, &evt); err != nil {
		return nil, nil, fmt.Errorf("gig_approved: unmarshal payload: %w", err)
	}

	recipientID, err := uuid.Parse(evt.RecipientID)

	if err != nil {
		return nil, nil, fmt.Errorf("gig_approved: invalid recipient_id: %w", err)
	}

	var actorID *uuid.UUID

	if evt.ApproverID != "" {
		id, err := uuid.Parse(evt.ApproverID)

		if err != nil {
			return nil, nil, fmt.Errorf("gig_approved: invalid approver_id: %w", err)
		}

		actorID = &id
	}

	title := "Your gig was approved"
	body := fmt.Sprintf("%s is now live.", evt.GigTitle)

	n := notification.NewNotification(
		recipientID,
		actorID,
		notification.TypeGigApproved,
		notification.EntityTypeGig,
		evt.GigID,
		title,
		body,
		map[string]any{
			"gig_id": evt.GigID,
		},
	)

	push := &notification.PushNotification{
		RecipientID: evt.RecipientID,
		Title:       title,
		Body:        body,
		Data: map[string]string{
			"type":   notification.TypeGigApproved,
			"gig_id": evt.GigID,
		},
	}

	return n, push, nil
}
