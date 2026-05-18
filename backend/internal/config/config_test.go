package config

import "testing"

func TestLoadSupportsDocumentedEnvAliases(t *testing.T) {
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	// validate() rejects the "changeme_*" placeholders; supply safe values
	// so the test exercises the env-alias logic instead of failing on the
	// placeholder guards.
	t.Setenv("POSTGRES_PASSWORD", "test-pg-pw")
	t.Setenv("REDIS_PASSWORD", "test-redis-pw")
	t.Setenv("JWT_ACCESS_EXPIRY_MINUTES", "")
	t.Setenv("JWT_ACCESS_TOKEN_EXPIRY", "42")
	t.Setenv("JWT_REFRESH_EXPIRY_HOURS", "")
	t.Setenv("JWT_REFRESH_TOKEN_EXPIRY", "240")
	t.Setenv("CORS_ALLOW_ORIGINS", "")
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://example.local")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.JWT.AccessTokenExpiry != 42 {
		t.Fatalf("AccessTokenExpiry = %d, want 42", cfg.JWT.AccessTokenExpiry)
	}
	if cfg.JWT.RefreshTokenExpiry != 240 {
		t.Fatalf("RefreshTokenExpiry = %d, want 240", cfg.JWT.RefreshTokenExpiry)
	}
	if got, want := len(cfg.CORS.AllowOrigins), 2; got != want {
		t.Fatalf("CORS.AllowOrigins length = %d, want %d", got, want)
	}
	if cfg.CORS.AllowOrigins[0] != "http://localhost:3000" || cfg.CORS.AllowOrigins[1] != "http://example.local" {
		t.Fatalf("CORS.AllowOrigins = %#v", cfg.CORS.AllowOrigins)
	}
}

func TestLoadRejectsBadEncryptionKey(t *testing.T) {
	cases := map[string]string{
		"empty":     "",
		"too short": "deadbeef",
		"too long":  "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef00",
		"non-hex":   "ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ",
	}
	for name, key := range cases {
		t.Run(name, func(t *testing.T) {
			t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")
			t.Setenv("ENCRYPTION_KEY", key)
			t.Setenv("POSTGRES_PASSWORD", "test-pg-pw")
			t.Setenv("REDIS_PASSWORD", "test-redis-pw")
			if _, err := Load(); err == nil {
				t.Fatalf("expected validation error for %s ENCRYPTION_KEY, got nil", name)
			}
		})
	}
}

func TestLoadRejectsPlaceholderCredentials(t *testing.T) {
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	t.Setenv("POSTGRES_PASSWORD", "changeme_db_password")
	t.Setenv("REDIS_PASSWORD", "test-redis-pw")
	if _, err := Load(); err == nil {
		t.Fatalf("expected validation error for placeholder POSTGRES_PASSWORD")
	}

	t.Setenv("POSTGRES_PASSWORD", "test-pg-pw")
	t.Setenv("REDIS_PASSWORD", "changeme_redis_password")
	if _, err := Load(); err == nil {
		t.Fatalf("expected validation error for placeholder REDIS_PASSWORD")
	}
}

func TestLogConfigSlogLevel(t *testing.T) {
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	t.Setenv("POSTGRES_PASSWORD", "test-pg-pw")
	t.Setenv("REDIS_PASSWORD", "test-redis-pw")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("LOG_FORMAT", "text")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("Log.Level = %q, want debug", cfg.Log.Level)
	}
	if cfg.Log.Format != "text" {
		t.Errorf("Log.Format = %q, want text", cfg.Log.Format)
	}
}
