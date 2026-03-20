package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MigrationLog struct {
	ID          int64     `json:"id"`
	MigrationID uuid.UUID `json:"migration_id"`
	Line        string    `json:"line"`
	Level       string    `json:"level"`
	Phase       string    `json:"phase"`
	CreatedAt   string    `json:"created_at"`
}

type MigrationLogRepository interface {
	Append(ctx context.Context, migrationID uuid.UUID, line, level, phase string) error
	ListByMigration(ctx context.Context, migrationID uuid.UUID) ([]MigrationLog, error)
	DeleteByMigration(ctx context.Context, migrationID uuid.UUID) error
}

type pgMigrationLogRepository struct {
	db *pgxpool.Pool
}

func NewMigrationLogRepository(db *pgxpool.Pool) MigrationLogRepository {
	return &pgMigrationLogRepository{db: db}
}

func (r *pgMigrationLogRepository) Append(ctx context.Context, migrationID uuid.UUID, line, level, phase string) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO migration_logs (migration_id, line, level, phase) VALUES ($1, $2, $3, $4)`,
		migrationID, line, level, phase)
	if err != nil {
		return fmt.Errorf("append migration log: %w", err)
	}
	return nil
}

func (r *pgMigrationLogRepository) ListByMigration(ctx context.Context, migrationID uuid.UUID) ([]MigrationLog, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, migration_id, line, level, phase, created_at FROM migration_logs WHERE migration_id = $1 ORDER BY id ASC`,
		migrationID)
	if err != nil {
		return nil, fmt.Errorf("list migration logs: %w", err)
	}
	defer rows.Close()

	var logs []MigrationLog
	for rows.Next() {
		var l MigrationLog
		if err := rows.Scan(&l.ID, &l.MigrationID, &l.Line, &l.Level, &l.Phase, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan migration log: %w", err)
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

func (r *pgMigrationLogRepository) DeleteByMigration(ctx context.Context, migrationID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM migration_logs WHERE migration_id = $1`, migrationID)
	if err != nil {
		return fmt.Errorf("delete migration logs: %w", err)
	}
	return nil
}
