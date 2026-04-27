package handler

import (
	"context"
	"strings"

	apiPkg "github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/service/netscan"
	nodeService "github.com/antigravity/prometheus/internal/service/node"
	sshPkg "github.com/antigravity/prometheus/internal/ssh"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// BandwidthHandler exposes the iperf3-based active bandwidth measurement
// between two managed Proxmox nodes. The actual orchestration lives in
// service/netscan/bandwidth.go — this handler only does request validation,
// node lookup, and adaptation between nodeService.RunSSHCommand and the
// bandwidthRunner interface that the service expects.
type BandwidthHandler struct {
	nodeSvc *nodeService.Service
}

func NewBandwidthHandler(nodeSvc *nodeService.Service) *BandwidthHandler {
	return &BandwidthHandler{nodeSvc: nodeSvc}
}

type bandwidthRequest struct {
	TargetNodeID string `json:"target_node_id"`
	TargetHost   string `json:"target_host,omitempty"`
	DurationSec  int    `json:"duration_sec,omitempty"`
	Port         int    `json:"port,omitempty"`
	Protocol     string `json:"protocol,omitempty"` // tcp|udp
	Reverse      bool   `json:"reverse,omitempty"`
}

// Run handles POST /nodes/:id/bandwidth-test.
func (h *BandwidthHandler) Run(c echo.Context) error {
	srcID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid source node id")
	}
	var req bandwidthRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}
	dstID, err := uuid.Parse(req.TargetNodeID)
	if err != nil {
		return apiPkg.BadRequest(c, "invalid target_node_id")
	}
	if srcID == dstID {
		return apiPkg.BadRequest(c, "Quell- und Ziel-Node müssen unterschiedlich sein")
	}

	ctx := c.Request().Context()
	dstNode, err := h.nodeSvc.GetByID(ctx, dstID)
	if err != nil {
		return apiPkg.NotFound(c, "Ziel-Node nicht gefunden")
	}

	host := strings.TrimSpace(req.TargetHost)
	if host == "" {
		host = dstNode.Hostname
	}
	if host == "" {
		return apiPkg.BadRequest(c, "Ziel-Node hat keinen Hostnamen — bitte target_host angeben")
	}

	src := nodeRunner{svc: h.nodeSvc, id: srcID}
	dst := nodeRunner{svc: h.nodeSvc, id: dstID}

	result, err := netscan.RunBandwidthTest(ctx, src, dst, srcID.String(), dstID.String(), netscan.BandwidthTestOptions{
		DurationSec: req.DurationSec,
		Port:        req.Port,
		Protocol:    req.Protocol,
		Reverse:     req.Reverse,
		TargetHost:  host,
	})
	if err != nil {
		return apiPkg.InternalError(c, err.Error())
	}
	return apiPkg.Success(c, result)
}

// nodeRunner adapts nodeService.RunSSHCommand to the bandwidthRunner
// interface used by the netscan package. Returning the exit code separately
// lets callers distinguish "iperf3 returned warnings" from "ssh broke".
type nodeRunner struct {
	svc *nodeService.Service
	id  uuid.UUID
}

func (r nodeRunner) Run(ctx context.Context, cmd string) (string, int, error) {
	res, err := r.svc.RunSSHCommand(ctx, r.id, cmd)
	if err != nil {
		return "", 0, err
	}
	return commandStdout(res), res.ExitCode, nil
}

func commandStdout(r *sshPkg.CommandResult) string {
	if r == nil {
		return ""
	}
	return r.Stdout
}
