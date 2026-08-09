package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync/atomic"

	"github.com/aws/aws-lambda-go/events"
	"golang.org/x/sync/errgroup"

	"github.com/vsayfb/gig-platform-notification-lambda/internal/event"
	"github.com/vsayfb/gig-platform-notification-lambda/internal/service"
)

// maxConcurrentRecords bounds how many SQS records are processed at once,
// so a full batch can't open more DB/FCM work than the pool is sized for
// while still finishing well inside the invocation's time budget.
const maxConcurrentRecords = 5

// Handler is transport-only: receive the SQS event, unmarshal each
// envelope, call the service. No SQL, no Firebase, no rendering.
type Handler struct {
	service *service.Service
}

func New(svc *service.Service) *Handler {
	return &Handler{service: svc}
}

func (h *Handler) Handle(ctx context.Context, sqsEvent events.SQSEvent) error {
	var g errgroup.Group
	g.SetLimit(maxConcurrentRecords)

	var failed atomic.Int32

	for _, record := range sqsEvent.Records {
		g.Go(func() error {
			var env event.Envelope

			if err := json.Unmarshal([]byte(record.Body), &env); err != nil {
				slog.Error("unmarshal envelope", "error", err, "message_id", record.MessageId)
				failed.Add(1)
				return nil // isolated: don't let one bad record cancel the rest of the batch
			}

			slog.InfoContext(
				ctx,
				"process notification event",
				"type", env.Type,
				"event_id", env.EventID,
				"version", env.Version,
				"message_id", record.MessageId,
			)

			if err := h.service.Handle(ctx, env); err != nil {
				slog.Error(
					"handle notification event",
					"error", err,
					"type", env.Type,
					"event_id", env.EventID,
					"version", env.Version,
					"message_id", record.MessageId,
				)
				failed.Add(1)
				return nil
			}

			slog.InfoContext(
				ctx,
				"notification event processed",
				"type", env.Type,
				"event_id", env.EventID,
				"version", env.Version,
				"message_id", record.MessageId,
			)

			return nil
		})
	}

	_ = g.Wait() // every goroutine above returns nil; failures are tracked via `failed`, not propagated as errors

	if n := failed.Load(); n > 0 {
		return fmt.Errorf("failed to process %d message(s)", n)
	}

	return nil
}
