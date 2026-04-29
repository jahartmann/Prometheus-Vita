package auth_test

import (
	"testing"

	"github.com/antigravity/prometheus-v2/internal/domain/auth"
	"github.com/stretchr/testify/require"
)

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := auth.HashPassword("correct-horse-battery-staple")
	require.NoError(t, err)
	require.NotEmpty(t, hash)
	require.NotEqual(t, "correct-horse-battery-staple", hash)

	require.NoError(t, auth.VerifyPassword(hash, "correct-horse-battery-staple"))
	require.Error(t, auth.VerifyPassword(hash, "wrong-password"))
}

func TestHashPassword_RejectsTooShort(t *testing.T) {
	_, err := auth.HashPassword("short")
	require.Error(t, err)
}
