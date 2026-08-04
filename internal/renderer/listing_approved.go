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

// ListingApprovedRenderer renders the display data carried by the approval
// event. It deliberately does not query listings or either typed-details table.
type ListingApprovedRenderer struct{}

func NewListingApprovedRenderer() *ListingApprovedRenderer {
	return &ListingApprovedRenderer{}
}

func (r *ListingApprovedRenderer) Render(
	ctx context.Context,
	payload json.RawMessage,
) (*notification.Notification, *notification.PushNotification, error) {
	var evt event.ListingApprovedNotificationEvent

	if err := json.Unmarshal(payload, &evt); err != nil {
		return nil, nil, fmt.Errorf("listing_approved: unmarshal payload: %w", err)
	}

	recipientID, err := uuid.Parse(evt.RecipientID)
	if err != nil {
		return nil, nil, fmt.Errorf("listing_approved: invalid recipient_id: %w", err)
	}

	listingID, err := uuid.Parse(evt.ListingID)
	if err != nil {
		return nil, nil, fmt.Errorf("listing_approved: invalid listing_id: %w", err)
	}

	if evt.BaseCategorySlug != notification.BaseCategoryGig {
		return nil, nil, fmt.Errorf(
			"listing_approved: unsupported base_category_slug %q; Yövmiye supports only %q",
			evt.BaseCategorySlug,
			notification.BaseCategoryGig,
		)
	}

	title := "Your listing was approved"
	body := "Your listing is now live."
	localizationKey := notification.LocalizationKeyListingApprovedGeneric
	localizationArgs := map[string]string{}

	if classifiedTitle := strings.TrimSpace(evt.Title); classifiedTitle != "" {
		body = fmt.Sprintf("%s is now live.", classifiedTitle)
		localizationKey = notification.LocalizationKeyListingApproved
		localizationArgs["title"] = classifiedTitle
	}

	n := notification.NewNotification(
		recipientID,
		nil,
		notification.TypeListingApproved,
		notification.EntityTypeListing,
		listingID.String(),
		title,
		body,
		notification.AddLocalizationMetadata(map[string]any{
			"listing_id":         listingID.String(),
			"base_category_slug": evt.BaseCategorySlug,
		}, localizationKey, localizationArgs),
	)

	push, err := notification.NewSemanticPush(
		evt.RecipientID,
		n,
		localizationKey,
		localizationArgs,
		title,
		body,
		nil,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("listing_approved: build semantic push: %w", err)
	}

	return n, push, nil
}
