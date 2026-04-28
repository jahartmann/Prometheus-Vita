package config_test

import (
	"testing"

	"github.com/antigravity/prometheus-v2/internal/config"
	"github.com/stretchr/testify/require"
)

func TestLoad_DefaultsWhenEnvIsEmpty(t *testing.T) {
	t.Setenv("PROMETHEUS_HTTP_ADDR", "")
	t.Setenv("PROMETHEUS_LOG_LEVEL", "")

	cfg, err := config.Load()
	require.NoError(t, err)
	require.Equal(t, ":8180", cfg.HTTPAddr)
	require.Equal(t, "info", cfg.LogLevel)
}

func TestLoad_OverridesFromEnv(t *testing.T) {
	t.Setenv("PROMETHEUS_HTTP_ADDR", ":9999")
	t.Setenv("PROMETHEUS_LOG_LEVEL", "debug")
	t.Setenv("PROMETHEUS_DATABASE_URL", "postgres://x:y@z/db")

	cfg, err := config.Load()
	require.NoError(t, err)
	require.Equal(t, ":9999", cfg.HTTPAddr)
	require.Equal(t, "debug", cfg.LogLevel)
	require.Equal(t, "postgres://x:y@z/db", cfg.DatabaseURL)
}

func TestLoad_AuthDefaults(t *testing.T) {
	t.Setenv("PROMETHEUS_JWT_SECRET", "")
	t.Setenv("PROMETHEUS_JWT_ISSUER", "")

	cfg, err := config.Load()
	require.NoError(t, err)
	require.Empty(t, cfg.JWTSecret)
	require.Equal(t, "prometheus-v2", cfg.JWTIssuer)
	require.False(t, cfg.CookieSecure)
}

func TestLoad_AuthOverrides(t *testing.T) {
	t.Setenv("PROMETHEUS_JWT_SECRET", "abc-secret")
	t.Setenv("PROMETHEUS_COOKIE_SECURE", "true")
	t.Setenv("PROMETHEUS_BOOTSTRAP_ADMIN_EMAIL", "admin@example.com")
	t.Setenv("PROMETHEUS_BOOTSTRAP_ADMIN_PASSWORD", "init-password")

	cfg, err := config.Load()
	require.NoError(t, err)
	require.Equal(t, "abc-secret", cfg.JWTSecret)
	require.True(t, cfg.CookieSecure)
	require.Equal(t, "admin@example.com", cfg.BootstrapAdminEmail)
	require.Equal(t, "init-password", cfg.BootstrapAdminPassword)
}
