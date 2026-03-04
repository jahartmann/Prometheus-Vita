package agent

import (
	"context"
	"encoding/json"
	"fmt"

	nodeService "github.com/antigravity/prometheus/internal/service/node"
)

type ListNodesTool struct {
	nodeService *nodeService.Service
}

func NewListNodesTool(nodeSvc *nodeService.Service) *ListNodesTool {
	return &ListNodesTool{nodeService: nodeSvc}
}

func (t *ListNodesTool) Name() string {
	return "list_nodes"
}

func (t *ListNodesTool) Description() string {
	return "Listet alle verwalteten Proxmox-Nodes auf mit Name, Typ, Online-Status"
}

func (t *ListNodesTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {},
		"required": []
	}`)
}

func (t *ListNodesTool) ReadOnly() bool { return true }

func (t *ListNodesTool) Execute(ctx context.Context, _ json.RawMessage) (json.RawMessage, error) {
	nodes, err := t.nodeService.List(ctx)
	if err != nil {
		return json.Marshal(map[string]string{"error": fmt.Sprintf("Fehler beim Abrufen der Nodes: %v", err)})
	}

	type nodeInfo struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Type     string `json:"type"`
		Hostname string `json:"hostname"`
		IsOnline bool   `json:"is_online"`
	}

	result := make([]nodeInfo, 0, len(nodes))
	for _, n := range nodes {
		result = append(result, nodeInfo{
			ID:       n.ID.String(),
			Name:     n.Name,
			Type:     string(n.Type),
			Hostname: n.Hostname,
			IsOnline: n.IsOnline,
		})
	}

	return json.Marshal(result)
}
