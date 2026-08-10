package renderer

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/vsayfb/gig-platform-notification-lambda/internal/event"
	"github.com/vsayfb/gig-platform-notification-lambda/internal/notification"
)

func TestNewMessageRendererEmitsSemanticPayload(t *testing.T) {
	t.Parallel()

	payload := json.RawMessage(`{
		"recipient_id":"` + uuid.NewString() + `",
		"sender_id":"` + uuid.NewString() + `",
		"sender_name":"Alex",
		"thread_id":"64b7f2c1a3e5d7890abc1234",
		"gig_id":"` + uuid.NewString() + `",
		"message_id":"64b7f2c1a3e5d7890abc5678",
		"message_preview":"Hello",
		"occurred_at":"2026-08-10T12:34:56Z"
	}`)

	n, push, err := NewNewMessageRenderer().Render(context.Background(), payload)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	assertSemanticPayload(
		t,
		n,
		push,
		notification.LocalizationKeyMessageReceived,
		`{"name":"Alex"}`,
	)
	if n.SourceEventID == nil || *n.SourceEventID != "64b7f2c1a3e5d7890abc5678" {
		t.Errorf("source_event_id = %v, want message_id", n.SourceEventID)
	}
	if n.EntityID != "64b7f2c1a3e5d7890abc1234" {
		t.Errorf("entity_id = %q, want thread_id", n.EntityID)
	}
	if push.Data["message_id"] != "64b7f2c1a3e5d7890abc5678" {
		t.Errorf("push message_id = %q", push.Data["message_id"])
	}
	if n.Metadata["thread_id"] != n.EntityID || push.Data["thread_id"] != n.EntityID {
		t.Errorf("thread routing metadata=%v push=%q entity=%q", n.Metadata["thread_id"], push.Data["thread_id"], n.EntityID)
	}
}

func TestNewMessageRendererRoutesDirectMessageWithEmptyGigID(t *testing.T) {
	t.Parallel()

	payload := json.RawMessage(`{
		"recipient_id":"` + uuid.NewString() + `",
		"sender_id":"` + uuid.NewString() + `",
		"sender_name":"Alex",
		"thread_id":"64b7f2c1a3e5d7890abc1234",
		"gig_id":"",
		"message_id":"64b7f2c1a3e5d7890abc5678",
		"message_preview":"[Ek]",
		"occurred_at":"2026-08-10T12:34:56Z"
	}`)

	n, push, err := NewNewMessageRenderer().Render(context.Background(), payload)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if n.Metadata["gig_id"] != "" || push.Data["gig_id"] != "" {
		t.Errorf("direct-message gig_id metadata=%v push=%q, want empty", n.Metadata["gig_id"], push.Data["gig_id"])
	}
}

func TestNewMessageRendererRejectsSenderAsRecipient(t *testing.T) {
	t.Parallel()

	userID := uuid.NewString()
	payload := json.RawMessage(`{
		"recipient_id":"` + userID + `",
		"sender_id":"` + userID + `",
		"sender_name":"Alex",
		"thread_id":"64b7f2c1a3e5d7890abc1234",
		"gig_id":"",
		"message_id":"64b7f2c1a3e5d7890abc5678",
		"message_preview":"Hello",
		"occurred_at":"2026-08-10T12:34:56Z"
	}`)

	if _, _, err := NewNewMessageRenderer().Render(context.Background(), payload); err == nil {
		t.Fatal("Render() error = nil, want sender/recipient validation error")
	}
}

func TestGigCategoryMatchedRendererUsesListingSemanticTarget(t *testing.T) {
	t.Parallel()

	listingID := uuid.New()
	evt := event.GigCategoryMatchedNotificationEvent{
		RecipientID: uuid.New(),
		GigID:       listingID,
		CategoryID:  uuid.New(),
	}
	payload, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	n, push, err := NewGigCategoryMatchedRenderer().Render(context.Background(), payload)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if n.EntityType != notification.EntityTypeListing {
		t.Errorf("entity_type = %q, want %q", n.EntityType, notification.EntityTypeListing)
	}
	if n.EntityID != listingID.String() {
		t.Errorf("entity_id = %q, want %q", n.EntityID, listingID)
	}
	assertSemanticPayload(
		t,
		n,
		push,
		notification.LocalizationKeyGigCategoryMatched,
		`{}`,
	)
}

func assertSemanticPayload(
	t *testing.T,
	n *notification.Notification,
	push *notification.PushNotification,
	localizationKey string,
	encodedArgs string,
) {
	t.Helper()

	if n.Metadata["localization_key"] != localizationKey {
		t.Errorf("metadata localization_key = %v, want %q", n.Metadata["localization_key"], localizationKey)
	}
	if push.Data["type"] != n.Type {
		t.Errorf("push type = %q, want %q", push.Data["type"], n.Type)
	}
	if push.Data["entity_type"] != n.EntityType {
		t.Errorf("push entity_type = %q, want %q", push.Data["entity_type"], n.EntityType)
	}
	if push.Data["entity_id"] != n.EntityID {
		t.Errorf("push entity_id = %q, want %q", push.Data["entity_id"], n.EntityID)
	}
	if push.Data["localization_key"] != localizationKey {
		t.Errorf("push localization_key = %q, want %q", push.Data["localization_key"], localizationKey)
	}
	if push.Data["localization_args"] != encodedArgs {
		t.Errorf("push localization_args = %q, want %q", push.Data["localization_args"], encodedArgs)
	}
}
