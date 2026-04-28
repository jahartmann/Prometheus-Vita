package auth

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const (
	bcryptCost     = 12
	minPasswordLen = 8
)

var ErrPasswordTooShort = errors.New("password must be at least 8 characters")

// HashPassword returns a bcrypt hash of the plaintext password. Returns
// ErrPasswordTooShort for passwords shorter than minPasswordLen.
func HashPassword(plain string) (string, error) {
	if len(plain) < minPasswordLen {
		return "", ErrPasswordTooShort
	}
	h, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(h), nil
}

// VerifyPassword returns nil if plain matches hash, or an error otherwise.
// The error is intentionally opaque so callers do not leak whether the
// failure was a wrong password or an invalid hash.
func VerifyPassword(hash, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
}
