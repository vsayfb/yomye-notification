package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/vsayfb/gig-platform-notification-lambda/internal/event"
	"github.com/vsayfb/gig-platform-notification-lambda/internal/subscriber"
)

func TestGigCategoryMatchedResolverUsesInternationalLocation(t *testing.T) {
	t.Parallel()

	gigID := uuid.New()
	categoryID := uuid.New()
	countryID := uuid.New()
	placeID := uuid.New()
	recipientID := uuid.New()
	repo := &fakeSubscriberRepository{
		subscribers: []subscriber.Subscriber{{ID: recipientID}},
	}
	resolver := NewGigCategoryMatchedResolver(repo)
	payload := json.RawMessage(`{
		"gig_id":"` + gigID.String() + `",
		"category_id":"` + categoryID.String() + `",
		"country_id":"` + countryID.String() + `",
		"place_id":"` + placeID.String() + `",
		"occurred_at":"2026-08-02T00:00:00Z"
	}`)

	resolved, err := resolver.Resolve(context.Background(), payload)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if repo.categoryID != categoryID {
		t.Errorf("repository category_id = %v, want %v", repo.categoryID, categoryID)
	}
	if repo.countryID == nil || *repo.countryID != countryID {
		t.Errorf("repository country_id = %v, want %v", repo.countryID, countryID)
	}
	if repo.placeID == nil || *repo.placeID != placeID {
		t.Errorf("repository place_id = %v, want %v", repo.placeID, placeID)
	}
	if len(resolved) != 1 {
		t.Fatalf("resolved payload count = %d, want 1", len(resolved))
	}

	var got event.GigCategoryMatchedNotificationEvent
	if err := json.Unmarshal(resolved[0], &got); err != nil {
		t.Fatalf("unmarshal resolved payload: %v", err)
	}
	if got.CountryID == nil || *got.CountryID != countryID {
		t.Errorf("resolved country_id = %v, want %v", got.CountryID, countryID)
	}
	if got.PlaceID == nil || *got.PlaceID != placeID {
		t.Errorf("resolved place_id = %v, want %v", got.PlaceID, placeID)
	}
	if strings.Contains(string(resolved[0]), "city_id") || strings.Contains(string(resolved[0]), "district_id") {
		t.Fatalf("resolved payload contains a legacy location field: %s", resolved[0])
	}
}

func TestGigCategoryMatchedResolverAllowsMissingLocation(t *testing.T) {
	t.Parallel()

	repo := &fakeSubscriberRepository{}
	resolver := NewGigCategoryMatchedResolver(repo)
	payload := json.RawMessage(`{
		"gig_id":"` + uuid.NewString() + `",
		"category_id":"` + uuid.NewString() + `",
		"occurred_at":"2026-08-02T00:00:00Z"
	}`)

	if _, err := resolver.Resolve(context.Background(), payload); err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if repo.countryID != nil || repo.placeID != nil {
		t.Errorf("repository location = (%v, %v), want (nil, nil)", repo.countryID, repo.placeID)
	}
}

func TestGigCategoryMatchedResolverRejectsPlaceWithoutCountry(t *testing.T) {
	t.Parallel()

	resolver := NewGigCategoryMatchedResolver(&fakeSubscriberRepository{})
	payload := json.RawMessage(`{
		"gig_id":"` + uuid.NewString() + `",
		"category_id":"` + uuid.NewString() + `",
		"place_id":"` + uuid.NewString() + `"
	}`)

	_, err := resolver.Resolve(context.Background(), payload)
	if err == nil || !strings.Contains(err.Error(), "place_id requires country_id") {
		t.Fatalf("Resolve() error = %v, want place/country validation error", err)
	}
}

func TestGigCategoryMatchedResolverRejectsLegacyLocation(t *testing.T) {
	t.Parallel()

	resolver := NewGigCategoryMatchedResolver(&fakeSubscriberRepository{})
	payload := json.RawMessage(`{
		"gig_id":"` + uuid.NewString() + `",
		"category_id":"` + uuid.NewString() + `",
		"city_id":7
	}`)

	_, err := resolver.Resolve(context.Background(), payload)
	if err == nil || !strings.Contains(err.Error(), "legacy city_id") {
		t.Fatalf("Resolve() error = %v, want legacy location error", err)
	}
}

type fakeSubscriberRepository struct {
	subscribers []subscriber.Subscriber
	categoryID  uuid.UUID
	countryID   *uuid.UUID
	placeID     *uuid.UUID
}

func (r *fakeSubscriberRepository) ListByCategoryAndLocation(
	_ context.Context,
	categoryID uuid.UUID,
	countryID *uuid.UUID,
	placeID *uuid.UUID,
) ([]subscriber.Subscriber, error) {
	r.categoryID = categoryID
	r.countryID = countryID
	r.placeID = placeID
	return r.subscribers, nil
}
