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
		"thread_id":"thread-123",
		"gig_id":"listing-123",
		"message_id":"message-123",
		"message_preview":"Hello"
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
