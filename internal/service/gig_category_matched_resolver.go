package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

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
	var evt event.GigCategoryMatchedEvent

	if err := json.Unmarshal(payload, &evt); err != nil {
		return nil, fmt.Errorf("gig_category_matched: unmarshal payload: %w", err)
	}

	subs, err := r.subscribers.ListByCategoryAndCity(ctx, evt.CategoryID, subscriber.CityID(evt.CityID))

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
			CityID:      evt.CityID,
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
