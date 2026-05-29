package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// fakeUserRepo is a minimal repository.UserRepository for middleware tests.
// Only GetByID returns meaningful data; the rest satisfy the interface.
type fakeUserRepo struct {
	user *model.User
}

func (f *fakeUserRepo) GetByID(_ context.Context, _ uuid.UUID) (*model.User, error) {
	return f.user, nil
}
func (f *fakeUserRepo) Create(_ context.Context, _ *model.User) error              { return nil }
func (f *fakeUserRepo) GetByUsername(_ context.Context, _ string) (*model.User, error) { return nil, nil }
func (f *fakeUserRepo) List(_ context.Context) ([]model.User, error)              { return nil, nil }
func (f *fakeUserRepo) Update(_ context.Context, _ *model.User) error             { return nil }
func (f *fakeUserRepo) Delete(_ context.Context, _ uuid.UUID) error               { return nil }
func (f *fakeUserRepo) UpdateLastLogin(_ context.Context, _ uuid.UUID) error      { return nil }
func (f *fakeUserRepo) UpdatePassword(_ context.Context, _ uuid.UUID, _ string) error { return nil }
func (f *fakeUserRepo) Count(_ context.Context) (int, error)                      { return 0, nil }
func (f *fakeUserRepo) CountByRole(_ context.Context, _ model.UserRole) (int, error) { return 0, nil }

// run executes the MustChangePassword middleware for the given request against a
// user that must change its password, and reports whether next() was reached.
func run(t *testing.T, method, path string) (reached bool, status int) {
	t.Helper()
	uid := uuid.New()
	// Use the real user id in the password path so the test exercises the
	// actual /api/v1/users/<uuid>/password route shape.
	if path == "__pwpath__" {
		path = "/api/v1/users/" + uid.String() + "/password"
	}
	repo := &fakeUserRepo{user: &model.User{ID: uid, IsActive: true, MustChangePassword: true}}

	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(httptest.NewRequest(method, path, nil), rec)
	c.Set(ContextKeyUserID, uid)

	handler := MustChangePassword(repo)(func(c echo.Context) error {
		reached = true
		return c.NoContent(http.StatusNoContent)
	})
	if err := handler(c); err != nil {
		t.Fatalf("handler returned echo error: %v", err)
	}
	return reached, rec.Code
}

// This is the critical first-run bug: a must-change-password user must be able
// to reach the real password-change route POST /api/v1/users/<uuid>/password.
func TestMustChangePasswordAllowsRealPasswordChangeRoute(t *testing.T) {
	reached, status := run(t, http.MethodPost, "__pwpath__")
	if !reached {
		t.Fatalf("password-change route was blocked for a must-change-password user (status %d); the seeded admin would be permanently locked out", status)
	}
}

// Other protected routes must stay blocked while the flag is set.
func TestMustChangePasswordBlocksOtherRoutes(t *testing.T) {
	reached, status := run(t, http.MethodGet, "/api/v1/nodes")
	if reached {
		t.Fatalf("protected route was allowed for a must-change-password user")
	}
	if status != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", status, http.StatusForbidden)
	}
}

// The forced-change page needs the password policy and the /auth/me refresh.
func TestMustChangePasswordAllowsSupportingReads(t *testing.T) {
	for _, tc := range []struct {
		method, path string
	}{
		{http.MethodGet, "/api/v1/auth/me"},
		{http.MethodGet, "/api/v1/password-policy"},
		{http.MethodPost, "/api/v1/auth/logout"},
		{http.MethodPost, "/api/v1/auth/refresh"},
	} {
		reached, status := run(t, tc.method, tc.path)
		if !reached {
			t.Fatalf("%s %s was blocked for a must-change-password user (status %d)", tc.method, tc.path, status)
		}
	}
}
