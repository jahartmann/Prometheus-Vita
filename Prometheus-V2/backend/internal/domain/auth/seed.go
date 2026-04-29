package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/antigravity/prometheus-v2/internal/db/repo"
	"github.com/google/uuid"
)

// SeedBootstrapAdmin creates an admin user if no users exist yet. It is a
// no-op when at least one user already exists. Returns an error if email/
// password are empty AND no user exists; the operator must provide them
// for the first start.
func SeedBootstrapAdmin(ctx context.Context, q Querier, email, password string, logger *slog.Logger) error {
	count, err := q.CountUsers(ctx)
	if err != nil {
		return fmt.Errorf("count users: %w", err)
	}
	if count > 0 {
		return nil
	}
	if email == "" || password == "" {
		return errors.New("no users in database; set PROMETHEUS_BOOTSTRAP_ADMIN_EMAIL and PROMETHEUS_BOOTSTRAP_ADMIN_PASSWORD on first start")
	}
	hash, err := HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash bootstrap password: %w", err)
	}
	id := uuid.New()
	if _, err := q.CreateUser(ctx, repo.CreateUserParams{
		ID:           pgUUID(id),
		Email:        email,
		Name:         "Bootstrap Admin",
		PasswordHash: hash,
		Role:         RoleAdmin,
		Enabled:      true,
	}); err != nil {
		return fmt.Errorf("create bootstrap admin: %w", err)
	}
	logger.Info("bootstrap admin created", slog.String("email", email), slog.String("id", id.String()))
	return nil
}
