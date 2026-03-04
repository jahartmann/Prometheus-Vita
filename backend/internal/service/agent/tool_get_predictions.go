package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/antigravity/prometheus/internal/service/prediction"
)

type GetPredictionsTool struct {
	predictionSvc *prediction.Service
}

func NewGetPredictionsTool(predictionSvc *prediction.Service) *GetPredictionsTool {
	return &GetPredictionsTool{predictionSvc: predictionSvc}
}

func (t *GetPredictionsTool) Name() string {
	return "get_predictions"
}

func (t *GetPredictionsTool) Description() string {
	return "Zeigt Predictive-Maintenance-Vorhersagen an (wann Disk/Memory Schwellenwerte erreicht werden)"
}

func (t *GetPredictionsTool) ReadOnly() bool { return true }

func (t *GetPredictionsTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"node_id": {
				"type": "string",
				"description": "Optionale Node-UUID - wenn leer, werden kritische Vorhersagen angezeigt"
			}
		},
		"required": []
	}`)
}

func (t *GetPredictionsTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
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
		preds, err := t.predictionSvc.ListByNode(ctx, nodeID)
		if err != nil {
			return json.Marshal(map[string]string{"error": fmt.Sprintf("Fehler: %v", err)})
		}
		if preds == nil {
			return json.Marshal(map[string]interface{}{"predictions": []interface{}{}, "count": 0})
		}
		return json.Marshal(map[string]interface{}{"predictions": preds, "count": len(preds)})
	}

	preds, err := t.predictionSvc.ListCritical(ctx)
	if err != nil {
		return json.Marshal(map[string]string{"error": fmt.Sprintf("Fehler: %v", err)})
	}
	if preds == nil {
		return json.Marshal(map[string]interface{}{"predictions": []interface{}{}, "count": 0})
	}
	return json.Marshal(map[string]interface{}{"predictions": preds, "count": len(preds)})
}
