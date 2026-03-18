package repository

import (
	"context"
	"fmt"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NetworkPortRepository interface {
	Upsert(ctx context.Context, port *model.NetworkPort) error
	ListByDevice(ctx context.Context, deviceID uuid.UUID) ([]model.NetworkPort, error)
}

type pgNetworkPortRepository struct {
	db *pgxpool.Pool
}

func NewNetworkPortRepository(db *pgxpool.Pool) NetworkPortRepository {
	return &pgNetworkPortRepository{db: db}
}

func (r *pgNetworkPortRepository) Upsert(ctx context.Context, port *model.NetworkPort) error {
	if port.ID == uuid.Nil {
		port.ID = uuid.New()
	}
	_, err := r.db.Exec(ctx,
		`INSERT INTO network_ports (id, device_id, port, protocol, state, service_name, service_version, last_seen)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		 ON CONFLICT (device_id, port, protocol) DO UPDATE SET state=EXCLUDED.state, service_name=EXCLUDED.service_name, service_version=EXCLUDED.service_version, last_seen=NOW()`,
		port.ID, port.DeviceID, port.Port, port.Protocol, port.State, port.ServiceName, port.ServiceVersion,
	)
	if err != nil {
		return fmt.Errorf("upsert network port: %w", err)
	}
	return nil
}

func (r *pgNetworkPortRepository) ListByDevice(ctx context.Context, deviceID uuid.UUID) ([]model.NetworkPort, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, device_id, port, protocol, state, service_name, service_version, last_seen
		 FROM network_ports WHERE device_id = $1 ORDER BY port ASC`,
		deviceID)
	if err != nil {
		return nil, fmt.Errorf("list network ports: %w", err)
	}
	defer rows.Close()

	var ports []model.NetworkPort
	for rows.Next() {
		var p model.NetworkPort
		if err := rows.Scan(&p.ID, &p.DeviceID, &p.Port, &p.Protocol, &p.State, &p.ServiceName, &p.ServiceVersion, &p.LastSeen); err != nil {
			return nil, fmt.Errorf("scan network port: %w", err)
		}
		ports = append(ports, p)
	}
	return ports, rows.Err()
}
