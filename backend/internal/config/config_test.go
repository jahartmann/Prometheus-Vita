package config

import "testing"

func TestLoadSupportsDocumentedEnvAliases(t *testing.T) {
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
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
