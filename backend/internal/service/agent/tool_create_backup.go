package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/antigravity/prometheus/internal/model"
	backupService "github.com/antigravity/prometheus/internal/service/backup"
)

type CreateBackupTool struct {
	backupService *backupService.Service
}

func NewCreateBackupTool(backupSvc *backupService.Service) *CreateBackupTool {
	return &CreateBackupTool{backupService: backupSvc}
}

func (t *CreateBackupTool) Name() string {
	return "create_backup"
}

func (t *CreateBackupTool) Description() string {
	return "Erstellt ein Konfigurations-Backup fuer einen Node"
}

func (t *CreateBackupTool) ReadOnly() bool { return false }

func (t *CreateBackupTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"node_id": {
				"type": "string",
				"description": "Die UUID des Nodes"
			},
			"notes": {
				"type": "string",
				"description": "Optionale Notizen zum Backup"
			}
		},
		"required": ["node_id"]
	}`)
}

func (t *CreateBackupTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var params struct {
		NodeID string `json:"node_id"`
		Notes  string `json:"notes"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("parse arguments: %w", err)
	}

	nodeID, err := uuid.Parse(params.NodeID)
	if err != nil {
		return json.Marshal(map[string]string{"error": "Ungueltige Node-ID"})
	}

	req := model.CreateBackupRequest{
		BackupType: model.BackupTypeManual,
		Notes:      params.Notes,
	}

	backup, err := t.backupService.CreateBackup(ctx, nodeID, req)
	if err != nil {
		return json.Marshal(map[string]string{"error": fmt.Sprintf("Fehler beim Erstellen des Backups: %v", err)})
	}

	return json.Marshal(map[string]interface{}{
		"id":         backup.ID.String(),
		"version":    backup.Version,
		"status":     string(backup.Status),
		"file_count": backup.FileCount,
		"total_size": backup.TotalSize,
	})
}
