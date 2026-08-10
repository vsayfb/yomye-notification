package handler

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	awsevents "github.com/aws/aws-lambda-go/events"
	"github.com/google/uuid"

	"github.com/vsayfb/gig-platform-notification-lambda/internal/notification"
	"github.com/vsayfb/gig-platform-notification-lambda/internal/renderer"
	"github.com/vsayfb/gig-platform-notification-lambda/internal/service"
)

func TestHandleReturnsOnlyRetryableRecordsAsBatchFailures(t *testing.T) {
	t.Parallel()

	dispatcher := service.NewDispatcher()
	dispatcher.Register(notification.NotificationEventType("retryable_test"), handlerFakeRenderer{})
	svc := service.New(dispatcher, handlerFailingNotificationRepository{}, nil, nil, nil)
	h := New(svc)

	response, err := h.Handle(context.Background(), awsevents.SQSEvent{Records: []awsevents.SQSMessage{
		{MessageId: "malformed-envelope", Body: `{`},
		{MessageId: "unknown-event", Body: `{"type":"unknown","payload":{}}`},
		{MessageId: "temporary-db-failure", Body: `{"type":"retryable_test","payload":{}}`},
	}})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if len(response.BatchItemFailures) != 1 || response.BatchItemFailures[0].ItemIdentifier != "temporary-db-failure" {
		t.Fatalf("batch failures = %#v, want only temporary-db-failure", response.BatchItemFailures)
	}
}

type handlerFakeRenderer struct{}

var _ renderer.Renderer = handlerFakeRenderer{}

func (handlerFakeRenderer) Render(
	context.Context,
	json.RawMessage,
) (*notification.Notification, *notification.PushNotification, error) {
	userID := uuid.New()
	n := notification.NewNotification(userID, nil, "test", "test", "entity", "title", "body", nil)
	return n, &notification.PushNotification{RecipientID: userID.String()}, nil
}

type handlerFailingNotificationRepository struct{}

var _ notification.NotificationRepository = handlerFailingNotificationRepository{}

func (handlerFailingNotificationRepository) Create(
	context.Context,
	*notification.Notification,
) (*notification.Notification, error) {
	return nil, errors.New("temporary database failure")
}

func (handlerFailingNotificationRepository) MarkPushed(context.Context, uuid.UUID, time.Time) error {
	return nil
}
