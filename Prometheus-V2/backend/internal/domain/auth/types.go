package auth

import (
	"time"

	"github.com/google/uuid"
)

// User is the domain representation of an authenticated principal.
type User struct {
	ID           uuid.UUID
	Email        string
	Name         string
	Role         string
	Enabled      bool
	Version      int32
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Session represents a long-lived refresh-token-bound session.
type Session struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	UserAgent  string
	IPAddress  string
	ExpiresAt  time.Time
	RevokedAt  *time.Time
	LastSeenAt time.Time
	CreatedAt  time.Time
}

// AuthTokens is the bundle returned to a client after login or refresh.
type AuthTokens struct {
	AccessToken    string
	RefreshToken   string // raw, set in HttpOnly cookie by handler
	AccessExpiry   time.Time
	RefreshExpiry  time.Time
	User           User
}
