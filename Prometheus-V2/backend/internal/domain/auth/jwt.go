package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const minJWTSecretLen = 32

var (
	ErrJWTSecretTooShort = errors.New("jwt secret must be at least 32 bytes")
	ErrInvalidToken      = errors.New("invalid token")
)

// AccessClaims is the payload of a Prometheus V2 access token.
type AccessClaims struct {
	UserID      uuid.UUID `json:"uid"`
	Role        string    `json:"role"`
	Permissions []string  `json:"perms,omitempty"`
	jwt.RegisteredClaims
}

// JWTSigner signs and verifies access tokens with HS256.
type JWTSigner struct {
	secret []byte
	issuer string
}

// NewJWTSigner constructs a signer. The secret must be at least 32 bytes;
// shorter secrets will fail at first sign call to surface misconfiguration
// in main.go before any request is served.
func NewJWTSigner(secret []byte, issuer string) *JWTSigner {
	return &JWTSigner{secret: secret, issuer: issuer}
}

func (s *JWTSigner) SignAccessToken(userID uuid.UUID, role string, perms []string, ttl time.Duration) (string, error) {
	if len(s.secret) < minJWTSecretLen {
		return "", ErrJWTSecretTooShort
	}
	now := time.Now().UTC()
	claims := AccessClaims{
		UserID:      userID,
		Role:        role,
		Permissions: perms,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}
	return signed, nil
}

func (s *JWTSigner) VerifyAccessToken(raw string) (*AccessClaims, error) {
	claims := &AccessClaims{}
	parsed, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	if !parsed.Valid {
		return nil, ErrInvalidToken
	}
	if claims.Issuer != s.issuer {
		return nil, fmt.Errorf("%w: wrong issuer", ErrInvalidToken)
	}
	return claims, nil
}
