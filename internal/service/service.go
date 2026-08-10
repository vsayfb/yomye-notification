package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/vsayfb/gig-platform-notification-lambda/internal/event"
	"github.com/vsayfb/gig-platform-notification-lambda/internal/fcm"
	"github.com/vsayfb/gig-platform-notification-lambda/internal/notification"
	"github.com/vsayfb/gig-platform-notification-lambda/internal/renderer"
)

// Service coordinates a notification event end to end. All
// notification-specific business logic lives in the renderers (and, for
// event types that need it, a resolver); Service only orchestrates.
type Service struct {
	dispatcher   *Dispatcher
	notifRepo    notification.NotificationRepository
	tokenRepo    fcm.TokenRepository
	pushProvider notification.PushProvider
	deduplicator event.Deduplicator
}

func New(
	dispatcher *Dispatcher,
	notifRepo notification.NotificationRepository,
	tokenRepo fcm.TokenRepository,
	pushProvider notification.PushProvider,
	deduplicator event.Deduplicator,
) *Service {
	return &Service{
		dispatcher:   dispatcher,
		notifRepo:    notifRepo,
		tokenRepo:    tokenRepo,
		pushProvider: pushProvider,
		deduplicator: deduplicator,
	}
}

// Handle processes one inbound envelope. If a fan-out resolver is
// registered for env.Type, the envelope's payload is first expanded into
// one payload per recipient (resolved live, e.g. against subscribers in
// the database); otherwise the payload is treated as already addressed to
// a single recipient, exactly as before. Either way, each resulting
// payload is rendered, persisted, and pushed independently, so one bad
// recipient (malformed id, DB hiccup, etc.) can't sink the rest.
func (s *Service) Handle(ctx context.Context, env event.Envelope) error {
	idempotencyID := env.EventID
	idempotencyVersion := env.Version
	versioned := event.RequiresVersionedEnvelope(env.Type) || env.EventID != uuid.Nil || env.Version != 0
	if env.Type == notification.EventNewMessage {
		claimID, err := event.NewMessageClaimID(env.Payload)
		if err != nil {
			return nonRetryable("malformed_event_payload", err)
		}
		idempotencyID = claimID
		idempotencyVersion = event.CurrentVersion
		versioned = true
	}
	if !versioned {
		return s.handleEvent(ctx, env)
	}
	if idempotencyID == uuid.Nil {
		return fmt.Errorf("event envelope: event_id is required for type %q", env.Type)
	}
	if idempotencyVersion != event.CurrentVersion {
		return fmt.Errorf(
			"event envelope: unsupported version %d for type %q; supported version is %d",
			idempotencyVersion,
			env.Type,
			event.CurrentVersion,
		)
	}
	if s.deduplicator == nil {
		return fmt.Errorf("event envelope: idempotency repository is not configured")
	}

	claimToken := uuid.New()
	claimStatus, err := s.deduplicator.Claim(
		ctx,
		idempotencyID,
		env.Type,
		idempotencyVersion,
		claimToken,
	)
	if err != nil {
		return fmt.Errorf("claim event %s: %w", idempotencyID, err)
	}
	switch claimStatus {
	case event.ClaimAlreadyProcessed:
		slog.InfoContext(
			ctx,
			"skip already processed event",
			"event_id",
			idempotencyID,
			"type",
			env.Type,
			"version",
			idempotencyVersion,
		)
		return nil
	case event.ClaimInProgress:
		return fmt.Errorf("event %s is already being processed", idempotencyID)
	case event.ClaimAcquired:
		slog.InfoContext(
			ctx,
			"event claim acquired",
			"event_id", idempotencyID,
			"type", env.Type,
			"version", idempotencyVersion,
		)
	default:
		return fmt.Errorf("event %s returned unknown claim status %d", idempotencyID, claimStatus)
	}

	if err := s.handleEvent(ctx, env); err != nil {
		releaseErr := s.deduplicator.Release(ctx, idempotencyID, claimToken)
		if releaseErr != nil {
			return errors.Join(err, fmt.Errorf("release event claim: %w", releaseErr))
		}
		return err
	}
	slog.InfoContext(
		ctx,
		"event work completed",
		"event_id", idempotencyID,
		"type", env.Type,
		"version", idempotencyVersion,
	)

	if err := s.deduplicator.Complete(ctx, idempotencyID, claimToken); err != nil {
		return fmt.Errorf("mark event %s processed: %w", idempotencyID, err)
	}
	slog.InfoContext(
		ctx,
		"event marked processed",
		"event_id", idempotencyID,
		"type", env.Type,
		"version", idempotencyVersion,
	)
	return nil
}

func (s *Service) handleEvent(ctx context.Context, env event.Envelope) error {
	r, err := s.dispatcher.Renderer(env.Type)

	if err != nil {
		return nonRetryable("unknown_event_type", err)
	}

	payloads, err := s.resolvePayloads(ctx, env)

	if err != nil {
		return fmt.Errorf("resolve recipients: %w", err)
	}

	var errs []error

	for _, payload := range payloads {
		if err := s.handleOne(ctx, r, payload); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("service: %d/%d recipient(s) failed: %w", len(errs), len(payloads), errors.Join(errs...))
	}

	return nil
}

func (s *Service) resolvePayloads(ctx context.Context, env event.Envelope) ([]json.RawMessage, error) {
	resolver, ok := s.dispatcher.FanOut(env.Type)

	if !ok {
		return []json.RawMessage{env.Payload}, nil
	}

	return resolver.Resolve(ctx, env.Payload)
}

func (s *Service) handleOne(ctx context.Context, r renderer.Renderer, payload json.RawMessage) error {
	n, push, err := r.Render(ctx, payload)

	if err != nil {
		return nonRetryable("malformed_event_payload", fmt.Errorf("render notification: %w", err))
	}

	persisted, err := s.notifRepo.Create(ctx, n)

	if err != nil {
		return fmt.Errorf("persist notification: %w", err)
	}

	if persisted.PushedAt != nil {
		// Already delivered for this (user, source event) — a
		// redelivered event shouldn't push twice. If the row already
		// existed but PushedAt is still nil, the previous attempt's
		// push never succeeded, so we fall through and retry it.
		return nil
	}

	tokens, err := s.tokenRepo.ListByUserID(ctx, persisted.UserID)

	if err != nil {
		return fmt.Errorf("lookup device tokens: %w", err)
	}

	if len(tokens) == 0 {
		return nil
	}

	result, sendErr := s.pushProvider.Send(ctx, tokens, push)

	if err := s.tokenRepo.DeleteByUserID(
		ctx,
		persisted.UserID,
		result.UnregisteredTokens,
	); err != nil {
		return fmt.Errorf("delete unregistered device tokens: %w", err)
	}

	if len(result.UnregisteredTokens) > 0 {
		slog.InfoContext(
			ctx,
			"deleted unregistered FCM tokens",
			"user_id",
			persisted.UserID,
			"count",
			len(result.UnregisteredTokens),
		)
	}

	if sendErr != nil {
		return fmt.Errorf("send push notification: %w", sendErr)
	}

	if err := s.notifRepo.MarkPushed(ctx, persisted.ID, time.Now().UTC()); err != nil {
		return fmt.Errorf("mark notification pushed: %w", err)
	}

	return nil
}
