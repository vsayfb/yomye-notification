package config

import (
	"context"
	"fmt"
	"os"

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
	env := os.Getenv(AppEnv)

	if env == EnvironmentProduction {
		return loadAWS(ctx)
	}

	return loadEnv()
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
