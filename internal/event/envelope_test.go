package event

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vsayfb/gig-platform-notification-lambda/internal/notification"
)

func TestEnvelopeParsesStableEventMetadata(t *testing.T) {
	t.Parallel()

	eventID := uuid.MustParse("11111111-1111-5111-8111-111111111111")
	occurredAt := time.Date(2026, time.August, 2, 0, 0, 0, 0, time.UTC)
	raw := []byte(`{
		"type":"listing_approved",
		"event_id":"` + eventID.String() + `",
		"version":1,
		"occurred_at":"2026-08-02T00:00:00Z",
		"payload":{}
	}`)

	var envelope Envelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if envelope.Type != notification.EventListingApproved {
		t.Errorf("type = %q", envelope.Type)
	}
	if envelope.EventID != eventID {
		t.Errorf("event_id = %v, want %v", envelope.EventID, eventID)
	}
	if envelope.Version != CurrentVersion {
		t.Errorf("version = %d, want %d", envelope.Version, CurrentVersion)
	}
	if !envelope.OccurredAt.Equal(occurredAt) {
		t.Errorf("occurred_at = %v, want %v", envelope.OccurredAt, occurredAt)
	}
}

func TestWorkerEventsRequireVersionedEnvelope(t *testing.T) {
	t.Parallel()

	for _, eventType := range []notification.NotificationEventType{
		notification.EventListingApproved,
		notification.EventGigCategoryMatched,
	} {
		if !RequiresVersionedEnvelope(eventType) {
			t.Errorf("RequiresVersionedEnvelope(%q) = false, want true", eventType)
		}
	}
	if RequiresVersionedEnvelope(notification.EventNewMessage) {
		t.Error("new_message unexpectedly requires worker envelope metadata")
	}
}
