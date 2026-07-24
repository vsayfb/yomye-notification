package event

import (
	"encoding/json"

	"github.com/vsayfb/gig-platform-notification-lambda/internal/notification"
)

// Envelope is the outer shape of every message a business service publishes
// to SQS: a type tag plus the type-specific payload. The handler unmarshals
// only the envelope; the payload is decoded by the renderer registered for
// its Type.
type Envelope struct {
	Type    notification.NotificationEventType `json:"type"`
	Payload json.RawMessage                    `json:"payload"`
}

