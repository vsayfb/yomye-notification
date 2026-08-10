package notification

import (
	"time"

	"github.com/google/uuid"
)

type NotificationEventType string

const (
	EventNewMessage         NotificationEventType = "new_message"
	EventListingApproved    NotificationEventType = "listing_approved"
	EventGigCategoryMatched NotificationEventType = "gig_category_matched"
)

// Notification.Type values (the persisted, notification-domain type,
// as distinct from NotificationEventType which identifies the inbound event).
const (
	TypeMessageReceived    = "message_received"
	TypeListingApproved    = "listing_approved"
	TypeGigCategoryMatched = "gig_category_matched"
)

// Flutter localization catalog keys. These are persisted and delivered as
// semantic identifiers; Lambda never chooses a user locale.
const (
	LocalizationKeyMessageReceived        = "notifications.message_received"
	LocalizationKeyListingApproved        = "notifications.listing_approved"
	LocalizationKeyListingApprovedGeneric = "notifications.listing_approved_generic"
	LocalizationKeyGigCategoryMatched     = "notifications.gig_category_matched"
)

// Notification.EntityType values.
const (
	EntityTypeThread  = "thread"
	EntityTypeListing = "listing"
)

// Stable base_categories.slug values used to route family-specific listing
// behavior. UUIDs are intentionally not represented in this package.
const (
	BaseCategoryGig = "gig"
)

type Notification struct {
	ID uuid.UUID `db:"id"`

	// SourceEventID identifies the producer-side occurrence independently
	// from EntityID, which remains the client navigation target. It is set
	// for chat messages so two messages in one thread remain distinct while
	// an SQS redelivery of the same message stays idempotent.
	SourceEventID *string `db:"source_event_id"`

	// Recipient
	UserID uuid.UUID `db:"user_id"`

	// User who triggered the notification (nil for system notifications)
	ActorID *uuid.UUID `db:"actor_id"`

	// Business event
	Type string `db:"type"`

	// Navigation target. Rows without a SourceEventID retain the legacy
	// (user_id, entity_type, entity_id, type) idempotency key.
	EntityType string `db:"entity_type"`
	EntityID   string `db:"entity_id"`

	// Display content
	Title string `db:"title"`
	Body  string `db:"body"`

	// Optional structured data
	Metadata map[string]any `db:"metadata"`

	// Read state
	ReadAt *time.Time `db:"read_at"`

	// PushedAt is set once a push has actually been delivered for this
	// notification. Nil means no push has been confirmed sent — so even
	// if the notification row already exists (dedup conflict), a push
	// should still be attempted.
	PushedAt *time.Time `db:"pushed_at"`

	// Audit
	CreatedAt time.Time `db:"created_at"`
}

type PushNotification struct {
	RecipientID string

	Title string
	Body  string

	Data map[string]string
}
