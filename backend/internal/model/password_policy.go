package model

import (
	"time"

	"github.com/google/uuid"
)

type PasswordPolicy struct {
	ID               uuid.UUID  `json:"id"`
	MinLength        int        `json:"min_length"`
	RequireUppercase bool       `json:"require_uppercase"`
	RequireLowercase bool       `json:"require_lowercase"`
	RequireDigit     bool       `json:"require_digit"`
	RequireSpecial   bool       `json:"require_special"`
	MaxLength        int        `json:"max_length"`
	DisallowUsername bool       `json:"disallow_username"`
	UpdatedAt        time.Time  `json:"updated_at"`
	UpdatedBy        *uuid.UUID `json:"updated_by,omitempty"`
}

type UpdatePasswordPolicyRequest struct {
	MinLength        *int  `json:"min_length,omitempty"`
	RequireUppercase *bool `json:"require_uppercase,omitempty"`
	RequireLowercase *bool `json:"require_lowercase,omitempty"`
	RequireDigit     *bool `json:"require_digit,omitempty"`
	RequireSpecial   *bool `json:"require_special,omitempty"`
	MaxLength        *int  `json:"max_length,omitempty"`
	DisallowUsername *bool `json:"disallow_username,omitempty"`
}
