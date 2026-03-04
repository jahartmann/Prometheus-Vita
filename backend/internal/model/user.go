package model

import (
	"time"

	"github.com/google/uuid"
)

type UserRole string

const (
	RoleAdmin    UserRole = "admin"
	RoleOperator UserRole = "operator"
	RoleViewer   UserRole = "viewer"
)

func (r UserRole) IsValid() bool {
	switch r {
	case RoleAdmin, RoleOperator, RoleViewer:
		return true
	}
	return false
}

// Autonomy levels for AI agent tool execution
const (
	AutonomyReadOnly  = 0 // Read-only: no write tools allowed
	AutonomyConfirm   = 1 // Confirm: write tools require approval
	AutonomyFullAuto  = 2 // Full auto: all tools executed immediately
)

type User struct {
	ID            uuid.UUID  `json:"id"`
	Username      string     `json:"username"`
	Email         string     `json:"email"`
	PasswordHash  string     `json:"-"`
	Role          UserRole   `json:"role"`
	IsActive      bool       `json:"is_active"`
	AutonomyLevel int        `json:"autonomy_level"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	LastLogin     *time.Time `json:"last_login,omitempty"`
}

// DTOs

type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type LoginResponse struct {
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	User         UserInfo `json:"user"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type RefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type UserInfo struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
	Role     UserRole  `json:"role"`
	IsActive bool      `json:"is_active"`
}

func (u *User) ToInfo() UserInfo {
	return UserInfo{
		ID:       u.ID,
		Username: u.Username,
		Role:     u.Role,
		IsActive: u.IsActive,
	}
}

// User Management DTOs

type UserResponse struct {
	ID            uuid.UUID  `json:"id"`
	Username      string     `json:"username"`
	Email         string     `json:"email"`
	Role          UserRole   `json:"role"`
	IsActive      bool       `json:"is_active"`
	AutonomyLevel int        `json:"autonomy_level"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	LastLogin     *time.Time `json:"last_login,omitempty"`
}

func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:            u.ID,
		Username:      u.Username,
		Email:         u.Email,
		Role:          u.Role,
		IsActive:      u.IsActive,
		AutonomyLevel: u.AutonomyLevel,
		CreatedAt:     u.CreatedAt,
		UpdatedAt:     u.UpdatedAt,
		LastLogin:     u.LastLogin,
	}
}

type CreateUserRequest struct {
	Username string   `json:"username" validate:"required"`
	Email    string   `json:"email"`
	Password string   `json:"password" validate:"required"`
	Role     UserRole `json:"role" validate:"required"`
}

type UpdateUserRequest struct {
	Username      *string   `json:"username,omitempty"`
	Email         *string   `json:"email,omitempty"`
	Role          *UserRole `json:"role,omitempty"`
	IsActive      *bool     `json:"is_active,omitempty"`
	AutonomyLevel *int      `json:"autonomy_level,omitempty"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password,omitempty"`
	NewPassword     string `json:"new_password" validate:"required"`
}
