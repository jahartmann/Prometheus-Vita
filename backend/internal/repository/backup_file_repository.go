package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BackupFileRepository interface {
	CreateBatch(ctx context.Context, files []model.BackupFile) error
	GetByBackupID(ctx context.Context, backupID uuid.UUID) ([]model.BackupFile, error)
	GetSingleFile(ctx context.Context, backupID uuid.UUID, filePath string) (*model.BackupFile, error)
	DeleteByBackupID(ctx context.Context, backupID uuid.UUID) error
}

type pgBackupFileRepository struct {
	db *pgxpool.Pool
}

func NewBackupFileRepository(db *pgxpool.Pool) BackupFileRepository {
	return &pgBackupFileRepository{db: db}
}

func (r *pgBackupFileRepository) CreateBatch(ctx context.Context, files []model.BackupFile) error {
	batch := &pgx.Batch{}
	for i := range files {
		files[i].ID = uuid.New()
		batch.Queue(
			`INSERT INTO backup_files (id, backup_id, file_path, file_hash, file_size,
			        file_permissions, file_owner, content, diff_from_previous, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())`,
			files[i].ID, files[i].BackupID, files[i].FilePath, files[i].FileHash,
			files[i].FileSize, files[i].FilePermissions, files[i].FileOwner,
			files[i].Content, files[i].DiffFromPrevious,
		)
	}

	br := r.db.SendBatch(ctx, batch)
	defer br.Close()

	for range files {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("create backup file batch: %w", err)
		}
	}
	return nil
}

func (r *pgBackupFileRepository) GetByBackupID(ctx context.Context, backupID uuid.UUID) ([]model.BackupFile, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, backup_id, file_path, file_hash, file_size,
		        file_permissions, file_owner, diff_from_previous, created_at
		 FROM backup_files WHERE backup_id = $1 ORDER BY file_path`, backupID)
	if err != nil {
		return nil, fmt.Errorf("list backup files: %w", err)
	}
	defer rows.Close()

	var files []model.BackupFile
	for rows.Next() {
		var f model.BackupFile
		if err := rows.Scan(&f.ID, &f.BackupID, &f.FilePath, &f.FileHash,
			&f.FileSize, &f.FilePermissions, &f.FileOwner,
			&f.DiffFromPrevious, &f.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan backup file: %w", err)
		}
		files = append(files, f)
	}
	return files, rows.Err()
}

func (r *pgBackupFileRepository) GetSingleFile(ctx context.Context, backupID uuid.UUID, filePath string) (*model.BackupFile, error) {
	var f model.BackupFile
	err := r.db.QueryRow(ctx,
		`SELECT id, backup_id, file_path, file_hash, file_size,
		        file_permissions, file_owner, content, diff_from_previous, created_at
		 FROM backup_files WHERE backup_id = $1 AND file_path = $2`, backupID, filePath,
	).Scan(&f.ID, &f.BackupID, &f.FilePath, &f.FileHash,
		&f.FileSize, &f.FilePermissions, &f.FileOwner,
		&f.Content, &f.DiffFromPrevious, &f.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get single backup file: %w", err)
	}
	return &f, nil
}

func (r *pgBackupFileRepository) DeleteByBackupID(ctx context.Context, backupID uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM backup_files WHERE backup_id=$1", backupID)
	if err != nil {
		return fmt.Errorf("delete backup files: %w", err)
	}
	return nil
}
