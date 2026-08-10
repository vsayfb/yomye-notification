package service

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/vsayfb/gig-platform-notification-lambda/internal/event"
	"github.com/vsayfb/gig-platform-notification-lambda/internal/fcm"
	"github.com/vsayfb/gig-platform-notification-lambda/internal/notification"
	"github.com/vsayfb/gig-platform-notification-lambda/internal/renderer"
)

func TestHandleOneDeletesUnregisteredTokensAndMarksPushed(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	notifRepo := &fakeNotificationRepository{}
	tokenRepo := &fakeTokenRepository{
		tokens: []string{"active-token", "dead-token"},
	}
	pushProvider := &fakePushProvider{
		result: notification.PushResult{
			UnregisteredTokens: []string{"dead-token"},
		},
	}
	svc := New(nil, notifRepo, tokenRepo, pushProvider, nil)

	err := svc.handleOne(context.Background(), fakeRenderer{userID: userID}, json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("handleOne() error = %v", err)
	}

	if !slices.Equal(tokenRepo.deleted, []string{"dead-token"}) {
		t.Errorf("deleted tokens = %v, want [dead-token]", tokenRepo.deleted)
	}
	if tokenRepo.deletedFor != userID {
		t.Errorf("deleted token user = %v, want %v", tokenRepo.deletedFor, userID)
	}
	if !notifRepo.markedPushed {
		t.Error("notification was not marked pushed")
	}
}

func TestHandleOneDeletesUnregisteredTokensButRetriesTransientFailure(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	notifRepo := &fakeNotificationRepository{}
	tokenRepo := &fakeTokenRepository{
		tokens: []string{"dead-token", "temporary-failure-token"},
	}
	pushProvider := &fakePushProvider{
		result: notification.PushResult{
			UnregisteredTokens: []string{"dead-token"},
		},
		err: errors.New("FCM unavailable"),
	}
	svc := New(nil, notifRepo, tokenRepo, pushProvider, nil)

	err := svc.handleOne(context.Background(), fakeRenderer{userID: userID}, json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("handleOne() error = nil, want transient send error")
	}

	if !slices.Equal(tokenRepo.deleted, []string{"dead-token"}) {
		t.Errorf("deleted tokens = %v, want [dead-token]", tokenRepo.deleted)
	}
	if notifRepo.markedPushed {
		t.Error("notification marked pushed despite transient failure")
	}
}

func TestMalformedRendererPayloadIsNonRetryable(t *testing.T) {
	t.Parallel()

	dispatcher := NewDispatcher()
	dispatcher.Register(notification.EventNewMessage, fakeRenderer{err: errors.New("private payload detail")})
	svc := New(dispatcher, nil, nil, nil, nil)

	err := svc.Handle(context.Background(), event.Envelope{
		Type:    notification.EventNewMessage,
		Payload: json.RawMessage(`{}`),
	})
	code, ok := NonRetryableCode(err)
	if !ok || code != "malformed_event_payload" {
		t.Fatalf("NonRetryableCode() = %q, %v, want malformed_event_payload, true; err=%v", code, ok, err)
	}
}

func TestUnknownEventTypeIsNonRetryable(t *testing.T) {
	t.Parallel()

	svc := New(NewDispatcher(), nil, nil, nil, nil)
	err := svc.Handle(context.Background(), event.Envelope{Type: "future_event"})
	code, ok := NonRetryableCode(err)
	if !ok || code != "unknown_event_type" {
		t.Fatalf("NonRetryableCode() = %q, %v, want unknown_event_type, true; err=%v", code, ok, err)
	}
}

type fakeRenderer struct {
	userID uuid.UUID
	err    error
}

var _ renderer.Renderer = fakeRenderer{}

func (r fakeRenderer) Render(
	context.Context,
	json.RawMessage,
) (*notification.Notification, *notification.PushNotification, error) {
	if r.err != nil {
		return nil, nil, r.err
	}
	return notification.NewNotification(
			r.userID,
			nil,
			"test",
			"test",
			uuid.NewString(),
			"title",
			"body",
			nil,
		), &notification.PushNotification{
			RecipientID: r.userID.String(),
			Title:       "title",
			Body:        "body",
		}, nil
}

type fakeNotificationRepository struct {
	markedPushed bool
}

func (r *fakeNotificationRepository) Create(
	_ context.Context,
	n *notification.Notification,
) (*notification.Notification, error) {
	return n, nil
}

func (r *fakeNotificationRepository) MarkPushed(
	context.Context,
	uuid.UUID,
	time.Time,
) error {
	r.markedPushed = true
	return nil
}

type fakeTokenRepository struct {
	tokens     []string
	deleted    []string
	deletedFor uuid.UUID
}

var _ fcm.TokenRepository = (*fakeTokenRepository)(nil)

func (r *fakeTokenRepository) ListByUserID(context.Context, uuid.UUID) ([]string, error) {
	return r.tokens, nil
}

func (r *fakeTokenRepository) DeleteByUserID(
	_ context.Context,
	userID uuid.UUID,
	tokens []string,
) error {
	r.deletedFor = userID
	r.deleted = append(r.deleted, tokens...)
	return nil
}

type fakePushProvider struct {
	result notification.PushResult
	err    error
}

func (p *fakePushProvider) Send(
	context.Context,
	[]string,
	*notification.PushNotification,
) (notification.PushResult, error) {
	return p.result, p.err
}
