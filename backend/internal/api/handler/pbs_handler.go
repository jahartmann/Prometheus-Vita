package handler

import (
	"errors"
	"fmt"

	apiPkg "github.com/antigravity/prometheus/internal/api"
	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/proxmox"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/crypto"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type PBSHandler struct {
	nodeRepo  repository.NodeRepository
	encryptor *crypto.Encryptor
}

func NewPBSHandler(nodeRepo repository.NodeRepository, encryptor *crypto.Encryptor) *PBSHandler {
	return &PBSHandler{
		nodeRepo:  nodeRepo,
		encryptor: encryptor,
	}
}

func (h *PBSHandler) createPBSClient(ctx echo.Context, nodeID uuid.UUID) (*proxmox.PBSClient, error) {
	node, err := h.nodeRepo.GetByID(ctx.Request().Context(), nodeID)
	if err != nil {
		return nil, err
	}

	if node.Type != model.NodeTypePBS {
		return nil, fmt.Errorf("node is not a PBS server")
	}

	tokenID, err := h.encryptor.Decrypt(node.APITokenID)
	if err != nil {
		return nil, fmt.Errorf("decrypt token id: %w", err)
	}

	tokenSecret, err := h.encryptor.Decrypt(node.APITokenSecret)
	if err != nil {
		return nil, fmt.Errorf("decrypt token secret: %w", err)
	}

	return proxmox.NewPBSClient(node.Hostname, node.Port, tokenID, tokenSecret), nil
}

type PBSDatastoreResponse struct {
	Name         string  `json:"name"`
	Path         string  `json:"path,omitempty"`
	Comment      string  `json:"comment,omitempty"`
	Total        int64   `json:"total"`
	Used         int64   `json:"used"`
	Available    int64   `json:"available"`
	UsagePercent float64 `json:"usage_percent"`
	GCStatus     string  `json:"gc_status,omitempty"`
}

func (h *PBSHandler) GetDatastores(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	pbsClient, err := h.createPBSClient(c, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "node not found")
		}
		return apiPkg.InternalError(c, "failed to connect to PBS server")
	}

	datastores, err := pbsClient.GetDatastores(c.Request().Context())
	if err != nil {
		return apiPkg.InternalError(c, "failed to get datastores")
	}

	statuses, err := pbsClient.GetDatastoreStatus(c.Request().Context())
	if err != nil {
		return apiPkg.InternalError(c, "failed to get datastore status")
	}

	statusMap := make(map[string]proxmox.PBSDatastoreStatus)
	for _, s := range statuses {
		statusMap[s.Store] = s
	}

	responses := make([]PBSDatastoreResponse, 0, len(datastores))
	for _, ds := range datastores {
		resp := PBSDatastoreResponse{
			Name:    ds.Name,
			Path:    ds.Path,
			Comment: ds.Comment,
		}
		if status, ok := statusMap[ds.Name]; ok {
			resp.Total = status.Total
			resp.Used = status.Used
			resp.Available = status.Available
			resp.UsagePercent = status.UsagePercent
			resp.GCStatus = status.GCStatus
		}
		responses = append(responses, resp)
	}

	return apiPkg.Success(c, responses)
}

func (h *PBSHandler) GetBackupJobs(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	pbsClient, err := h.createPBSClient(c, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "node not found")
		}
		return apiPkg.InternalError(c, "failed to connect to PBS server")
	}

	jobs, err := pbsClient.GetBackupJobs(c.Request().Context())
	if err != nil {
		return apiPkg.InternalError(c, "failed to get backup jobs")
	}

	return apiPkg.Success(c, jobs)
}
