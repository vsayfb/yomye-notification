package fcm

import (
	"context"

	"github.com/google/uuid"
)

type TokenRepository interface {
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]string, error)
}
