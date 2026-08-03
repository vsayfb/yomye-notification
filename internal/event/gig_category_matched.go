package event

import (
	"time"

	"github.com/google/uuid"
)

// GigCategoryMatchedEvent is published by the gig-processing worker once a
// gig's profession/category has been resolved. It carries the fact only —
// no resolved subscriber list — since the worker doesn't own
// subscriber-domain data and a snapshot taken at publish time would go
// stale (and risk the SQS 256KB message cap) for popular categories/places.
// Subscribers are resolved live at consume time using the category and
// optional international location context.
type GigCategoryMatchedEvent struct {
	GigID      uuid.UUID  `json:"gig_id"`
	CategoryID uuid.UUID  `json:"category_id"`
	CountryID  *uuid.UUID `json:"country_id,omitempty"`
	PlaceID    *uuid.UUID `json:"place_id,omitempty"`
	OccurredAt time.Time  `json:"occurred_at"`
}

// GigCategoryMatchedNotificationEvent is the per-recipient payload the
// service synthesizes — one per matched subscriber, resolved live against
// the database — and feeds into GigCategoryMatchedRenderer. It never
// travels over SQS itself; it only ever exists in-process between the
// fan-out resolver and the renderer.
type GigCategoryMatchedNotificationEvent struct {
	RecipientID uuid.UUID  `json:"recipient_id"`
	GigID       uuid.UUID  `json:"gig_id"`
	CategoryID  uuid.UUID  `json:"category_id"`
	CountryID   *uuid.UUID `json:"country_id,omitempty"`
	PlaceID     *uuid.UUID `json:"place_id,omitempty"`
	OccurredAt  time.Time  `json:"occurred_at"`
}
