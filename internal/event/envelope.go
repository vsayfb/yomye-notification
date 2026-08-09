package event

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/vsayfb/gig-platform-notification-lambda/internal/notification"
)

const CurrentVersion = 1

// Envelope is the outer shape of every message a business service publishes
// to SQS: a type tag plus the type-specific payload. The handler unmarshals
// only the envelope; the payload is decoded by the renderer registered for
// its Type.
type Envelope struct {
	Type       notification.NotificationEventType `json:"type"`
	EventID    uuid.UUID                          `json:"event_id"`
	Version    int                                `json:"version"`
	OccurredAt time.Time                          `json:"occurred_at"`
	Payload    json.RawMessage                    `json:"payload"`
}

// RequiresVersionedEnvelope identifies producer contracts that guarantee a
// deterministic event_id. Chat's new_message contract has not migrated yet.
func RequiresVersionedEnvelope(eventType notification.NotificationEventType) bool {
	return eventType == notification.EventListingApproved ||
		eventType == notification.EventGigCategoryMatched
}
