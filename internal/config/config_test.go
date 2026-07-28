package config

import (
	"context"
	"strings"
	"testing"
)

func TestLoadRejectsUnknownEnvironmentBeforeLoadingLocalConfig(t *testing.T) {
	t.Setenv(AppEnv, "staging")
	t.Setenv(AWSLambdaFunctionName, "")

	_, err := Load(context.Background())
	if err == nil {
		t.Fatal("Load() error = nil, want unsupported environment error")
	}
	if !strings.Contains(err.Error(), "unsupported APP_ENV") {
		t.Fatalf("Load() error = %q, want unsupported APP_ENV error", err)
	}
}

func TestUseAWSConfig(t *testing.T) {
	tests := []struct {
		name       string
		env        string
		lambdaName string
		want       bool
	}{
		{name: "production", env: EnvironmentProduction, want: true},
		{name: "prod alias", env: EnvironmentProd, want: true},
		{name: "Lambda without APP_ENV", lambdaName: "notification-service", want: true},
		{name: "local development", env: EnvironmentDevelopment, want: false},
		{name: "local without environment", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := useAWSConfig(tt.env, tt.lambdaName); got != tt.want {
				t.Errorf("useAWSConfig(%q, %q) = %v, want %v", tt.env, tt.lambdaName, got, tt.want)
			}
		})
	}
}
