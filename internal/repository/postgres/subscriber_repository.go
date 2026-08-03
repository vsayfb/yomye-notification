package postgres

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vsayfb/gig-platform-notification-lambda/internal/subscriber"
)

// SubscriberRepository is the PostgreSQL implementation of
// subscriber.Repository.
type SubscriberRepository struct {
	pool *pgxpool.Pool
}

func NewSubscriberRepository(pool *pgxpool.Pool) *SubscriberRepository {
	return &SubscriberRepository{pool: pool}
}

var _ subscriber.Repository = (*SubscriberRepository)(nil)

func (r *SubscriberRepository) ListByCategoryAndLocation(
	ctx context.Context,
	categoryID uuid.UUID,
	countryID *uuid.UUID,
	placeID *uuid.UUID,
) ([]subscriber.Subscriber, error) {
	slog.InfoContext(
		ctx,
		"find subscribers",
		"category_id",
		categoryID,
		"country_id",
		countryID,
		"place_id",
		placeID,
	)

	const query = `
		SELECT
			u.id
		FROM users u
		JOIN user_categories uc
			ON uc.user_id = u.id
		WHERE uc.category_id = $1
		  AND u.country_id IS NOT DISTINCT FROM $2::uuid
		  AND ($3::uuid IS NULL OR u.place_id = $3);
	`

	rows, err := r.pool.Query(ctx, query, categoryID, countryID, placeID)

	if err != nil {
		return nil, fmt.Errorf("subscriber_repository: query: %w", err)
	}

	defer rows.Close()

	subscribers, err := pgx.CollectRows(rows, pgx.RowToStructByName[subscriber.Subscriber])

	if err != nil {
		return nil, fmt.Errorf("subscriber_repository: collect rows: %w", err)
	}

	return subscribers, nil
}
