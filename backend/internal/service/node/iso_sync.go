package node

import (
	"context"
	"fmt"

	"github.com/antigravity/prometheus/internal/proxmox"
	"github.com/google/uuid"
)

// ListISOs lists all ISO images on a node's local storage.
func (s *Service) ListISOs(ctx context.Context, nodeID uuid.UUID) ([]proxmox.StorageContent, error) {
	node, err := s.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	client, err := s.clientFactory.CreateClient(node)
	if err != nil {
		return nil, fmt.Errorf("create proxmox client: %w", err)
	}

	nodes, err := client.GetNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("get nodes: %w", err)
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("no nodes found")
	}

	return client.GetStorageContent(ctx, nodes[0], "local", "iso")
}

// ListTemplates lists all container templates on a node's local storage.
func (s *Service) ListTemplates(ctx context.Context, nodeID uuid.UUID) ([]proxmox.StorageContent, error) {
	node, err := s.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	client, err := s.clientFactory.CreateClient(node)
	if err != nil {
		return nil, fmt.Errorf("create proxmox client: %w", err)
	}

	nodes, err := client.GetNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("get nodes: %w", err)
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("no nodes found")
	}

	return client.GetStorageContent(ctx, nodes[0], "local", "vztmpl")
}

// SyncContentRequest holds the parameters for syncing content between nodes.
type SyncContentRequest struct {
	SourceNodeID  string `json:"source_node_id"`
	Volid         string `json:"volid"`
	TargetStorage string `json:"target_storage"`
}

// SyncContent downloads content from a source node and uploads it to the target node.
func (s *Service) SyncContent(ctx context.Context, targetNodeID uuid.UUID, req SyncContentRequest) (string, error) {
	sourceID, err := uuid.Parse(req.SourceNodeID)
	if err != nil {
		return "", fmt.Errorf("invalid source node id: %w", err)
	}

	sourceNode, err := s.nodeRepo.GetByID(ctx, sourceID)
	if err != nil {
		return "", fmt.Errorf("get source node: %w", err)
	}

	targetNode, err := s.nodeRepo.GetByID(ctx, targetNodeID)
	if err != nil {
		return "", fmt.Errorf("get target node: %w", err)
	}

	// Build download URL from source node
	// Proxmox storage content can be downloaded via the API
	sourceTokenID, err := s.encryptor.Decrypt(sourceNode.APITokenID)
	if err != nil {
		return "", fmt.Errorf("decrypt source token id: %w", err)
	}

	sourceTokenSecret, err := s.encryptor.Decrypt(sourceNode.APITokenSecret)
	if err != nil {
		return "", fmt.Errorf("decrypt source token secret: %w", err)
	}

	// Get source node's PVE node name
	sourceClient := s.clientFactory.CreateClientFromCredentials(sourceNode.Hostname, sourceNode.Port, sourceTokenID, sourceTokenSecret)
	sourceNodes, err := sourceClient.GetNodes(ctx)
	if err != nil {
		return "", fmt.Errorf("get source nodes: %w", err)
	}
	if len(sourceNodes) == 0 {
		return "", fmt.Errorf("no source nodes found")
	}

	// Build the download URL for the volume
	downloadURL := fmt.Sprintf("https://%s:%d/api2/json/nodes/%s/storage/local/content/%s",
		sourceNode.Hostname, sourceNode.Port, sourceNodes[0], req.Volid)

	// Extract filename from volid (e.g., "local:iso/debian.iso" -> "debian.iso")
	filename := req.Volid
	for i := len(req.Volid) - 1; i >= 0; i-- {
		if req.Volid[i] == '/' {
			filename = req.Volid[i+1:]
			break
		}
	}

	targetClient, err := s.clientFactory.CreateClient(targetNode)
	if err != nil {
		return "", fmt.Errorf("create target proxmox client: %w", err)
	}

	targetNodes, err := targetClient.GetNodes(ctx)
	if err != nil {
		return "", fmt.Errorf("get target nodes: %w", err)
	}
	if len(targetNodes) == 0 {
		return "", fmt.Errorf("no target nodes found")
	}

	targetStorage := req.TargetStorage
	if targetStorage == "" {
		targetStorage = "local"
	}

	return targetClient.DownloadURL(ctx, targetNodes[0], targetStorage, filename, downloadURL)
}
