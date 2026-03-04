package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SSHKeyRepository interface {
	Create(ctx context.Context, key *model.SSHKey) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.SSHKey, error)
	ListByNode(ctx context.Context, nodeID uuid.UUID) ([]model.SSHKey, error)
	Update(ctx context.Context, key *model.SSHKey) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetExpiringSoon(ctx context.Context, before time.Time) ([]model.SSHKey, error)

	CreateRotationSchedule(ctx context.Context, sched *model.SSHKeyRotationSchedule) error
	GetRotationScheduleByNode(ctx context.Context, nodeID uuid.UUID) (*model.SSHKeyRotationSchedule, error)
	UpdateRotationSchedule(ctx context.Context, sched *model.SSHKeyRotationSchedule) error
	ListDueRotations(ctx context.Context, before time.Time) ([]model.SSHKeyRotationSchedule, error)
}

type pgSSHKeyRepository struct {
	db *pgxpool.Pool
}

func NewSSHKeyRepository(db *pgxpool.Pool) SSHKeyRepository {
	return &pgSSHKeyRepository{db: db}
}

func (r *pgSSHKeyRepository) Create(ctx context.Context, key *model.SSHKey) error {
	key.ID = uuid.New()
	_, err := r.db.Exec(ctx,
		`INSERT INTO ssh_keys (id, node_id, name, key_type, public_key, private_key, fingerprint, is_deployed, deployed_at, expires_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())`,
		key.ID, key.NodeID, key.Name, key.KeyType, key.PublicKey, key.PrivateKey,
		key.Fingerprint, key.IsDeployed, key.DeployedAt, key.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("create ssh key: %w", err)
	}
	return nil
}

func (r *pgSSHKeyRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.SSHKey, error) {
	var k model.SSHKey
	err := r.db.QueryRow(ctx,
		`SELECT id, node_id, name, key_type, public_key, private_key, fingerprint, is_deployed, deployed_at, expires_at, created_at, updated_at
		 FROM ssh_keys WHERE id = $1`, id,
	).Scan(&k.ID, &k.NodeID, &k.Name, &k.KeyType, &k.PublicKey, &k.PrivateKey,
		&k.Fingerprint, &k.IsDeployed, &k.DeployedAt, &k.ExpiresAt, &k.CreatedAt, &k.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get ssh key: %w", err)
	}
	return &k, nil
}

func (r *pgSSHKeyRepository) ListByNode(ctx context.Context, nodeID uuid.UUID) ([]model.SSHKey, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, node_id, name, key_type, public_key, private_key, fingerprint, is_deployed, deployed_at, expires_at, created_at, updated_at
		 FROM ssh_keys WHERE node_id = $1 ORDER BY created_at DESC`, nodeID)
	if err != nil {
		return nil, fmt.Errorf("list ssh keys: %w", err)
	}
	defer rows.Close()

	var keys []model.SSHKey
	for rows.Next() {
		var k model.SSHKey
		if err := rows.Scan(&k.ID, &k.NodeID, &k.Name, &k.KeyType, &k.PublicKey, &k.PrivateKey,
			&k.Fingerprint, &k.IsDeployed, &k.DeployedAt, &k.ExpiresAt, &k.CreatedAt, &k.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan ssh key: %w", err)
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func (r *pgSSHKeyRepository) Update(ctx context.Context, key *model.SSHKey) error {
	_, err := r.db.Exec(ctx,
		`UPDATE ssh_keys SET name=$1, public_key=$2, private_key=$3, fingerprint=$4, is_deployed=$5, deployed_at=$6, expires_at=$7, updated_at=NOW()
		 WHERE id=$8`,
		key.Name, key.PublicKey, key.PrivateKey, key.Fingerprint, key.IsDeployed, key.DeployedAt, key.ExpiresAt, key.ID,
	)
	if err != nil {
		return fmt.Errorf("update ssh key: %w", err)
	}
	return nil
}

func (r *pgSSHKeyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM ssh_keys WHERE id=$1", id)
	if err != nil {
		return fmt.Errorf("delete ssh key: %w", err)
	}
	return nil
}

func (r *pgSSHKeyRepository) GetExpiringSoon(ctx context.Context, before time.Time) ([]model.SSHKey, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, node_id, name, key_type, public_key, private_key, fingerprint, is_deployed, deployed_at, expires_at, created_at, updated_at
		 FROM ssh_keys WHERE expires_at IS NOT NULL AND expires_at < $1 AND is_deployed = true ORDER BY expires_at ASC`, before)
	if err != nil {
		return nil, fmt.Errorf("get expiring ssh keys: %w", err)
	}
	defer rows.Close()

	var keys []model.SSHKey
	for rows.Next() {
		var k model.SSHKey
		if err := rows.Scan(&k.ID, &k.NodeID, &k.Name, &k.KeyType, &k.PublicKey, &k.PrivateKey,
			&k.Fingerprint, &k.IsDeployed, &k.DeployedAt, &k.ExpiresAt, &k.CreatedAt, &k.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan ssh key: %w", err)
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func (r *pgSSHKeyRepository) CreateRotationSchedule(ctx context.Context, sched *model.SSHKeyRotationSchedule) error {
	sched.ID = uuid.New()
	_, err := r.db.Exec(ctx,
		`INSERT INTO ssh_key_rotation_schedules (id, node_id, interval_days, is_active, last_rotated_at, next_rotation_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())`,
		sched.ID, sched.NodeID, sched.IntervalDays, sched.IsActive, sched.LastRotatedAt, sched.NextRotationAt,
	)
	if err != nil {
		return fmt.Errorf("create rotation schedule: %w", err)
	}
	return nil
}

func (r *pgSSHKeyRepository) GetRotationScheduleByNode(ctx context.Context, nodeID uuid.UUID) (*model.SSHKeyRotationSchedule, error) {
	var s model.SSHKeyRotationSchedule
	err := r.db.QueryRow(ctx,
		`SELECT id, node_id, interval_days, is_active, last_rotated_at, next_rotation_at, created_at, updated_at
		 FROM ssh_key_rotation_schedules WHERE node_id = $1`, nodeID,
	).Scan(&s.ID, &s.NodeID, &s.IntervalDays, &s.IsActive, &s.LastRotatedAt, &s.NextRotationAt, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get rotation schedule: %w", err)
	}
	return &s, nil
}

func (r *pgSSHKeyRepository) UpdateRotationSchedule(ctx context.Context, sched *model.SSHKeyRotationSchedule) error {
	_, err := r.db.Exec(ctx,
		`UPDATE ssh_key_rotation_schedules SET interval_days=$1, is_active=$2, last_rotated_at=$3, next_rotation_at=$4, updated_at=NOW()
		 WHERE id=$5`,
		sched.IntervalDays, sched.IsActive, sched.LastRotatedAt, sched.NextRotationAt, sched.ID,
	)
	if err != nil {
		return fmt.Errorf("update rotation schedule: %w", err)
	}
	return nil
}

func (r *pgSSHKeyRepository) ListDueRotations(ctx context.Context, before time.Time) ([]model.SSHKeyRotationSchedule, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, node_id, interval_days, is_active, last_rotated_at, next_rotation_at, created_at, updated_at
		 FROM ssh_key_rotation_schedules WHERE is_active = true AND next_rotation_at IS NOT NULL AND next_rotation_at < $1`, before)
	if err != nil {
		return nil, fmt.Errorf("list due rotations: %w", err)
	}
	defer rows.Close()

	var schedules []model.SSHKeyRotationSchedule
	for rows.Next() {
		var s model.SSHKeyRotationSchedule
		if err := rows.Scan(&s.ID, &s.NodeID, &s.IntervalDays, &s.IsActive, &s.LastRotatedAt, &s.NextRotationAt, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan rotation schedule: %w", err)
		}
		schedules = append(schedules, s)
	}
	return schedules, rows.Err()
}
