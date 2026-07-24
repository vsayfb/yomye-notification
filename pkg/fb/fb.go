package fb

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

type FCMClient struct {
	msgClient *messaging.Client
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

func NewClient(ctx context.Context, creds *FirebaseServiceAccount) (*FCMClient, error) {
	jsonBytes, err := json.Marshal(creds)

	if err != nil {
		return nil, err
	}

	app, err := firebase.NewApp(ctx, nil, option.WithAuthCredentialsJSON(option.AuthorizedUser, jsonBytes))

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
		return fmt.Errorf("fcm: send failed: %w", err)
	}

	return nil
}
