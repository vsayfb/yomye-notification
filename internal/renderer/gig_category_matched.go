package renderer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/vsayfb/gig-platform-notification-lambda/internal/event"
	"github.com/vsayfb/gig-platform-notification-lambda/internal/notification"
)

// GigCategoryMatchedRenderer renders a single
// GigCategoryMatchedNotificationEvent (one resolved recipient) into a
// persisted Notification and a PushNotification. It has no idea a fan-out
// happened upstream — from here it's just another per-recipient event.
type GigCategoryMatchedRenderer struct{}

func NewGigCategoryMatchedRenderer() *GigCategoryMatchedRenderer {
	return &GigCategoryMatchedRenderer{}
}

func (r *GigCategoryMatchedRenderer) Render(
	ctx context.Context,
	payload json.RawMessage,
) (*notification.Notification, *notification.PushNotification, error) {
	var evt event.GigCategoryMatchedNotificationEvent

	if err := json.Unmarshal(payload, &evt); err != nil {
		return nil, nil, fmt.Errorf("gig_category_matched: unmarshal payload: %w", err)
	}

	title := "New gig in your category"
	body := "A gig matching your category just went live near you."
	localizationArgs := map[string]string{}

	n := notification.NewNotification(
		evt.RecipientID,
		nil, // system-matched, no acting user
		notification.TypeGigCategoryMatched,
		notification.EntityTypeListing,
		evt.GigID.String(),
		title,
		body,
		notification.AddLocalizationMetadata(map[string]any{
			"category_id": evt.CategoryID,
		}, notification.LocalizationKeyGigCategoryMatched, localizationArgs),
	)

	push, err := notification.NewSemanticPush(
		evt.RecipientID.String(),
		n,
		notification.LocalizationKeyGigCategoryMatched,
		localizationArgs,
		title,
		body,
		map[string]string{
			"category_id": evt.CategoryID.String(),
		},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("gig_category_matched: build semantic push: %w", err)
	}

	return n, push, nil
}
