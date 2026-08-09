package event

import (
	"context"

	"github.com/google/uuid"
	"github.com/vsayfb/gig-platform-notification-lambda/internal/notification"
)

type ClaimStatus uint8

const (
	ClaimAcquired ClaimStatus = iota + 1
	ClaimAlreadyProcessed
	ClaimInProgress
)

// Deduplicator coordinates at-least-once domain events. A claim is not a
// processed marker: Complete is called only after every required side effect
// succeeds, while Release makes an ordinary failure immediately retryable.
type Deduplicator interface {
	Claim(
		ctx context.Context,
		eventID uuid.UUID,
		eventType notification.NotificationEventType,
		version int,
		claimToken uuid.UUID,
	) (ClaimStatus, error)
	Complete(ctx context.Context, eventID, claimToken uuid.UUID) error
	Release(ctx context.Context, eventID, claimToken uuid.UUID) error
}
