package auth_test

import (
	"testing"
	"time"

	"github.com/antigravity/prometheus-v2/internal/domain/auth"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestSignAndVerifyAccessToken(t *testing.T) {
	signer := auth.NewJWTSigner([]byte("test-secret-please-be-32-bytes-or-more!"), "prometheus-v2")
	userID := uuid.New()

	tok, err := signer.SignAccessToken(userID, "admin", []string{"host:read", "vm:read"}, 15*time.Minute)
	require.NoError(t, err)
	require.NotEmpty(t, tok)

	claims, err := signer.VerifyAccessToken(tok)
	require.NoError(t, err)
	require.Equal(t, userID, claims.UserID)
	require.Equal(t, "admin", claims.Role)
	require.ElementsMatch(t, []string{"host:read", "vm:read"}, claims.Permissions)
}

func TestVerifyAccessToken_RejectsExpired(t *testing.T) {
	signer := auth.NewJWTSigner([]byte("test-secret-please-be-32-bytes-or-more!"), "prometheus-v2")
	tok, err := signer.SignAccessToken(uuid.New(), "viewer", nil, -1*time.Minute)
	require.NoError(t, err)

	_, err = signer.VerifyAccessToken(tok)
	require.Error(t, err)
}

func TestVerifyAccessToken_RejectsWrongSecret(t *testing.T) {
	signer := auth.NewJWTSigner([]byte("test-secret-please-be-32-bytes-or-more!"), "prometheus-v2")
	tok, err := signer.SignAccessToken(uuid.New(), "admin", nil, 5*time.Minute)
	require.NoError(t, err)

	other := auth.NewJWTSigner([]byte("a-different-secret-also-32-or-more-yes!"), "prometheus-v2")
	_, err = other.VerifyAccessToken(tok)
	require.Error(t, err)
}

// TestVerifyAccessToken_RejectsNonHS256Algorithm guards against algorithm
// confusion: a token signed with HS512 (or any non-HS256 algorithm) using
// the same secret must be rejected, because the verifier pins to HS256.
func TestVerifyAccessToken_RejectsNonHS256Algorithm(t *testing.T) {
	secret := []byte("test-secret-please-be-32-bytes-or-more!")
	signer := auth.NewJWTSigner(secret, "prometheus-v2")

	// Manually sign a token with HS512 using the same secret.
	now := time.Now().UTC()
	claims := jwt.RegisteredClaims{
		Issuer:    "prometheus-v2",
		Subject:   uuid.New().String(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(5 * time.Minute)),
	}
	hs512 := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	raw, err := hs512.SignedString(secret)
	require.NoError(t, err)

	_, err = signer.VerifyAccessToken(raw)
	require.Error(t, err)
}
