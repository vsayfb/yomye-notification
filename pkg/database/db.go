package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// pingTimeout bounds the initial connectivity check done at cold start.
// Without this, a bad DSN or unreachable DB would hang on
// context.Background() (no deadline) instead of failing fast with a clear
// error during init.
const pingTimeout = 5 * time.Second

func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)

	if err != nil {
		return nil, fmt.Errorf("database: failed to create pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("database: ping failed: %w", err)
	}

	slog.Info("db connected")

	return pool, nil
}
