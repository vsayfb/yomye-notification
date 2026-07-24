package firebase

import (
	"context"
	"errors"
	"fmt"

	"github.com/vsayfb/gig-platform-notification-lambda/internal/notification"
	"github.com/vsayfb/gig-platform-notification-lambda/pkg/fb"
)

// PushProvider is the Firebase implementation of notification.PushProvider.
// It translates the internal PushNotification abstraction into FCM sends;
// it is not itself an FCM model.
type PushProvider struct {
	client *fb.FCMClient
}

func NewPushProvider(client *fb.FCMClient) *PushProvider {
	return &PushProvider{client: client}
}

var _ notification.PushProvider = (*PushProvider)(nil)

func (p *PushProvider) Send(ctx context.Context, tokens []string, push *notification.PushNotification) error {
	var errs []error

	for _, token := range tokens {
		if err := p.client.Send(ctx, token, push.Title, push.Body, push.Data); err != nil {
			errs = append(errs, fmt.Errorf("token %s: %w", token, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("firebase: %d/%d sends failed: %w", len(errs), len(tokens), errors.Join(errs...))
	}

	return nil
}
