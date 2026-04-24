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
	ID                 uuid.UUID  `json:"id"`
	Username           string     `json:"username"`
	Email              string     `json:"email"`
	PasswordHash       string     `json:"-"`
	Role               UserRole   `json:"role"`
	IsActive           bool       `json:"is_active"`
	AutonomyLevel      int        `json:"autonomy_level"`
	MustChangePassword bool       `json:"must_change_password"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	LastLogin          *time.Time `json:"last_login,omitempty"`
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
	ID                 uuid.UUID `json:"id"`
	Username           string    `json:"username"`
	Role               UserRole  `json:"role"`
	IsActive           bool      `json:"is_active"`
	MustChangePassword bool      `json:"must_change_password"`
}

func (u *User) ToInfo() UserInfo {
	return UserInfo{
		ID:                 u.ID,
		Username:           u.Username,
		Role:               u.Role,
		IsActive:           u.IsActive,
		MustChangePassword: u.MustChangePassword,
	}
}

// User Management DTOs

type UserResponse struct {
	ID                 uuid.UUID  `json:"id"`
	Username           string     `json:"username"`
	Email              string     `json:"email"`
	Role               UserRole   `json:"role"`
	IsActive           bool       `json:"is_active"`
	AutonomyLevel      int        `json:"autonomy_level"`
	MustChangePassword bool       `json:"must_change_password"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	LastLogin          *time.Time `json:"last_login,omitempty"`
}

func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:                 u.ID,
		Username:           u.Username,
		Email:              u.Email,
		Role:               u.Role,
		IsActive:           u.IsActive,
		AutonomyLevel:      u.AutonomyLevel,
		MustChangePassword: u.MustChangePassword,
		CreatedAt:          u.CreatedAt,
		UpdatedAt:          u.UpdatedAt,
		LastLogin:          u.LastLogin,
	}
}

type UserInvitation struct {
	ID          uuid.UUID  `json:"id"`
	Username    string     `json:"username"`
	Email       string     `json:"email"`
	Role        UserRole   `json:"role"`
	TokenHash   string     `json:"-"`
	TokenPrefix string     `json:"token_prefix"`
	ExpiresAt   time.Time  `json:"expires_at"`
	AcceptedAt  *time.Time `json:"accepted_at,omitempty"`
	CreatedBy   uuid.UUID  `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
}

type UserInvitationResponse struct {
	ID          uuid.UUID  `json:"id"`
	Username    string     `json:"username"`
	Email       string     `json:"email"`
	Role        UserRole   `json:"role"`
	TokenPrefix string     `json:"token_prefix"`
	ExpiresAt   time.Time  `json:"expires_at"`
	AcceptedAt  *time.Time `json:"accepted_at,omitempty"`
	CreatedBy   uuid.UUID  `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
}

func (i *UserInvitation) ToResponse() UserInvitationResponse {
	return UserInvitationResponse{
		ID:          i.ID,
		Username:    i.Username,
		Email:       i.Email,
		Role:        i.Role,
		TokenPrefix: i.TokenPrefix,
		ExpiresAt:   i.ExpiresAt,
		AcceptedAt:  i.AcceptedAt,
		CreatedBy:   i.CreatedBy,
		CreatedAt:   i.CreatedAt,
	}
}

type CreateUserInvitationRequest struct {
	Username       string   `json:"username" validate:"required"`
	Email          string   `json:"email"`
	Role           UserRole `json:"role" validate:"required"`
	ExpiresInHours int      `json:"expires_in_hours,omitempty"`
}

type CreateUserInvitationResponse struct {
	Invitation UserInvitationResponse `json:"invitation"`
	Token      string                 `json:"token"`
}

type AcceptUserInvitationRequest struct {
	Token    string `json:"token" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type UserSession struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	Revoked   bool      `json:"revoked"`
	IsActive  bool      `json:"is_active"`
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
