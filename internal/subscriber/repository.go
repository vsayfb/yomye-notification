package subscriber

import (
	"context"

	"github.com/google/uuid"
)

// CityID mirrors the users.city_id column type.
type CityID int16

// Subscriber is the minimal shape needed to notify a matched user — just
// enough to build a per-recipient event payload. It intentionally isn't
// the full user/subscriber domain model.
type Subscriber struct {
	ID uuid.UUID `db:"id"`
}

// Repository resolves subscribers live at read time. It must never be
// snapshotted into an event payload: subscriptions and city can change
// between publish and consume, and the caller wants the current state.
type Repository interface {
	ListByCategoryAndCity(ctx context.Context, categoryID uuid.UUID, cityID CityID) ([]Subscriber, error)
}
