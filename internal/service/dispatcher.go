package service

import (
	"fmt"

	"github.com/vsayfb/gig-platform-notification-lambda/internal/notification"
	"github.com/vsayfb/gig-platform-notification-lambda/internal/renderer"
)

// Dispatcher owns one renderer registration per notification event type,
// plus an optional fan-out resolver registration. Adding a plain
// per-recipient notification type requires only an event DTO, a renderer,
// and a Register call here. Adding one that needs recipients resolved at
// consume time (rather than named in the event itself) additionally needs
// a Resolver and a RegisterFanOut call — nothing else in the codebase
// changes, and Renderer's one-in-one-out shape never does either.
type Dispatcher struct {
	renderers map[notification.NotificationEventType]renderer.Renderer
	fanOuts   map[notification.NotificationEventType]Resolver
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		renderers: make(map[notification.NotificationEventType]renderer.Renderer),
		fanOuts:   make(map[notification.NotificationEventType]Resolver),
	}
}

func (d *Dispatcher) Register(eventType notification.NotificationEventType, r renderer.Renderer) {
	d.renderers[eventType] = r
}

func (d *Dispatcher) Renderer(eventType notification.NotificationEventType) (renderer.Renderer, error) {
	r, ok := d.renderers[eventType]

	if !ok {
		return nil, fmt.Errorf("dispatcher: no renderer registered for event type %q", eventType)
	}

	return r, nil
}

func (d *Dispatcher) RegisterFanOut(eventType notification.NotificationEventType, r Resolver) {
	d.fanOuts[eventType] = r
}

// FanOut returns the resolver registered for eventType, if any. Absence is
// not an error: it just means the event's payload already names its own
// recipient and should be rendered as-is.
func (d *Dispatcher) FanOut(eventType notification.NotificationEventType) (Resolver, bool) {
	r, ok := d.fanOuts[eventType]
	return r, ok
}
