package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// testJWTSecret is 32 bytes — meets minJWTSecretLen — and is reused across
// tests so a generated token verifies with the same signer.
var testJWTSecret = []byte("0123456789abcdef0123456789abcdef")

// TestRequireAuth_RejectsMissingBearer verifies that requests without an
// Authorization header are rejected with 401 and the wrapped handler is not
// invoked.
func TestRequireAuth_RejectsMissingBearer(t *testing.T) {
	signer := NewJWTSigner(testJWTSecret, "test")
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	called := false
	mw := RequireAuth(signer)
	err := mw(func(c echo.Context) error {
		called = true
		return c.NoContent(http.StatusOK)
	})(c)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	httpErr, ok := err.(*echo.HTTPError)
	if !ok {
		t.Fatalf("expected *echo.HTTPError, got %T", err)
	}
	if httpErr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", httpErr.Code)
	}
	if called {
		t.Fatal("inner handler must not be called when auth fails")
	}
}

// TestRequireAuth_PassesValidToken verifies that a valid bearer token reaches
// the wrapped handler and the claims are stored in the context where
// ClaimsFromContext can retrieve them.
func TestRequireAuth_PassesValidToken(t *testing.T) {
	signer := NewJWTSigner(testJWTSecret, "test")
	uid := uuid.New()
	tok, err := signer.SignAccessToken(uid, "admin", []string{"system.read"}, time.Minute)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	called := false
	mw := RequireAuth(signer)
	err = mw(func(c echo.Context) error {
		called = true
		claims, ok := ClaimsFromContext(c)
		if !ok {
			t.Fatal("ClaimsFromContext returned (nil, false) inside protected handler")
		}
		if claims.UserID != uid {
			t.Fatalf("user id mismatch: got %s want %s", claims.UserID, uid)
		}
		return c.NoContent(http.StatusOK)
	})(c)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("inner handler must be called when auth succeeds")
	}
}
