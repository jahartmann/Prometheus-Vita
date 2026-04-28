package http_test

import (
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
