package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/labstack/echo/v4"
)

func TestRequirePermissionAllowsRolePermission(t *testing.T) {
	e := echo.New()
	c := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
	c.Set(ContextKeyRole, model.RoleOperator)

	handler := RequirePermission(model.PermissionVMPower)(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})

	if err := handler(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
}

func TestRequirePermissionRejectsMissingRolePermission(t *testing.T) {
	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), rec)
	c.Set(ContextKeyRole, model.RoleViewer)

	handler := RequirePermission(model.PermissionVMPower)(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})

	if err := handler(c); err != nil {
		t.Fatalf("handler returned unexpected echo error: %v", err)
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestRequirePermissionRestrictsAPITokenScopes(t *testing.T) {
	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(httptest.NewRequest(http.MethodPost, "/", nil), rec)
	c.Set(ContextKeyRole, model.RoleAdmin)
	c.Set(ContextKeyAPIPermissions, []model.Permission{model.PermissionNodesRead})

	handler := RequirePermission(model.PermissionSettingsManage)(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})

	if err := handler(c); err != nil {
		t.Fatalf("handler returned unexpected echo error: %v", err)
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestRequirePermissionUsesLoadedRolePermissions(t *testing.T) {
	e := echo.New()
	c := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
	c.Set(ContextKeyRole, model.RoleViewer)
	c.Set(ContextKeyRolePermissions, []model.Permission{model.PermissionVMPower})

	handler := RequirePermission(model.PermissionVMPower)(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})

	if err := handler(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
}

func TestRequirePermissionLoadedRolePermissionsCanRestrictStaticRole(t *testing.T) {
	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(httptest.NewRequest(http.MethodPost, "/", nil), rec)
	c.Set(ContextKeyRole, model.RoleOperator)
	c.Set(ContextKeyRolePermissions, []model.Permission{model.PermissionNodesRead})

	handler := RequirePermission(model.PermissionVMPower)(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})

	if err := handler(c); err != nil {
		t.Fatalf("handler returned unexpected echo error: %v", err)
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}
