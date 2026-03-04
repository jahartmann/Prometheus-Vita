package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/antigravity/prometheus/internal/service/auth"
)

func seedAdmin(ctx context.Context, authService *auth.Service) {
	username := os.Getenv("ADMIN_USERNAME")
	if username == "" {
		username = "admin"
	}

	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		password = "changeme"
	}

	if err := authService.SeedAdmin(ctx, username, password); err != nil {
		slog.Error("failed to seed admin user", slog.Any("error", err))
	}
}
