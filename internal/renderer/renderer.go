package renderer

import (
	"context"
	"encoding/json"

	"github.com/vsayfb/gig-platform-notification-lambda/internal/notification"
)

type Renderer interface {
	Render(ctx context.Context, payload json.RawMessage) (*notification.Notification, *notification.PushNotification, error)
}
