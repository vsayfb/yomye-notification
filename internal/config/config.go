package config

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/vsayfb/gig-platform-notification-lambda/pkg/fb"
)

type Config struct {
	APP AppConfig
	DB  DBConfig
	TEL TelemetryConfig
}

type AppConfig struct {
	Env                     string
	FirebaseCredentialsPath string
	FirebaseCredentials     *fb.FirebaseServiceAccount
}

type TelemetryConfig struct {
	OtelCollectorAddr string
}

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

func Load(ctx context.Context) (*Config, error) {
	env := strings.ToLower(strings.TrimSpace(os.Getenv(AppEnv)))

	switch env {
	case "", EnvironmentDevelopment:
		return loadEnv()
	case EnvironmentStaging:
		return loadAWS(ctx)
	case EnvironmentProduction:
		return loadGCP(ctx)
	default:
		return nil, fmt.Errorf(
			"unsupported %s value %q; expected %q, %q, or %q",
			AppEnv,
			env,
			EnvironmentDevelopment,
			EnvironmentStaging,
			EnvironmentProduction,
		)
	}
}

func (c *DBConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host,
		c.Port,
		c.User,
		c.Password,
		c.Name,
		c.SSLMode,
	)
}
