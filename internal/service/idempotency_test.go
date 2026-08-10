package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/vsayfb/gig-platform-notification-lambda/internal/event"
	"github.com/vsayfb/gig-platform-notification-lambda/internal/notification"
)

func TestHandleCompletesVersionedEventAfterSuccessfulWork(t *testing.T) {
	t.Parallel()

	eventID := uuid.New()
	deduplicator := &fakeDeduplicator{status: event.ClaimAcquired}
	dispatcher := NewDispatcher()
	dispatcher.Register(notification.EventListingApproved, fakeRenderer{userID: uuid.New()})
	notifRepo := &fakeNotificationRepository{}
	svc := New(
		dispatcher,
		notifRepo,
		&fakeTokenRepository{},
		&fakePushProvider{},
		deduplicator,
	)

	err := svc.Handle(context.Background(), event.Envelope{
		Type:    notification.EventListingApproved,
		EventID: eventID,
		Version: event.CurrentVersion,
		Payload: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if deduplicator.claimedEventID != eventID {
		t.Errorf("claimed event_id = %v, want %v", deduplicator.claimedEventID, eventID)
	}
	if deduplicator.claimedVersion != event.CurrentVersion {
		t.Errorf("claimed version = %d, want %d", deduplicator.claimedVersion, event.CurrentVersion)
	}
	if !deduplicator.completed {
		t.Error("event was not marked processed")
	}
	if deduplicator.released {
		t.Error("successful event claim was released")
	}
}

func TestHandleUsesNewMessageIDForConcurrentSafeClaim(t *testing.T) {
	t.Parallel()

	payload := json.RawMessage(`{"message_id":"64b7f2c1a3e5d7890abc5678"}`)
	wantClaimID, err := event.NewMessageClaimID(payload)
	if err != nil {
		t.Fatalf("NewMessageClaimID() error = %v", err)
	}
	deduplicator := &fakeDeduplicator{status: event.ClaimAlreadyProcessed}
	svc := New(nil, nil, nil, nil, deduplicator)

	err = svc.Handle(context.Background(), event.Envelope{
		Type:    notification.EventNewMessage,
		Payload: payload,
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if deduplicator.claimedEventID != wantClaimID {
		t.Errorf("claimed event_id = %s, want %s derived from message_id", deduplicator.claimedEventID, wantClaimID)
	}
	if deduplicator.claimedVersion != event.CurrentVersion {
		t.Errorf("claimed version = %d, want %d", deduplicator.claimedVersion, event.CurrentVersion)
	}
}

func TestHandleAcknowledgesAlreadyProcessedEventWithoutSideEffects(t *testing.T) {
	t.Parallel()

	deduplicator := &fakeDeduplicator{status: event.ClaimAlreadyProcessed}
	svc := New(nil, nil, nil, nil, deduplicator)

	err := svc.Handle(context.Background(), event.Envelope{
		Type:    notification.EventGigCategoryMatched,
		EventID: uuid.New(),
		Version: event.CurrentVersion,
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if deduplicator.completed || deduplicator.released {
		t.Error("already processed event changed its claim state")
	}
}

func TestHandleRetriesEventAlreadyInProgressWithoutSideEffects(t *testing.T) {
	t.Parallel()

	deduplicator := &fakeDeduplicator{status: event.ClaimInProgress}
	svc := New(nil, nil, nil, nil, deduplicator)

	err := svc.Handle(context.Background(), event.Envelope{
		Type:    notification.EventListingApproved,
		EventID: uuid.New(),
		Version: event.CurrentVersion,
	})
	if err == nil || !strings.Contains(err.Error(), "already being processed") {
		t.Fatalf("Handle() error = %v, want in-progress retry error", err)
	}
	if deduplicator.completed || deduplicator.released {
		t.Error("in-progress duplicate changed claim state")
	}
}

func TestHandleReleasesClaimWhenWorkFails(t *testing.T) {
	t.Parallel()

	deduplicator := &fakeDeduplicator{status: event.ClaimAcquired}
	svc := New(NewDispatcher(), nil, nil, nil, deduplicator)

	err := svc.Handle(context.Background(), event.Envelope{
		Type:    notification.EventListingApproved,
		EventID: uuid.New(),
		Version: event.CurrentVersion,
	})
	if err == nil {
		t.Fatal("Handle() error = nil, want missing renderer error")
	}
	if !deduplicator.released {
		t.Error("failed event claim was not released")
	}
	if deduplicator.completed {
		t.Error("failed event was marked processed")
	}
}

func TestHandleRejectsUnsupportedVersionBeforeClaim(t *testing.T) {
	t.Parallel()

	deduplicator := &fakeDeduplicator{status: event.ClaimAcquired}
	svc := New(nil, nil, nil, nil, deduplicator)

	err := svc.Handle(context.Background(), event.Envelope{
		Type:    notification.EventListingApproved,
		EventID: uuid.New(),
		Version: 2,
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported version") {
		t.Fatalf("Handle() error = %v, want unsupported version error", err)
	}
	if deduplicator.claimedEventID != uuid.Nil {
		t.Error("unsupported event version was claimed")
	}
}

func TestHandleRejectsWorkerEventWithoutEventID(t *testing.T) {
	t.Parallel()

	deduplicator := &fakeDeduplicator{status: event.ClaimAcquired}
	svc := New(nil, nil, nil, nil, deduplicator)

	err := svc.Handle(context.Background(), event.Envelope{
		Type:    notification.EventGigCategoryMatched,
		Version: event.CurrentVersion,
	})
	if err == nil || !strings.Contains(err.Error(), "event_id is required") {
		t.Fatalf("Handle() error = %v, want event_id validation error", err)
	}
}

type fakeDeduplicator struct {
	status         event.ClaimStatus
	err            error
	claimedEventID uuid.UUID
	claimedType    notification.NotificationEventType
	claimedVersion int
	claimToken     uuid.UUID
	completed      bool
	released       bool
}

func (d *fakeDeduplicator) Claim(
	_ context.Context,
	eventID uuid.UUID,
	eventType notification.NotificationEventType,
	version int,
	claimToken uuid.UUID,
) (event.ClaimStatus, error) {
	d.claimedEventID = eventID
	d.claimedType = eventType
	d.claimedVersion = version
	d.claimToken = claimToken
	return d.status, d.err
}

func (d *fakeDeduplicator) Complete(_ context.Context, eventID, claimToken uuid.UUID) error {
	if eventID != d.claimedEventID || claimToken != d.claimToken {
		return context.Canceled
	}
	d.completed = true
	return nil
}

func (d *fakeDeduplicator) Release(_ context.Context, eventID, claimToken uuid.UUID) error {
	if eventID != d.claimedEventID || claimToken != d.claimToken {
		return context.Canceled
	}
	d.released = true
	return nil
}
