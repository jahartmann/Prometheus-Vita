package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/antigravity/prometheus/internal/service/anomaly"
)

type GetAnomaliesTool struct {
	anomalySvc *anomaly.Service
}

func NewGetAnomaliesTool(anomalySvc *anomaly.Service) *GetAnomaliesTool {
	return &GetAnomaliesTool{anomalySvc: anomalySvc}
}

func (t *GetAnomaliesTool) Name() string {
	return "get_anomalies"
}

func (t *GetAnomaliesTool) Description() string {
	return "Listet erkannte Anomalien in den Metriken auf (ungeklaerte oder pro Node)"
}

func (t *GetAnomaliesTool) ReadOnly() bool { return true }

func (t *GetAnomaliesTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"node_id": {
				"type": "string",
				"description": "Optionale Node-UUID - wenn leer, werden alle ungeloesten Anomalien angezeigt"
			}
		},
		"required": []
	}`)
}

func (t *GetAnomaliesTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var params struct {
		NodeID string `json:"node_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("parse arguments: %w", err)
	}

	if params.NodeID != "" {
		nodeID, err := uuid.Parse(params.NodeID)
		if err != nil {
			return json.Marshal(map[string]string{"error": "Ungueltige Node-ID"})
		}
		records, err := t.anomalySvc.ListByNode(ctx, nodeID)
		if err != nil {
			return json.Marshal(map[string]string{"error": fmt.Sprintf("Fehler: %v", err)})
		}
		if records == nil {
			return json.Marshal(map[string]interface{}{"anomalies": []interface{}{}, "count": 0})
		}
		return json.Marshal(map[string]interface{}{"anomalies": records, "count": len(records)})
	}

	records, err := t.anomalySvc.ListUnresolved(ctx)
	if err != nil {
		return json.Marshal(map[string]string{"error": fmt.Sprintf("Fehler: %v", err)})
	}
	if records == nil {
		return json.Marshal(map[string]interface{}{"anomalies": []interface{}{}, "count": 0})
	}
	return json.Marshal(map[string]interface{}{"anomalies": records, "count": len(records)})
}
