package renderer

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/vsayfb/gig-platform-notification-lambda/internal/notification"
)

func TestListingApprovedRenderer(t *testing.T) {
	t.Parallel()

	payload := json.RawMessage(`{
		"recipient_id": "74b061c1-502f-4eb6-83bf-ce9b8df155d7",
		"listing_id": "24d97762-95d5-4a3d-af75-e3c218d66677",
		"base_category_slug": "gig",
		"title": "Elektrikçi Aranıyor",
		"occurred_at": "2026-07-25T12:30:00Z"
	}`)

	got, push, err := NewListingApprovedRenderer().Render(context.Background(), payload)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if got.Type != notification.TypeListingApproved {
		t.Errorf("notification type = %q, want %q", got.Type, notification.TypeListingApproved)
	}
	if got.EntityType != notification.EntityTypeListing {
		t.Errorf("entity type = %q, want %q", got.EntityType, notification.EntityTypeListing)
	}
	if got.EntityID != "24d97762-95d5-4a3d-af75-e3c218d66677" {
		t.Errorf("entity ID = %q", got.EntityID)
	}
	if got.ActorID != nil {
		t.Errorf("actor ID = %v, want nil", got.ActorID)
	}
	if got.Metadata["listing_id"] != got.EntityID {
		t.Errorf("metadata listing_id = %v, want %q", got.Metadata["listing_id"], got.EntityID)
	}
	if got.Metadata["base_category_slug"] != notification.BaseCategoryGig {
		t.Errorf(
			"metadata base_category_slug = %v, want %q",
			got.Metadata["base_category_slug"],
			notification.BaseCategoryGig,
		)
	}
	if got.Metadata["localization_key"] != notification.LocalizationKeyListingApproved {
		t.Errorf("metadata localization_key = %v", got.Metadata["localization_key"])
	}
	if push.Data["entity_type"] != notification.EntityTypeListing {
		t.Errorf("push entity_type = %q, want %q", push.Data["entity_type"], notification.EntityTypeListing)
	}
	if push.Data["entity_id"] != got.EntityID {
		t.Errorf("push entity_id = %q, want %q", push.Data["entity_id"], got.EntityID)
	}
	if push.Data["localization_key"] != notification.LocalizationKeyListingApproved {
		t.Errorf("push localization_key = %q", push.Data["localization_key"])
	}
	if push.Data["localization_args"] != `{"title":"Elektrikçi Aranıyor"}` {
		t.Errorf("push localization_args = %q", push.Data["localization_args"])
	}
	if _, exists := push.Data["listing_id"]; exists {
		t.Error("push contains redundant legacy listing_id")
	}
	if _, exists := push.Data["base_category_slug"]; exists {
		t.Error("push contains unnecessary base_category_slug discriminator")
	}
}

func TestListingApprovedRendererRejectsForSale(t *testing.T) {
	t.Parallel()

	payload := json.RawMessage(`{
		"recipient_id": "74b061c1-502f-4eb6-83bf-ce9b8df155d7",
		"listing_id": "24d97762-95d5-4a3d-af75-e3c218d66677",
		"base_category_slug": "for-sale",
		"title": "Bisiklet",
		"occurred_at": "2026-07-25T12:30:00Z"
	}`)

	if _, _, err := NewListingApprovedRenderer().Render(context.Background(), payload); err == nil {
		t.Fatal("Render() error = nil, want gig-only scope error")
	}
}

func TestListingApprovedRendererRejectsUnknownFamily(t *testing.T) {
	t.Parallel()

	payload := json.RawMessage(`{
		"recipient_id": "74b061c1-502f-4eb6-83bf-ce9b8df155d7",
		"listing_id": "24d97762-95d5-4a3d-af75-e3c218d66677",
		"base_category_slug": "unknown",
		"title": "Listing",
		"occurred_at": "2026-07-25T12:30:00Z"
	}`)

	if _, _, err := NewListingApprovedRenderer().Render(context.Background(), payload); err == nil {
		t.Fatal("Render() error = nil, want unsupported family error")
	}
}
