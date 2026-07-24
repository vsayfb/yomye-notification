package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

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
}

func New(
	dispatcher *Dispatcher,
	notifRepo notification.NotificationRepository,
	tokenRepo fcm.TokenRepository,
	pushProvider notification.PushProvider,
) *Service {
	return &Service{
		dispatcher:   dispatcher,
		notifRepo:    notifRepo,
		tokenRepo:    tokenRepo,
		pushProvider: pushProvider,
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
	r, err := s.dispatcher.Renderer(env.Type)

	if err != nil {
		return err
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
		return fmt.Errorf("render notification: %w", err)
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

	if err := s.pushProvider.Send(ctx, tokens, push); err != nil {
		return fmt.Errorf("send push notification: %w", err)
	}

	if err := s.notifRepo.MarkPushed(ctx, persisted.ID, time.Now().UTC()); err != nil {
		return fmt.Errorf("mark notification pushed: %w", err)
	}

	return nil
}
