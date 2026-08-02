package firebase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

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

func (p *PushProvider) Send(
	ctx context.Context,
	tokens []string,
	push *notification.PushNotification,
) (notification.PushResult, error) {
	result := notification.PushResult{
		UnregisteredTokens: make([]string, 0),
	}
	var errs []error

	for i, token := range tokens {
		if err := p.client.Send(ctx, token, push.Title, push.Body, push.Data); err != nil {
			httpStatus, status, fcmErrorCode, message := fb.ErrorDetails(err)
			slog.WarnContext(
				ctx,
				"FCM send failed",
				"device_index",
				i,
				"http_status",
				httpStatus,
				"error_status",
				status,
				"fcm_error_code",
				fcmErrorCode,
				"message",
				message,
			)

			if fb.IsUnregistered(err) {
				result.UnregisteredTokens = append(result.UnregisteredTokens, token)
				continue
			}

			// Tokens are credentials and must not be emitted into application
			// logs. The position is enough to correlate a failure within this
			// delivery attempt.
			errs = append(errs, fmt.Errorf("device %d: %w", i, err))
		}
	}

	if len(errs) > 0 {
		return result, fmt.Errorf(
			"firebase: %d/%d sends failed: %w",
			len(errs),
			len(tokens),
			errors.Join(errs...),
		)
	}

	return result, nil
}
