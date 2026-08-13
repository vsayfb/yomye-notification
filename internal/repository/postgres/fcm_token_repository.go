package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vsayfb/gig-platform-notification-lambda/internal/fcm"
)

// FCMTokenRepository is the PostgreSQL implementation of fcm.TokenRepository.
//
// Core owns registration, token transfer, platform metadata, freshness, and
// stale-token pruning. Lambda only reads a user's tokens and removes tokens
// Firebase has structurally classified as UNREGISTERED.
type FCMTokenRepository struct {
	pool *pgxpool.Pool
}

func NewFCMTokenRepository(pool *pgxpool.Pool) *FCMTokenRepository {
	return &FCMTokenRepository{pool: pool}
}

var _ fcm.TokenRepository = (*FCMTokenRepository)(nil)

func (r *FCMTokenRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT token FROM fcm_tokens WHERE user_id = $1`, userID)

	if err != nil {
		return nil, fmt.Errorf("fcm_token_repository: query: %w", err)
	}

	defer rows.Close()

	var tokens []string

	for rows.Next() {
		var token string

		if err := rows.Scan(&token); err != nil {
			return nil, fmt.Errorf("fcm_token_repository: scan: %w", err)
		}

		tokens = append(tokens, token)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("fcm_token_repository: rows: %w", err)
	}

	return tokens, nil
}

func (r *FCMTokenRepository) DeleteByUserID(
	ctx context.Context,
	userID uuid.UUID,
	tokens []string,
) error {
	if len(tokens) == 0 {
		return nil
	}

	const query = `DELETE FROM fcm_tokens WHERE user_id = $1 AND token = ANY($2)`

	if _, err := r.pool.Exec(ctx, query, userID, tokens); err != nil {
		return fmt.Errorf("fcm_token_repository: delete invalid tokens: %w", err)
	}

	return nil
}
