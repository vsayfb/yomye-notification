package event

import "time"

// ListingApprovedNotificationEvent is published after a listing has been
// classified, all family-specific details have been persisted, and the listing
// has been opened. BaseCategorySlug is the stable routing discriminator; base
// category UUIDs must not cross the event boundary.
type ListingApprovedNotificationEvent struct {
	RecipientID      string    `json:"recipient_id"`
	ListingID        string    `json:"listing_id"`
	BaseCategorySlug string    `json:"base_category_slug"`
	Title            string    `json:"title"`
	OccurredAt       time.Time `json:"occurred_at"`
}
