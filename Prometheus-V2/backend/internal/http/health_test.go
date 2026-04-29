package http_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	httpserver "github.com/antigravity/prometheus-v2/internal/http"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestHealthz_Returns200(t *testing.T) {
	e := echo.New()
	httpserver.RegisterHealth(e, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.JSONEq(t, `{"status":"ok"}`, rec.Body.String())
}

func TestReadyz_ReturnsServiceUnavailableWhenDBNil(t *testing.T) {
	e := echo.New()
	httpserver.RegisterHealth(e, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

func TestSystemHealth_ReturnsExpectedShape(t *testing.T) {
	deps := httpserver.Deps{Logger: slog.Default(), DB: nil, Redis: nil}
	server := httpserver.NewServer(deps)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/health", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"status":"ok"`)
	require.Contains(t, rec.Body.String(), `"version":"0.1.0"`)
}
