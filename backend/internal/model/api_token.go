package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type APIToken struct {
	ID          uuid.UUID       `json:"id"`
	UserID      uuid.UUID       `json:"user_id"`
	Name        string          `json:"name"`
	TokenHash   string          `json:"-"`
	TokenPrefix string          `json:"token_prefix"`
	Permissions json.RawMessage `json:"permissions"`
	IsActive    bool            `json:"is_active"`
	LastUsedAt  *time.Time      `json:"last_used_at,omitempty"`
	ExpiresAt   *time.Time      `json:"expires_at,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type CreateAPITokenRequest struct {
	Name        string   `json:"name" validate:"required"`
	Permissions []string `json:"permissions,omitempty"`
	ExpiresAt   string   `json:"expires_at,omitempty"`
}

type CreateAPITokenResponse struct {
	Token   string    `json:"token"` // only returned once at creation
	TokenID uuid.UUID `json:"token_id"`
	Name    string    `json:"name"`
	Prefix  string    `json:"prefix"`
}
