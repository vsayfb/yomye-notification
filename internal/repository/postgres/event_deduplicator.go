package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vsayfb/gig-platform-notification-lambda/internal/event"
	"github.com/vsayfb/gig-platform-notification-lambda/internal/notification"
)

const eventClaimLease = 5 * time.Minute

type EventDeduplicator struct {
	pool *pgxpool.Pool
}

func NewEventDeduplicator(pool *pgxpool.Pool) *EventDeduplicator {
	return &EventDeduplicator{pool: pool}
}

var _ event.Deduplicator = (*EventDeduplicator)(nil)

func (r *EventDeduplicator) Claim(
	ctx context.Context,
	eventID uuid.UUID,
	eventType notification.NotificationEventType,
	version int,
	claimToken uuid.UUID,
) (event.ClaimStatus, error) {
	const claimQuery = `
		INSERT INTO notification_processed_events
			(event_id, event_type, event_version, claim_token, claimed_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (event_id) DO UPDATE
		SET claim_token = EXCLUDED.claim_token,
			claimed_at = NOW()
		WHERE notification_processed_events.processed_at IS NULL
		  AND notification_processed_events.claimed_at < NOW() - make_interval(secs => $5)
		  AND notification_processed_events.event_type = EXCLUDED.event_type
		  AND notification_processed_events.event_version = EXCLUDED.event_version
		RETURNING event_type, event_version
	`

	var storedType notification.NotificationEventType
	var storedVersion int
	err := r.pool.QueryRow(
		ctx,
		claimQuery,
		eventID,
		eventType,
		version,
		claimToken,
		int(eventClaimLease.Seconds()),
	).Scan(&storedType, &storedVersion)
	if err == nil {
		if storedType != eventType || storedVersion != version {
			return 0, fmt.Errorf(
				"event deduplicator: event_id %s contract mismatch: stored %s/v%d, received %s/v%d",
				eventID,
				storedType,
				storedVersion,
				eventType,
				version,
			)
		}
		return event.ClaimAcquired, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("event deduplicator: claim: %w", err)
	}

	const stateQuery = `
		SELECT event_type, event_version, processed_at
		FROM notification_processed_events
		WHERE event_id = $1
	`
	var processedAt *time.Time
	if err := r.pool.QueryRow(ctx, stateQuery, eventID).Scan(
		&storedType,
		&storedVersion,
		&processedAt,
	); err != nil {
		return 0, fmt.Errorf("event deduplicator: read existing claim: %w", err)
	}
	if storedType != eventType || storedVersion != version {
		return 0, fmt.Errorf(
			"event deduplicator: event_id %s contract mismatch: stored %s/v%d, received %s/v%d",
			eventID,
			storedType,
			storedVersion,
			eventType,
			version,
		)
	}
	if processedAt != nil {
		return event.ClaimAlreadyProcessed, nil
	}
	return event.ClaimInProgress, nil
}

func (r *EventDeduplicator) Complete(ctx context.Context, eventID, claimToken uuid.UUID) error {
	const query = `
		UPDATE notification_processed_events
		SET processed_at = NOW()
		WHERE event_id = $1
		  AND claim_token = $2
		  AND processed_at IS NULL
	`
	tag, err := r.pool.Exec(ctx, query, eventID, claimToken)
	if err != nil {
		return fmt.Errorf("event deduplicator: complete: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("event deduplicator: complete: claim is no longer owned")
	}
	return nil
}

func (r *EventDeduplicator) Release(ctx context.Context, eventID, claimToken uuid.UUID) error {
	const query = `
		DELETE FROM notification_processed_events
		WHERE event_id = $1
		  AND claim_token = $2
		  AND processed_at IS NULL
	`
	if _, err := r.pool.Exec(ctx, query, eventID, claimToken); err != nil {
		return fmt.Errorf("event deduplicator: release: %w", err)
	}
	return nil
}
