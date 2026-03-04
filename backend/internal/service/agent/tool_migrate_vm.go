package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/antigravity/prometheus/internal/model"
	migrationService "github.com/antigravity/prometheus/internal/service/migration"
)

type MigrateVMTool struct {
	migrationService *migrationService.Service
}

func NewMigrateVMTool(migrationSvc *migrationService.Service) *MigrateVMTool {
	return &MigrateVMTool{migrationService: migrationSvc}
}

func (t *MigrateVMTool) Name() string {
	return "migrate_vm"
}

func (t *MigrateVMTool) Description() string {
	return "Migriert eine VM von einem Proxmox-Node zu einem anderen via vzdump-Backup und Restore"
}

func (t *MigrateVMTool) ReadOnly() bool { return false }

func (t *MigrateVMTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"source_node_id": {
				"type": "string",
				"description": "Die UUID des Quell-Nodes"
			},
			"target_node_id": {
				"type": "string",
				"description": "Die UUID des Ziel-Nodes"
			},
			"vmid": {
				"type": "integer",
				"description": "Die VM-ID (z.B. 101)"
			},
			"target_storage": {
				"type": "string",
				"description": "Der Ziel-Storage auf dem Ziel-Node (z.B. local-zfs)"
			},
			"mode": {
				"type": "string",
				"enum": ["stop", "snapshot", "suspend"],
				"description": "Migrations-Modus: stop (VM herunterfahren), snapshot (live, keine Downtime), suspend (kurze Pause). Standard: snapshot"
			}
		},
		"required": ["source_node_id", "target_node_id", "vmid", "target_storage"]
	}`)
}

func (t *MigrateVMTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var params struct {
		SourceNodeID  string `json:"source_node_id"`
		TargetNodeID  string `json:"target_node_id"`
		VMID          int    `json:"vmid"`
		TargetStorage string `json:"target_storage"`
		Mode          string `json:"mode"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("parse arguments: %w", err)
	}

	sourceID, err := uuid.Parse(params.SourceNodeID)
	if err != nil {
		return json.Marshal(map[string]string{"error": "Ungueltige Source-Node-ID"})
	}
	targetID, err := uuid.Parse(params.TargetNodeID)
	if err != nil {
		return json.Marshal(map[string]string{"error": "Ungueltige Target-Node-ID"})
	}

	mode := model.MigrationModeSnapshot
	if params.Mode != "" {
		mode = model.MigrationMode(params.Mode)
		if !mode.IsValid() {
			return json.Marshal(map[string]string{"error": fmt.Sprintf("Ungueltiger Modus: %s", params.Mode)})
		}
	}

	req := model.StartMigrationRequest{
		SourceNodeID:  sourceID,
		TargetNodeID:  targetID,
		VMID:          params.VMID,
		TargetStorage: params.TargetStorage,
		Mode:          mode,
		CleanupSource: true,
		CleanupTarget: true,
	}

	migration, err := t.migrationService.StartMigration(ctx, req, nil)
	if err != nil {
		return json.Marshal(map[string]string{"error": fmt.Sprintf("Fehler beim Starten der Migration: %v", err)})
	}

	return json.Marshal(map[string]interface{}{
		"migration_id": migration.ID.String(),
		"status":       string(migration.Status),
		"vmid":         migration.VMID,
		"vm_name":      migration.VMName,
		"message":      fmt.Sprintf("Migration von VM %d (%s) wurde gestartet. Status kann ueber die Migrations-Uebersicht verfolgt werden.", migration.VMID, migration.VMName),
	})
}
