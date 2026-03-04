package model

import (
	"time"

	"github.com/google/uuid"
)

type SSHKey struct {
	ID          uuid.UUID  `json:"id"`
	NodeID      uuid.UUID  `json:"node_id"`
	Name        string     `json:"name"`
	KeyType     string     `json:"key_type"`
	PublicKey   string     `json:"public_key"`
	PrivateKey  string     `json:"-"` // encrypted, never exposed
	Fingerprint string     `json:"fingerprint"`
	IsDeployed  bool       `json:"is_deployed"`
	DeployedAt  *time.Time `json:"deployed_at,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type SSHKeyRotationSchedule struct {
	ID             uuid.UUID  `json:"id"`
	NodeID         uuid.UUID  `json:"node_id"`
	IntervalDays   int        `json:"interval_days"`
	IsActive       bool       `json:"is_active"`
	LastRotatedAt  *time.Time `json:"last_rotated_at,omitempty"`
	NextRotationAt *time.Time `json:"next_rotation_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type CreateSSHKeyRequest struct {
	Name      string `json:"name" validate:"required"`
	KeyType   string `json:"key_type,omitempty"` // ed25519 (default), rsa
	ExpiresAt string `json:"expires_at,omitempty"`
	Deploy    bool   `json:"deploy,omitempty"`
}

type CreateRotationScheduleRequest struct {
	IntervalDays int  `json:"interval_days" validate:"required"`
	IsActive     bool `json:"is_active"`
}
