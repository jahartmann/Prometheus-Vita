package auth

import (
	"context"

	"github.com/google/uuid"
)

// Reader is the read-only surface other domains can depend on. They never
// import the service or repo directly.
type Reader interface {
	GetUser(ctx context.Context, id uuid.UUID) (*User, error)
}
