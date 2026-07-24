package notification

import "context"

// PushProvider sends a rendered push notification to a set of device
// tokens. The Firebase adapter is the concrete implementation; the
// service depends only on this interface.
type PushProvider interface {
	Send(ctx context.Context, tokens []string, push *PushNotification) error
}
