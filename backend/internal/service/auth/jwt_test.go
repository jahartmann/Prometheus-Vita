package auth

import (
	"testing"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const testSecret = "test-secret-at-least-32-characters-long!"

func testUser() *model.User {
	return &model.User{ID: uuid.New(), Username: "alice", Role: model.RoleAdmin}
}

func TestJWTRoundTrip(t *testing.T) {
	svc := NewJWTService(testSecret, 15, 24)
	u := testUser()
	pair, err := svc.GenerateTokenPair(u)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	claims, err := svc.ValidateAccessToken(pair.AccessToken)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if claims.UserID != u.ID || claims.Username != "alice" || claims.Role != model.RoleAdmin {
		t.Fatalf("claims mismatch: %+v", claims)
	}
}

func TestJWTRejectsWrongSecret(t *testing.T) {
	a := NewJWTService("secret-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 15, 24)
	b := NewJWTService("secret-bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 15, 24)
	pair, _ := a.GenerateTokenPair(testUser())
	if _, err := b.ValidateAccessToken(pair.AccessToken); err == nil {
		t.Fatalf("token signed with a different secret must be rejected")
	}
}

// The classic JWT bypass: a token with alg=none must be rejected by the HMAC
// signing-method check, not accepted as unsigned.
func TestJWTRejectsNoneAlgorithm(t *testing.T) {
	svc := NewJWTService(testSecret, 15, 24)
	tok := jwt.NewWithClaims(jwt.SigningMethodNone, &Claims{Username: "evil", Role: model.RoleAdmin})
	forged, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("sign none token: %v", err)
	}
	if _, err := svc.ValidateAccessToken(forged); err == nil {
		t.Fatalf("alg=none token must be rejected by the HMAC method check")
	}
}

func TestJWTRejectsExpiredToken(t *testing.T) {
	// Negative access expiry → the generated token is already expired.
	svc := NewJWTService(testSecret, -1, 24)
	pair, _ := svc.GenerateTokenPair(testUser())
	if _, err := svc.ValidateAccessToken(pair.AccessToken); err == nil {
		t.Fatalf("expired token must be rejected")
	}
}

func TestJWTRejectsGarbage(t *testing.T) {
	svc := NewJWTService(testSecret, 15, 24)
	if _, err := svc.ValidateAccessToken("not.a.valid.jwt"); err == nil {
		t.Fatalf("corrupted token string must be rejected")
	}
}
