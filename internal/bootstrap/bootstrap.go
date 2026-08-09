package bootstrap

import (
	"context"
	"fmt"

	"github.com/vsayfb/gig-platform-notification-lambda/internal/config"
	"github.com/vsayfb/gig-platform-notification-lambda/internal/handler"
	"github.com/vsayfb/gig-platform-notification-lambda/internal/notification"
	"github.com/vsayfb/gig-platform-notification-lambda/internal/renderer"
	"github.com/vsayfb/gig-platform-notification-lambda/internal/repository/firebase"
	"github.com/vsayfb/gig-platform-notification-lambda/internal/repository/postgres"
	"github.com/vsayfb/gig-platform-notification-lambda/internal/service"

	"github.com/vsayfb/gig-platform-notification-lambda/pkg/database"
	"github.com/vsayfb/gig-platform-notification-lambda/pkg/fb"
	"github.com/vsayfb/gig-platform-notification-lambda/pkg/logger"
)

func NewHandler() (*handler.Handler, error) {
	ctx := context.Background()

	cfg, err := config.Load(ctx)

	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	logger.Init(cfg.APP.Env)

	pool, err := database.NewPool(ctx, cfg.DB.DSN())

	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}

	creds, err := cfg.APP.GetFireBaseCredentials()

	if err != nil {
		return nil, fmt.Errorf("load firebase credentials: %w", err)
	}

	fcmClient, err := fb.NewClient(ctx, creds)

	if err != nil {
		return nil, fmt.Errorf("create firebase client: %w", err)
	}

	notifRepo := postgres.NewNotificationRepository(pool)
	tokenRepo := postgres.NewFCMTokenRepository(pool)
	subscriberRepo := postgres.NewSubscriberRepository(pool)
	eventDeduplicator := postgres.NewEventDeduplicator(pool)
	pushProvider := firebase.NewPushProvider(fcmClient)

	dispatcher := service.NewDispatcher()
	dispatcher.Register(notification.EventNewMessage, renderer.NewNewMessageRenderer())
	dispatcher.Register(notification.EventListingApproved, renderer.NewListingApprovedRenderer())

	dispatcher.Register(notification.EventGigCategoryMatched, renderer.NewGigCategoryMatchedRenderer())
	dispatcher.RegisterFanOut(notification.EventGigCategoryMatched, service.NewGigCategoryMatchedResolver(subscriberRepo))

	svc := service.New(dispatcher, notifRepo, tokenRepo, pushProvider, eventDeduplicator)

	return handler.New(svc), nil
}
