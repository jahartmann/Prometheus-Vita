package handler

import (
	"log/slog"
	"sync"

	apiPkg "github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/proxmox"
	"github.com/antigravity/prometheus/internal/repository"
	nodeService "github.com/antigravity/prometheus/internal/service/node"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// ClusterISO represents an ISO/template available across multiple nodes.
type ClusterISO struct {
	Name   string   `json:"name"`
	Volid  string   `json:"volid"`
	Format string   `json:"format"`
	Size   int64    `json:"size"`
	CTime  int64    `json:"ctime"`
	Nodes  []string `json:"nodes"` // node IDs where this ISO exists
}

// SyncCenterHandler handles cluster-wide ISO and tag sync operations.
type SyncCenterHandler struct {
	nodeService *nodeService.Service
	nodeRepo    repository.NodeRepository
	tagRepo     repository.TagRepository
}

func NewSyncCenterHandler(nodeSvc *nodeService.Service, nodeRepo repository.NodeRepository, tagRepo repository.TagRepository) *SyncCenterHandler {
	return &SyncCenterHandler{
		nodeService: nodeSvc,
		nodeRepo:    nodeRepo,
		tagRepo:     tagRepo,
	}
}

// ListClusterISOs returns all ISOs and templates from all online PVE nodes, merged by filename.
func (h *SyncCenterHandler) ListClusterISOs(c echo.Context) error {
	ctx := c.Request().Context()

	nodes, err := h.nodeRepo.List(ctx)
	if err != nil {
		return apiPkg.InternalError(c, "failed to list nodes")
	}

	type nodeResult struct {
		NodeID string
		ISOs   []proxmox.StorageContent
		Err    error
	}

	var wg sync.WaitGroup
	results := make([]nodeResult, 0, len(nodes))
	var mu sync.Mutex

	for i := range nodes {
		n := nodes[i]
		if !n.IsOnline || n.Type != "pve" {
			continue
		}
		wg.Add(1)
		go func(nodeID uuid.UUID, nodeIDStr string) {
			defer wg.Done()

			isos, isoErr := h.nodeService.ListISOs(ctx, nodeID)
			templates, tplErr := h.nodeService.ListTemplates(ctx, nodeID)

			mu.Lock()
			defer mu.Unlock()

			if isoErr != nil && tplErr != nil {
				slog.Warn("failed to list ISOs/templates for node",
					slog.String("node_id", nodeIDStr), slog.Any("error", isoErr))
				results = append(results, nodeResult{NodeID: nodeIDStr, Err: isoErr})
				return
			}

			var combined []proxmox.StorageContent
			if isoErr == nil {
				combined = append(combined, isos...)
			}
			if tplErr == nil {
				combined = append(combined, templates...)
			}
			results = append(results, nodeResult{NodeID: nodeIDStr, ISOs: combined})
		}(n.ID, n.ID.String())
	}

	wg.Wait()

	// Merge by filename
	isoMap := make(map[string]*ClusterISO)
	for _, r := range results {
		if r.Err != nil {
			continue
		}
		for _, iso := range r.ISOs {
			filename := extractFilename(iso.Volid)

			if existing, ok := isoMap[filename]; ok {
				existing.Nodes = append(existing.Nodes, r.NodeID)
				if iso.Size > existing.Size {
					existing.Size = iso.Size
				}
			} else {
				isoMap[filename] = &ClusterISO{
					Name:   filename,
					Volid:  iso.Volid,
					Format: iso.Format,
					Size:   iso.Size,
					CTime:  iso.CTime,
					Nodes:  []string{r.NodeID},
				}
			}
		}
	}

	clusterISOs := make([]ClusterISO, 0, len(isoMap))
	for _, iso := range isoMap {
		clusterISOs = append(clusterISOs, *iso)
	}

	return apiPkg.Success(c, clusterISOs)
}

// SyncAllTags syncs tags from ALL online PVE nodes at once.
func (h *SyncCenterHandler) SyncAllTags(c echo.Context) error {
	ctx := c.Request().Context()

	nodes, err := h.nodeRepo.List(ctx)
	if err != nil {
		return apiPkg.InternalError(c, "failed to list nodes")
	}

	type syncResult struct {
		NodeID   string `json:"node_id"`
		NodeName string `json:"node_name"`
		Imported int    `json:"imported"`
		Error    string `json:"error,omitempty"`
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var results []syncResult
	totalImported := 0

	for i := range nodes {
		n := nodes[i]
		if !n.IsOnline || n.Type != "pve" {
			continue
		}
		wg.Add(1)
		go func(nodeID uuid.UUID, nodeName string) {
			defer wg.Done()

			count, syncErr := h.nodeService.SyncTagsFromProxmox(ctx, nodeID)

			mu.Lock()
			defer mu.Unlock()

			r := syncResult{
				NodeID:   nodeID.String(),
				NodeName: nodeName,
				Imported: count,
			}
			if syncErr != nil {
				r.Error = syncErr.Error()
				slog.Warn("failed to sync tags from node",
					slog.String("node_id", nodeID.String()), slog.Any("error", syncErr))
			} else {
				totalImported += count
			}
			results = append(results, r)
		}(n.ID, n.Name)
	}

	wg.Wait()

	if results == nil {
		results = []syncResult{}
	}

	return apiPkg.Success(c, map[string]interface{}{
		"total_imported": totalImported,
		"results":        results,
	})
}

// extractFilename extracts the filename from a Proxmox volid like "local:iso/debian.iso"
func extractFilename(volid string) string {
	for i := len(volid) - 1; i >= 0; i-- {
		if volid[i] == '/' {
			return volid[i+1:]
		}
	}
	for i := len(volid) - 1; i >= 0; i-- {
		if volid[i] == ':' {
			return volid[i+1:]
		}
	}
	return volid
}
