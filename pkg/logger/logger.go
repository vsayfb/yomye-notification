package logger

import (
	"log/slog"
	"os"

	"github.com/vsayfb/gig-platform-notification-lambda/internal/config"
)

func Init(env string) {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	var handler slog.Handler

	switch env {
	case config.EnvironmentProduction:
		handler = slog.NewJSONHandler(os.Stdout, opts)
	default:
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	slog.SetDefault(slog.New(handler))
}
