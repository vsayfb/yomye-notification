package notification

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewSemanticPush(t *testing.T) {
	t.Parallel()

	n := NewNotification(
		uuid.New(),
		nil,
		TypeListingApproved,
		EntityTypeListing,
		uuid.NewString(),
		"legacy title",
		"legacy body",
		nil,
	)

	push, err := NewSemanticPush(
		n.UserID.String(),
		n,
		LocalizationKeyListingApproved,
		map[string]string{"title": "Elektrikçi Aranıyor"},
		n.Title,
		n.Body,
		nil,
	)
	if err != nil {
		t.Fatalf("NewSemanticPush() error = %v", err)
	}

	want := map[string]string{
		"type":              TypeListingApproved,
		"entity_type":       EntityTypeListing,
		"entity_id":         n.EntityID,
		"localization_key":  LocalizationKeyListingApproved,
		"localization_args": `{"title":"Elektrikçi Aranıyor"}`,
	}
	for key, value := range want {
		if push.Data[key] != value {
			t.Errorf("data[%q] = %q, want %q", key, push.Data[key], value)
		}
	}
}

func TestNewSemanticPushRejectsReservedRoutingOverride(t *testing.T) {
	t.Parallel()

	n := NewNotification(
		uuid.New(), nil, "type", "entity", "id", "title", "body", nil,
	)

	_, err := NewSemanticPush(
		n.UserID.String(),
		n,
		"notifications.test",
		nil,
		n.Title,
		n.Body,
		map[string]string{"entity_id": "different"},
	)
	if err == nil {
		t.Fatal("NewSemanticPush() error = nil, want reserved-field error")
	}
}

func TestAddLocalizationMetadataUsesEmptyArgsObject(t *testing.T) {
	t.Parallel()

	metadata := AddLocalizationMetadata(nil, "notifications.test", nil)
	args, ok := metadata["localization_args"].(map[string]string)
	if !ok {
		t.Fatalf("localization_args type = %T, want map[string]string", metadata["localization_args"])
	}
	if len(args) != 0 {
		t.Errorf("localization_args = %v, want empty object", args)
	}
}
