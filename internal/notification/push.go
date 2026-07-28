package notification

import "context"

// PushResult describes permanent per-device outcomes that the service must
// reconcile after an FCM send. An unregistered token is handled, not retried.
type PushResult struct {
	UnregisteredTokens []string
}

// PushProvider sends a rendered push notification to a set of device
// tokens. The Firebase adapter is the concrete implementation; the
// service depends only on this interface.
type PushProvider interface {
	Send(ctx context.Context, tokens []string, push *PushNotification) (PushResult, error)
}
