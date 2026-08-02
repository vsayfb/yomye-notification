package fb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

type FCMClient struct {
	msgClient *messaging.Client
}

// SendError preserves the structured classification made against the original
// Firebase SDK error before any contextual wrapping is added.
type SendError struct {
	HTTPStatus   int
	Status       string
	FCMErrorCode string
	Message      string
	Unregistered bool
	cause        error
}

func (e *SendError) Error() string {
	return e.Message
}

func (e *SendError) Unwrap() error {
	return e.cause
}

type FirebaseServiceAccount struct {
	Type                    string `json:"type"`
	ProjectID               string `json:"project_id"`
	PrivateKeyID            string `json:"private_key_id"`
	PrivateKey              string `json:"private_key"`
	ClientEmail             string `json:"client_email"`
	ClientID                string `json:"client_id"`
	AuthURI                 string `json:"auth_uri"`
	TokenURI                string `json:"token_uri"`
	AuthProviderX509CertURL string `json:"auth_provider_x509_cert_url"`
	ClientX509CertURL       string `json:"client_x509_cert_url"`
	UniverseDomain          string `json:"universe_domain"`
}

func (c *FirebaseServiceAccount) Validate() error {
	if c == nil {
		return fmt.Errorf("credentials are nil")
	}
	if c.Type != string(option.ServiceAccount) {
		return fmt.Errorf("credential type is %q, want %q", c.Type, option.ServiceAccount)
	}
	if c.ProjectID == "" {
		return fmt.Errorf("project_id is required")
	}
	if c.PrivateKey == "" {
		return fmt.Errorf("private_key is required")
	}
	if c.ClientEmail == "" {
		return fmt.Errorf("client_email is required")
	}
	return nil
}

func NewClient(ctx context.Context, creds *FirebaseServiceAccount) (*FCMClient, error) {
	if err := creds.Validate(); err != nil {
		return nil, fmt.Errorf("fcm: invalid service account credentials: %w", err)
	}

	jsonBytes, err := json.Marshal(creds)

	if err != nil {
		return nil, err
	}

	app, err := firebase.NewApp(ctx, nil, option.WithAuthCredentialsJSON(option.ServiceAccount, jsonBytes))

	if err != nil {
		return nil, fmt.Errorf("fcm: failed to init messaging client: %w", err)
	}

	msgClient, err := app.Messaging(ctx)

	if err != nil {
		return nil, fmt.Errorf("fcm: failed to init messaging client: %w", err)
	}

	slog.Info("FCM client initialized successfully")

	return &FCMClient{msgClient: msgClient}, nil
}

func (c *FCMClient) Send(ctx context.Context, token, title, body string, data map[string]string) error {
	if c.msgClient == nil {
		return fmt.Errorf("fcm: messaging client not initialized")
	}

	msg := &messaging.Message{
		Token: token,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: data,
	}

	_, err := c.msgClient.Send(ctx, msg)

	if err != nil {
		details := firebaseErrorDetails(err)
		return &SendError{
			HTTPStatus:   details.HTTPStatus,
			Status:       details.Status,
			FCMErrorCode: details.FCMErrorCode,
			Message:      err.Error(),
			// This is intentionally the only permanent-token predicate.
			// It checks the original, unwrapped SDK error and therefore
			// requires the structured FCM error code UNREGISTERED.
			Unregistered: messaging.IsUnregistered(err),
			cause:        err,
		}
	}

	return nil
}

// IsUnregistered reports whether Firebase has permanently invalidated the
// target registration token. Callers should remove such tokens rather than
// retrying them.
func IsUnregistered(err error) bool {
	var sendErr *SendError
	if errors.As(err, &sendErr) {
		return sendErr.Unregistered
	}

	return errorTreeContains(err, messaging.IsUnregistered)
}

// ErrorDetails returns safe diagnostics for an FCM send failure. It contains
// no registration token.
func ErrorDetails(err error) (httpStatus int, status, fcmErrorCode, message string) {
	var sendErr *SendError
	if errors.As(err, &sendErr) {
		return sendErr.HTTPStatus, sendErr.Status, sendErr.FCMErrorCode, sendErr.Message
	}
	return 0, "", "", err.Error()
}

type fcmErrorDetails struct {
	HTTPStatus   int
	Status       string
	FCMErrorCode string
}

// firebaseErrorDetails reads the exported diagnostic fields on the Admin
// SDK's internal FirebaseError. The SDK does not expose a public accessor for
// these fields, even though they contain the HTTP response and structured FCM
// error code used by messaging.IsUnregistered.
func firebaseErrorDetails(err error) fcmErrorDetails {
	value := reflect.ValueOf(err)
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return fcmErrorDetails{}
	}

	value = value.Elem()
	if value.Kind() != reflect.Struct || value.Type().Name() != "FirebaseError" {
		return fcmErrorDetails{}
	}

	var details fcmErrorDetails

	if field := value.FieldByName("ErrorCode"); field.IsValid() && field.Kind() == reflect.String {
		details.Status = field.String()
	}

	if field := value.FieldByName("Response"); field.IsValid() && field.CanInterface() && !field.IsNil() {
		if response, ok := field.Interface().(*http.Response); ok && response != nil {
			details.HTTPStatus = response.StatusCode
		}
	}

	if field := value.FieldByName("Ext"); field.IsValid() && field.Kind() == reflect.Map {
		key := reflect.ValueOf("messagingErrorCode")
		code := field.MapIndex(key)
		if code.IsValid() && code.Kind() == reflect.Interface && !code.IsNil() {
			code = code.Elem()
		}
		if code.IsValid() && code.Kind() == reflect.String {
			details.FCMErrorCode = code.String()
		}
	}

	return details
}

// errorTreeContains walks ordinary wrapped errors as well as errors.Join
// trees. Firebase's classifier uses a concrete type assertion rather than
// errors.As, so it cannot recognize its own error after callers add context
// with fmt.Errorf("%w").
func errorTreeContains(err error, match func(error) bool) bool {
	if err == nil {
		return false
	}
	if match(err) {
		return true
	}

	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		for _, child := range joined.Unwrap() {
			if errorTreeContains(child, match) {
				return true
			}
		}
		return false
	}

	return errorTreeContains(errors.Unwrap(err), match)
}
