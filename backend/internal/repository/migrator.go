package repository

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Migrator struct {
	db  *pgxpool.Pool
	dir string
}

func NewMigrator(db *pgxpool.Pool, migrationsDir string) *Migrator {
	return &Migrator{db: db, dir: migrationsDir}
}

func (m *Migrator) Run(ctx context.Context) error {
	// Acquire a Postgres advisory lock so that when multiple replicas of the
	// server boot at the same time, only one of them actually runs the
	// migrations. The others block here and then observe an up-to-date schema.
	// The arbitrary 64-bit key below is just a project-specific constant.
	const advisoryLockKey int64 = 0x70726f6d65746575 // "prometeu" — project-specific 8-byte constant
	if _, err := m.db.Exec(ctx, "SELECT pg_advisory_lock($1)", advisoryLockKey); err != nil {
		return fmt.Errorf("acquire advisory lock: %w", err)
	}
	defer func() {
		if _, err := m.db.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", advisoryLockKey); err != nil {
			slog.Warn("failed to release migration advisory lock", slog.Any("error", err))
		}
	}()

	if err := m.ensureMigrationsTable(ctx); err != nil {
		return fmt.Errorf("ensure migrations table: %w", err)
	}

	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("get applied migrations: %w", err)
	}

	files, err := m.getMigrationFiles()
	if err != nil {
		return fmt.Errorf("get migration files: %w", err)
	}

	for _, f := range files {
		name := filepath.Base(f)
		if applied[name] {
			continue
		}

		slog.Info("applying migration", slog.String("file", name))

		content, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		tx, err := m.db.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin tx for %s: %w", name, err)
		}

		if _, err := tx.Exec(ctx, string(content)); err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				slog.Warn("migration rollback failed", slog.String("file", name), slog.Any("error", rbErr))
			}
			return fmt.Errorf("execute migration %s: %w", name, err)
		}

		if _, err := tx.Exec(ctx,
			"INSERT INTO schema_migrations (filename) VALUES ($1)", name); err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				slog.Warn("migration rollback failed", slog.String("file", name), slog.Any("error", rbErr))
			}
			return fmt.Errorf("record migration %s: %w", name, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit migration %s: %w", name, err)
		}

		slog.Info("migration applied", slog.String("file", name))
	}

	return nil
}

func (m *Migrator) ensureMigrationsTable(ctx context.Context) error {
	_, err := m.db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id SERIAL PRIMARY KEY,
			filename TEXT NOT NULL UNIQUE,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	return err
}

func (m *Migrator) getAppliedMigrations(ctx context.Context) (map[string]bool, error) {
	rows, err := m.db.Query(ctx, "SELECT filename FROM schema_migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		applied[name] = true
	}
	return applied, rows.Err()
}

func (m *Migrator) getMigrationFiles() ([]string, error) {
	entries, err := os.ReadDir(m.dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, filepath.Join(m.dir, e.Name()))
		}
	}

	sort.Strings(files)
	return files, nil
}
