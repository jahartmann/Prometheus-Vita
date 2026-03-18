package repository

import (
	"context"
	"fmt"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NetworkDeviceRepository interface {
	Upsert(ctx context.Context, device *model.NetworkDevice) error
	ListByNode(ctx context.Context, nodeID uuid.UUID) ([]model.NetworkDevice, error)
	Update(ctx context.Context, id uuid.UUID, req model.UpdateNetworkDeviceRequest) error
}

type pgNetworkDeviceRepository struct {
	db *pgxpool.Pool
}

func NewNetworkDeviceRepository(db *pgxpool.Pool) NetworkDeviceRepository {
	return &pgNetworkDeviceRepository{db: db}
}

func (r *pgNetworkDeviceRepository) Upsert(ctx context.Context, device *model.NetworkDevice) error {
	if device.ID == uuid.Nil {
		device.ID = uuid.New()
	}
	_, err := r.db.Exec(ctx,
		`INSERT INTO network_devices (id, node_id, ip, mac, hostname, first_seen, last_seen, is_known)
		 VALUES ($1, $2, $3, $4, $5, NOW(), NOW(), $6)
		 ON CONFLICT (node_id, ip) DO UPDATE SET last_seen=NOW(), mac=EXCLUDED.mac, hostname=EXCLUDED.hostname`,
		device.ID, device.NodeID, device.IP, device.MAC, device.Hostname, device.IsKnown,
	)
	if err != nil {
		return fmt.Errorf("upsert network device: %w", err)
	}
	return nil
}

func (r *pgNetworkDeviceRepository) ListByNode(ctx context.Context, nodeID uuid.UUID) ([]model.NetworkDevice, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, node_id, ip, mac, hostname, first_seen, last_seen, is_known
		 FROM network_devices WHERE node_id = $1 ORDER BY ip ASC`,
		nodeID)
	if err != nil {
		return nil, fmt.Errorf("list network devices: %w", err)
	}
	defer rows.Close()

	var devices []model.NetworkDevice
	for rows.Next() {
		var d model.NetworkDevice
		if err := rows.Scan(&d.ID, &d.NodeID, &d.IP, &d.MAC, &d.Hostname, &d.FirstSeen, &d.LastSeen, &d.IsKnown); err != nil {
			return nil, fmt.Errorf("scan network device: %w", err)
		}
		devices = append(devices, d)
	}
	return devices, rows.Err()
}

func (r *pgNetworkDeviceRepository) Update(ctx context.Context, id uuid.UUID, req model.UpdateNetworkDeviceRequest) error {
	if req.Hostname != nil {
		_, err := r.db.Exec(ctx, `UPDATE network_devices SET hostname=$2 WHERE id=$1`, id, *req.Hostname)
		if err != nil {
			return fmt.Errorf("update network device hostname: %w", err)
		}
	}
	if req.IsKnown != nil {
		_, err := r.db.Exec(ctx, `UPDATE network_devices SET is_known=$2 WHERE id=$1`, id, *req.IsKnown)
		if err != nil {
			return fmt.Errorf("update network device is_known: %w", err)
		}
	}
	return nil
}
