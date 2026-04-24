package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/antigravity/prometheus/internal/model"
	backupService "github.com/antigravity/prometheus/internal/service/backup"
)

type RestoreConfigTool struct {
	restoreService *backupService.RestoreService
}

func NewRestoreConfigTool(restoreSvc *backupService.RestoreService) *RestoreConfigTool {
	return &RestoreConfigTool{restoreService: restoreSvc}
}

func (t *RestoreConfigTool) Name() string {
	return "restore_config"
}

func (t *RestoreConfigTool) Description() string {
	return "Stellt Konfigurationsdateien aus einem Backup wieder her (mit Vorschau im Dry-Run-Modus)"
}

func (t *RestoreConfigTool) ReadOnly() bool { return false }

func (t *RestoreConfigTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"backup_id": {
				"type": "string",
				"description": "Die UUID des Backups"
			},
			"file_paths": {
				"type": "array",
				"items": {"type": "string"},
				"description": "Liste der wiederherzustellenden Dateipfade (z.B. ['/etc/network/interfaces'])"
			},
			"dry_run": {
				"type": "boolean",
				"description": "Wenn true, wird nur eine Vorschau angezeigt ohne tatsaechliche Wiederherstellung. Standard: true"
			}
		},
		"required": ["backup_id", "file_paths"]
	}`)
}

func (t *RestoreConfigTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var params struct {
		BackupID  string   `json:"backup_id"`
		FilePaths []string `json:"file_paths"`
		DryRun    *bool    `json:"dry_run"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("parse arguments: %w", err)
	}

	backupID, err := uuid.Parse(params.BackupID)
	if err != nil {
		return json.Marshal(map[string]string{"error": "Ungueltige Backup-ID"})
	}

	dryRun := true
	if params.DryRun != nil {
		dryRun = *params.DryRun
	}
	if dryRun {
		return t.Preview(ctx, args)
	}

	req := model.RestoreRequest{
		FilePaths: params.FilePaths,
		DryRun:    dryRun,
	}

	preview, err := t.restoreService.RestoreFiles(ctx, backupID, req)
	if err != nil {
		return json.Marshal(map[string]string{"error": fmt.Sprintf("Fehler bei der Wiederherstellung: %v", err)})
	}

	result := map[string]interface{}{
		"dry_run": dryRun,
		"files":   preview.Files,
	}
	if dryRun {
		result["message"] = "Vorschau der Wiederherstellung. Setze dry_run=false um die Dateien tatsaechlich wiederherzustellen."
	} else {
		result["message"] = fmt.Sprintf("%d Datei(en) erfolgreich wiederhergestellt.", len(preview.Files))
	}

	return json.Marshal(result)
}

func (t *RestoreConfigTool) Preview(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var params struct {
		BackupID  string   `json:"backup_id"`
		FilePaths []string `json:"file_paths"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("parse arguments: %w", err)
	}
	backupID, err := uuid.Parse(params.BackupID)
	if err != nil {
		return json.Marshal(map[string]string{"error": "Ungueltige Backup-ID"})
	}
	req := model.RestoreRequest{
		FilePaths: params.FilePaths,
		DryRun:    true,
	}
	preview, err := t.restoreService.RestoreFiles(ctx, backupID, req)
	if err != nil {
		return json.Marshal(map[string]string{"error": fmt.Sprintf("Fehler bei der Restore-Vorschau: %v", err)})
	}
	return json.Marshal(map[string]interface{}{
		"dry_run": true,
		"action":  "restore_config",
		"files":   preview.Files,
		"message": "Restore-Vorschau erzeugt. Fuer echte Wiederherstellung dry_run=false setzen und Approval freigeben.",
	})
}
