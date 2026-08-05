package config

import (
	"context"
	"strings"
	"testing"

	"github.com/vsayfb/gig-platform-notification-lambda/pkg/fb"
)

func TestLoadRejectsUnknownEnvironment(t *testing.T) {
	t.Setenv(AppEnv, "stage")

	_, err := Load(context.Background())
	if err == nil || !strings.Contains(err.Error(), "unsupported APP_ENV") {
		t.Fatalf("Load() error = %v, want unsupported APP_ENV error", err)
	}
}

func TestProductionFailsClearlyUntilGCPConfigExists(t *testing.T) {
	t.Setenv(AppEnv, EnvironmentProduction)

	config, err := Load(context.Background())
	if config != nil {
		t.Fatalf("Load() config = %#v, want nil", config)
	}
	if err == nil || !strings.Contains(err.Error(), "GCP production configuration is not implemented") {
		t.Fatalf("Load() error = %v, want explicit GCP implementation error", err)
	}
}

func TestFirebaseCredentialsUseInMemoryValueInStaging(t *testing.T) {
	credentials := &fb.FirebaseServiceAccount{
		Type:        "service_account",
		ProjectID:   "test-project",
		PrivateKey:  "private-key",
		ClientEmail: "firebase@example.iam.gserviceaccount.com",
	}
	app := AppConfig{
		Env:                 EnvironmentStaging,
		FirebaseCredentials: credentials,
	}

	got, err := app.GetFireBaseCredentials()
	if err != nil {
		t.Fatalf("GetFireBaseCredentials() error = %v", err)
	}
	if got != credentials {
		t.Fatalf("GetFireBaseCredentials() = %p, want %p", got, credentials)
	}
}

func TestFirebaseCredentialsRequireConfiguredSource(t *testing.T) {
	app := AppConfig{Env: EnvironmentStaging}

	_, err := app.GetFireBaseCredentials()
	if err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("GetFireBaseCredentials() error = %v, want not configured error", err)
	}
}
