package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/vsayfb/gig-platform-notification-lambda/internal/event"
	"github.com/vsayfb/gig-platform-notification-lambda/internal/subscriber"
)

// GigCategoryMatchedResolver turns a single GigCategoryMatchedEvent — which
// carries no resolved recipients — into one per-recipient payload per
// currently-matched subscriber. The match is resolved live against the
// database on every consume, never from a snapshot, so someone who
// subscribes or unsubscribes between publish and consume gets the current
// answer rather than a stale one.
type GigCategoryMatchedResolver struct {
	subscribers subscriber.Repository
}

func NewGigCategoryMatchedResolver(subscribers subscriber.Repository) *GigCategoryMatchedResolver {
	return &GigCategoryMatchedResolver{subscribers: subscribers}
}

var _ Resolver = (*GigCategoryMatchedResolver)(nil)

func (r *GigCategoryMatchedResolver) Resolve(ctx context.Context, payload json.RawMessage) ([]json.RawMessage, error) {
	var rawFields map[string]json.RawMessage
	if err := json.Unmarshal(payload, &rawFields); err != nil {
		return nil, fmt.Errorf("gig_category_matched: unmarshal payload: %w", err)
	}
	if _, exists := rawFields["city_id"]; exists {
		return nil, fmt.Errorf("gig_category_matched: legacy city_id is no longer supported")
	}
	if _, exists := rawFields["district_id"]; exists {
		return nil, fmt.Errorf("gig_category_matched: legacy district_id is no longer supported")
	}

	var evt event.GigCategoryMatchedEvent

	if err := json.Unmarshal(payload, &evt); err != nil {
		return nil, fmt.Errorf("gig_category_matched: unmarshal payload: %w", err)
	}

	if evt.GigID == uuid.Nil {
		return nil, fmt.Errorf("gig_category_matched: gig_id is required")
	}
	if evt.CategoryID == uuid.Nil {
		return nil, fmt.Errorf("gig_category_matched: category_id is required")
	}
	if evt.CountryID != nil && *evt.CountryID == uuid.Nil {
		return nil, fmt.Errorf("gig_category_matched: country_id must not be nil UUID")
	}
	if evt.PlaceID != nil && *evt.PlaceID == uuid.Nil {
		return nil, fmt.Errorf("gig_category_matched: place_id must not be nil UUID")
	}
	if evt.PlaceID != nil && evt.CountryID == nil {
		return nil, fmt.Errorf("gig_category_matched: place_id requires country_id")
	}

	subs, err := r.subscribers.ListByCategoryAndLocation(
		ctx,
		evt.CategoryID,
		evt.CountryID,
		evt.PlaceID,
	)

	slog.InfoContext(ctx, "subscribers found for category", "subscribers", subs)

	if err != nil {
		return nil, fmt.Errorf("gig_category_matched: list subscribers: %w", err)
	}

	payloads := make([]json.RawMessage, 0, len(subs))

	for _, sub := range subs {
		perRecipient := event.GigCategoryMatchedNotificationEvent{
			RecipientID: sub.ID,
			GigID:       evt.GigID,
			CategoryID:  evt.CategoryID,
			CountryID:   evt.CountryID,
			PlaceID:     evt.PlaceID,
			OccurredAt:  evt.OccurredAt,
		}

		encoded, err := json.Marshal(perRecipient)

		if err != nil {
			return nil, fmt.Errorf("gig_category_matched: marshal recipient payload: %w", err)
		}

		payloads = append(payloads, encoded)
	}

	return payloads, nil
}
