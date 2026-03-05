package node

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/proxmox"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/crypto"
	"github.com/antigravity/prometheus/internal/ssh"
	"github.com/google/uuid"
	gossh "golang.org/x/crypto/ssh"
)

// ErrNodeUnreachable indicates that the Proxmox node could not be reached.
var ErrNodeUnreachable = errors.New("node unreachable")

type NetworkInterfaceWithAlias struct {
	proxmox.NetworkInterface
	DisplayName string `json:"display_name,omitempty"`
	Description string `json:"description,omitempty"`
	Color       string `json:"color,omitempty"`
}

type Service struct {
	nodeRepo      repository.NodeRepository
	aliasRepo     repository.NetworkAliasRepository
	tagRepo       repository.TagRepository
	encryptor     *crypto.Encryptor
	clientFactory proxmox.ClientFactory
	sshPool       *ssh.Pool
}

func NewService(
	nodeRepo repository.NodeRepository,
	encryptor *crypto.Encryptor,
	clientFactory proxmox.ClientFactory,
	aliasRepo repository.NetworkAliasRepository,
	tagRepo repository.TagRepository,
	sshPool *ssh.Pool,
) *Service {
	return &Service{
		nodeRepo:      nodeRepo,
		aliasRepo:     aliasRepo,
		tagRepo:       tagRepo,
		encryptor:     encryptor,
		clientFactory: clientFactory,
		sshPool:       sshPool,
	}
}

func (s *Service) Create(ctx context.Context, req model.CreateNodeRequest) (*model.Node, error) {
	if !req.Type.IsValid() {
		return nil, fmt.Errorf("invalid node type: %s", req.Type)
	}

	port := req.Port
	if port == 0 {
		port = 8006
	}

	sshPort := req.SSHPort
	if sshPort == 0 {
		sshPort = 22
	}

	sshUser := req.SSHUser
	if sshUser == "" {
		sshUser = "root"
	}

	encTokenID, err := s.encryptor.Encrypt(req.APITokenID)
	if err != nil {
		return nil, fmt.Errorf("encrypt token id: %w", err)
	}

	encTokenSecret, err := s.encryptor.Encrypt(req.APITokenSecret)
	if err != nil {
		return nil, fmt.Errorf("encrypt token secret: %w", err)
	}

	var encSSHPrivateKey string
	if req.SSHPrivateKey != "" {
		encSSHPrivateKey, err = s.encryptor.Encrypt(req.SSHPrivateKey)
		if err != nil {
			return nil, fmt.Errorf("encrypt ssh private key: %w", err)
		}
	}

	metadata := req.Metadata
	if metadata == nil {
		metadata = json.RawMessage("{}")
	}

	node := &model.Node{
		Name:           req.Name,
		Type:           req.Type,
		Hostname:       req.Hostname,
		Port:           port,
		APITokenID:     encTokenID,
		APITokenSecret: encTokenSecret,
		SSHPort:        sshPort,
		SSHUser:        sshUser,
		SSHPrivateKey:  encSSHPrivateKey,
		IsOnline:       false,
		Metadata:       metadata,
	}

	if err := s.nodeRepo.Create(ctx, node); err != nil {
		return nil, fmt.Errorf("create node: %w", err)
	}

	slog.Info("node created", slog.String("name", node.Name), slog.String("id", node.ID.String()))

	// Immediate status check so new nodes don't start as offline
	s.checkNodeOnline(ctx, node)

	return node, nil
}

func (s *Service) Onboard(ctx context.Context, req model.OnboardNodeRequest) (*model.Node, error) {
	if !req.Type.IsValid() {
		return nil, fmt.Errorf("invalid node type: %s", req.Type)
	}

	// Defaults
	port := req.Port
	if port == 0 {
		port = 8006
	}
	username := req.Username
	if username == "" {
		username = "root@pam"
	}
	sshPort := req.SSHPort
	if sshPort == 0 {
		sshPort = 22
	}

	// 1. Ticket-Auth
	ticket, csrf, err := proxmox.GetTicket(ctx, req.Hostname, port, username, req.Password)
	if err != nil {
		return nil, fmt.Errorf("proxmox authentication failed: %w", err)
	}

	// 2. Create API token (unique per node to avoid conflicts in clusters)
	sanitizedHost := strings.ReplaceAll(strings.ReplaceAll(req.Hostname, ".", "-"), ":", "-")
	tokenName := fmt.Sprintf("prometheus-vita-%s", sanitizedHost)
	tokenID, tokenSecret, err := proxmox.CreateAPITokenWithTicket(ctx, req.Hostname, port, username, ticket, csrf, tokenName)
	if err != nil {
		return nil, fmt.Errorf("create API token failed: %w", err)
	}

	// 3. Generate SSH keypair (Ed25519)
	pubKey, privKey, err := generateEd25519KeyPair()
	if err != nil {
		return nil, fmt.Errorf("generate ssh key pair: %w", err)
	}

	// 4. SSH with password -> deploy public key
	sshClient, err := ssh.NewClient(ssh.SSHConfig{
		Host:     req.Hostname,
		Port:     sshPort,
		User:     "root",
		Password: req.Password,
	})
	if err != nil {
		return nil, fmt.Errorf("ssh connect with password: %w", err)
	}
	defer sshClient.Close()

	deployCmd := fmt.Sprintf(
		`mkdir -p ~/.ssh && chmod 700 ~/.ssh && echo %q >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys`,
		strings.TrimSpace(pubKey),
	)
	result, err := sshClient.RunCommand(ctx, deployCmd)
	if err != nil {
		return nil, fmt.Errorf("deploy ssh key: %w", err)
	}
	if result.ExitCode != 0 {
		return nil, fmt.Errorf("deploy ssh key failed: %s", result.Stderr)
	}

	// 5. Get PVE node name via SSH hostname command
	hostnameResult, err := sshClient.RunCommand(ctx, "hostname")
	pveNodeName := ""
	if err == nil && hostnameResult.ExitCode == 0 {
		pveNodeName = strings.TrimSpace(hostnameResult.Stdout)
	}

	// Build metadata with pve_node
	var metadata json.RawMessage
	if pveNodeName != "" {
		metaBytes, _ := json.Marshal(map[string]string{"pve_node": pveNodeName})
		metadata = metaBytes
	}

	// 6. Save node via existing Create flow
	createReq := model.CreateNodeRequest{
		Name:           req.Name,
		Type:           req.Type,
		Hostname:       req.Hostname,
		Port:           port,
		APITokenID:     tokenID,
		APITokenSecret: tokenSecret,
		SSHPort:        sshPort,
		SSHUser:        "root",
		SSHPrivateKey:  privKey,
		Metadata:       metadata,
	}

	node, err := s.Create(ctx, createReq)
	if err != nil {
		return nil, fmt.Errorf("save node: %w", err)
	}

	slog.Info("node onboarded", slog.String("name", node.Name), slog.String("id", node.ID.String()))
	return node, nil
}

// checkNodeOnline performs an immediate status check and updates the node if online.
func (s *Service) checkNodeOnline(ctx context.Context, node *model.Node) {
	client, err := s.clientFactory.CreateClient(node)
	if err != nil {
		return
	}

	_, err = client.GetVersion(ctx)
	if err != nil {
		return
	}

	now := time.Now().UTC()
	node.IsOnline = true
	node.LastSeen = &now
	if updateErr := s.nodeRepo.Update(ctx, node); updateErr != nil {
		slog.Warn("failed to update node online status", slog.Any("error", updateErr))
	}
}

func generateEd25519KeyPair() (publicKey, privateKey string, err error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("generate ed25519 key: %w", err)
	}

	sshPub, err := gossh.NewPublicKey(pub)
	if err != nil {
		return "", "", fmt.Errorf("create ssh public key: %w", err)
	}

	pubKeyStr := string(gossh.MarshalAuthorizedKey(sshPub))

	privPEM, err := gossh.MarshalPrivateKey(priv, "")
	if err != nil {
		return "", "", fmt.Errorf("marshal private key: %w", err)
	}
	privKeyStr := string(pem.EncodeToMemory(privPEM))

	return pubKeyStr, privKeyStr, nil
}

// getClientAndNode retrieves the node from the DB, creates a Proxmox client,
// and resolves the correct PVE node name. Uses cached pve_node from metadata
// when available to avoid an extra API round-trip. Wraps connection errors
// with ErrNodeUnreachable so handlers can return 503 instead of 500.
func (s *Service) getClientAndNode(ctx context.Context, id uuid.UUID) (*model.Node, *proxmox.Client, string, error) {
	node, err := s.nodeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, "", err
	}

	client, err := s.clientFactory.CreateClient(node)
	if err != nil {
		return nil, nil, "", fmt.Errorf("%w: create proxmox client: %v", ErrNodeUnreachable, err)
	}

	// Fast path: use cached pve_node from metadata (skips GetNodes API call)
	if cachedNode := getCachedPVENode(node); cachedNode != "" {
		return node, client, cachedNode, nil
	}

	// Slow path: resolve via Proxmox API call
	pveNodes, err := client.GetNodes(ctx)
	if err != nil {
		return nil, nil, "", fmt.Errorf("%w: %v", ErrNodeUnreachable, err)
	}

	if len(pveNodes) == 0 {
		return nil, nil, "", fmt.Errorf("%w: no nodes found in cluster", ErrNodeUnreachable)
	}

	pveNode := ResolvePVENode(node, pveNodes)

	// Cache for future requests (fire-and-forget)
	go s.cachePVENode(context.Background(), node, pveNode)

	return node, client, pveNode, nil
}

// getCachedPVENode extracts pve_node from node metadata.
func getCachedPVENode(node *model.Node) string {
	if len(node.Metadata) == 0 {
		return ""
	}
	var meta map[string]interface{}
	if err := json.Unmarshal(node.Metadata, &meta); err != nil {
		return ""
	}
	if pn, ok := meta["pve_node"].(string); ok && pn != "" {
		return pn
	}
	return ""
}

// cachePVENode stores the resolved PVE node name in the node's metadata.
func (s *Service) cachePVENode(ctx context.Context, node *model.Node, pveNode string) {
	var meta map[string]interface{}
	if len(node.Metadata) > 0 {
		if err := json.Unmarshal(node.Metadata, &meta); err != nil {
			meta = make(map[string]interface{})
		}
	} else {
		meta = make(map[string]interface{})
	}
	meta["pve_node"] = pveNode
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return
	}
	node.Metadata = metaBytes
	if err := s.nodeRepo.Update(ctx, node); err != nil {
		slog.Warn("failed to cache pve_node", slog.Any("error", err))
	}
}

// ResolvePVENode determines which PVE cluster node corresponds to the registered node.
// Priority: 1) pve_node from metadata, 2) name match, 3) first node.
func ResolvePVENode(node *model.Node, pveNodes []string) string {
	// 1. Check metadata for stored pve_node
	if len(node.Metadata) > 0 {
		var meta map[string]interface{}
		if err := json.Unmarshal(node.Metadata, &meta); err == nil {
			if pn, ok := meta["pve_node"].(string); ok && pn != "" {
				for _, n := range pveNodes {
					if strings.EqualFold(n, pn) {
						return n
					}
				}
			}
		}
	}

	// 2. Try to match node.Name against PVE node names
	for _, n := range pveNodes {
		if strings.EqualFold(n, node.Name) {
			return n
		}
	}

	// 3. Fallback to first node
	return pveNodes[0]
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*model.Node, error) {
	return s.nodeRepo.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]model.Node, error) {
	return s.nodeRepo.List(ctx)
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, req model.UpdateNodeRequest) (*model.Node, error) {
	node, err := s.nodeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		node.Name = *req.Name
	}
	if req.Hostname != nil {
		node.Hostname = *req.Hostname
	}
	if req.Port != nil {
		node.Port = *req.Port
	}
	if req.APITokenID != nil {
		enc, err := s.encryptor.Encrypt(*req.APITokenID)
		if err != nil {
			return nil, fmt.Errorf("encrypt token id: %w", err)
		}
		node.APITokenID = enc
	}
	if req.APITokenSecret != nil {
		enc, err := s.encryptor.Encrypt(*req.APITokenSecret)
		if err != nil {
			return nil, fmt.Errorf("encrypt token secret: %w", err)
		}
		node.APITokenSecret = enc
	}
	if req.SSHPort != nil {
		node.SSHPort = *req.SSHPort
	}
	if req.SSHUser != nil {
		node.SSHUser = *req.SSHUser
	}
	if req.SSHPrivateKey != nil {
		if *req.SSHPrivateKey != "" {
			enc, err := s.encryptor.Encrypt(*req.SSHPrivateKey)
			if err != nil {
				return nil, fmt.Errorf("encrypt ssh private key: %w", err)
			}
			node.SSHPrivateKey = enc
		} else {
			node.SSHPrivateKey = ""
		}
	}
	if req.Metadata != nil {
		node.Metadata = *req.Metadata
	}

	if err := s.nodeRepo.Update(ctx, node); err != nil {
		return nil, fmt.Errorf("update node: %w", err)
	}

	return node, nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.nodeRepo.Delete(ctx, id)
}

func (s *Service) TestConnection(ctx context.Context, req model.TestConnectionRequest) *model.TestConnectionResponse {
	port := req.Port
	if port == 0 {
		port = 8006
	}

	client := s.clientFactory.CreateClientFromCredentials(req.Hostname, port, req.APITokenID, req.APITokenSecret)

	version, err := client.GetVersion(ctx)
	if err != nil {
		return &model.TestConnectionResponse{
			Success: false,
			Error:   err.Error(),
		}
	}

	nodes, err := client.GetNodes(ctx)
	nodeName := ""
	if err == nil && len(nodes) > 0 {
		nodeName = nodes[0]
	}

	return &model.TestConnectionResponse{
		Success: true,
		Version: version.Version,
		Node:    nodeName,
	}
}

func (s *Service) GetStatus(ctx context.Context, id uuid.UUID) (*proxmox.NodeStatus, error) {
	_, client, pveNode, err := s.getClientAndNode(ctx, id)
	if err != nil {
		return nil, err
	}

	status, err := client.GetNodeStatus(ctx, pveNode)
	if err != nil {
		return nil, fmt.Errorf("%w: get node status: %v", ErrNodeUnreachable, err)
	}

	// Enrich status with VM/CT counts
	vms, err := client.GetVMs(ctx, pveNode)
	if err == nil {
		for _, vm := range vms {
			if vm.Type == "qemu" {
				status.VMCount++
				if vm.Status == "running" {
					status.VMRunning++
				}
			} else if vm.Type == "lxc" {
				status.CTCount++
				if vm.Status == "running" {
					status.CTRunning++
				}
			}
		}
	}

	return status, nil
}

func (s *Service) GetVMs(ctx context.Context, id uuid.UUID) ([]proxmox.VMInfo, error) {
	node, client, pveNode, err := s.getClientAndNode(ctx, id)
	if err == nil {
		vms, vmErr := client.GetVMs(ctx, pveNode)
		if vmErr == nil {
			return vms, nil
		}
		slog.Warn("direct VM fetch failed",
			slog.String("node", node.Name), slog.String("pve_node", pveNode), slog.Any("error", vmErr))

		// Try re-resolving PVE node name
		if vms, resolved := s.retryVMsWithResolvedName(ctx, node, client, pveNode); resolved {
			return vms, nil
		}
	}

	return s.getVMsViaCluster(ctx, id)
}

// retryVMsWithResolvedName re-resolves the PVE node name when VM fetch fails.
func (s *Service) retryVMsWithResolvedName(ctx context.Context, node *model.Node, client *proxmox.Client, triedName string) ([]proxmox.VMInfo, bool) {
	pveNodes, err := client.GetNodes(ctx)
	if err != nil || len(pveNodes) == 0 {
		return nil, false
	}
	for _, pn := range pveNodes {
		if pn == triedName {
			continue
		}
		vms, vmErr := client.GetVMs(ctx, pn)
		if vmErr == nil {
			slog.Info("resolved correct PVE node name for VMs",
				slog.String("node", node.Name), slog.String("correct_pve_node", pn))
			go s.cachePVENode(context.Background(), node, pn)
			return vms, true
		}
	}
	return nil, false
}

// getVMsViaCluster fetches VMs for a node through another reachable cluster node.
func (s *Service) getVMsViaCluster(ctx context.Context, targetID uuid.UUID) ([]proxmox.VMInfo, error) {
	targetNode, err := s.nodeRepo.GetByID(ctx, targetID)
	if err != nil {
		return nil, err
	}
	targetPVENode := getCachedPVENode(targetNode)
	if targetPVENode == "" {
		targetPVENode = targetNode.Name
	}

	allNodes, err := s.nodeRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list nodes for fallback: %w", err)
	}

	var lastErr error
	for _, n := range allNodes {
		if n.ID == targetID || !n.IsOnline || n.Type != "pve" {
			continue
		}
		client, clientErr := s.clientFactory.CreateClient(&n)
		if clientErr != nil {
			continue
		}
		vms, vmErr := client.GetVMs(ctx, targetPVENode)
		if vmErr == nil {
			slog.Info("fetched VMs via cluster fallback",
				slog.String("target", targetNode.Name), slog.String("via", n.Name))
			return vms, nil
		}
		lastErr = vmErr

		// Try all PVE node names in case the target name is wrong
		pveNodes, nodesErr := client.GetNodes(ctx)
		if nodesErr != nil {
			continue
		}
		for _, pn := range pveNodes {
			if pn == targetPVENode {
				continue
			}
			vms, retryErr := client.GetVMs(ctx, pn)
			if retryErr == nil {
				slog.Info("fetched VMs via cluster fallback with resolved name",
					slog.String("target", targetNode.Name),
					slog.String("via", n.Name),
					slog.String("resolved_pve_node", pn))
				go s.cachePVENode(context.Background(), targetNode, pn)
				return vms, nil
			}
		}
	}

	detail := "no other online PVE nodes available for fallback"
	if lastErr != nil {
		detail = fmt.Sprintf("all fallback attempts failed, last error: %v", lastErr)
	}
	return nil, fmt.Errorf("%w: %s", ErrNodeUnreachable, detail)
}

func (s *Service) GetStorage(ctx context.Context, id uuid.UUID) ([]proxmox.StorageInfo, error) {
	slog.Info("GetStorage called", slog.String("node_id", id.String()))

	node, client, pveNode, err := s.getClientAndNode(ctx, id)
	if err == nil {
		slog.Info("GetStorage: resolved client and node",
			slog.String("node_name", node.Name),
			slog.String("pve_node", pveNode))

		// Try node-specific endpoint first
		storage, storErr := client.GetStorage(ctx, pveNode)
		if storErr == nil {
			slog.Info("GetStorage: SUCCESS via direct node endpoint",
				slog.String("node", node.Name),
				slog.String("pve_node", pveNode),
				slog.Int("count", len(storage)))
			return storage, nil
		}
		slog.Warn("GetStorage: direct storage fetch failed",
			slog.String("node", node.Name), slog.String("pve_node", pveNode), slog.Any("error", storErr))

		// Try cluster-level endpoint (doesn't need correct node name)
		clusterStorage, clusterErr := client.GetClusterStorages(ctx)
		if clusterErr == nil {
			slog.Info("GetStorage: SUCCESS via cluster endpoint",
				slog.String("node", node.Name),
				slog.Int("count", len(clusterStorage)))
			return clusterStorage, nil
		}
		slog.Warn("GetStorage: cluster endpoint also failed",
			slog.String("node", node.Name), slog.Any("error", clusterErr))

		// The PVE node name might be wrong - try re-resolving it
		if storage, resolved := s.retryStorageWithResolvedName(ctx, node, client, pveNode); resolved {
			slog.Info("GetStorage: SUCCESS via retry with re-resolved PVE node name",
				slog.String("node", node.Name),
				slog.Int("count", len(storage)))
			return storage, nil
		}
		slog.Warn("GetStorage: retry with re-resolved name also failed", slog.String("node", node.Name))
	} else {
		slog.Warn("GetStorage: cannot reach node, trying cluster fallback",
			slog.String("node_id", id.String()), slog.Any("error", err))
	}

	// Cluster fallback: query storage through any other reachable node
	slog.Info("GetStorage: attempting cluster fallback via other nodes", slog.String("node_id", id.String()))
	fallbackStorage, fallbackErr := s.getStorageViaCluster(ctx, id)
	if fallbackErr == nil {
		slog.Info("GetStorage: SUCCESS via cluster fallback",
			slog.String("node_id", id.String()),
			slog.Int("count", len(fallbackStorage)))
	} else {
		slog.Error("GetStorage: all methods failed",
			slog.String("node_id", id.String()),
			slog.Any("error", fallbackErr))
	}
	return fallbackStorage, fallbackErr
}

// retryStorageWithResolvedName re-resolves the PVE node name when the initial
// storage call fails. This handles the common case where the cached pve_node
// or the fallback name doesn't match the actual Proxmox node name.
func (s *Service) retryStorageWithResolvedName(ctx context.Context, node *model.Node, client *proxmox.Client, triedName string) ([]proxmox.StorageInfo, bool) {
	pveNodes, err := client.GetNodes(ctx)
	if err != nil || len(pveNodes) == 0 {
		return nil, false
	}

	slog.Info("re-resolving PVE node name for storage",
		slog.String("node", node.Name),
		slog.String("tried", triedName),
		slog.Any("available_pve_nodes", pveNodes))

	for _, pn := range pveNodes {
		if pn == triedName {
			continue
		}
		storage, storErr := client.GetStorage(ctx, pn)
		if storErr == nil {
			slog.Info("resolved correct PVE node name for storage",
				slog.String("node", node.Name),
				slog.String("correct_pve_node", pn))
			go s.cachePVENode(context.Background(), node, pn)
			return storage, true
		}
	}
	return nil, false
}

// getStorageViaCluster tries to fetch storage info for a node by connecting
// through another online node in the same Proxmox cluster.
func (s *Service) getStorageViaCluster(ctx context.Context, targetID uuid.UUID) ([]proxmox.StorageInfo, error) {
	targetNode, err := s.nodeRepo.GetByID(ctx, targetID)
	if err != nil {
		return nil, err
	}

	targetPVENode := getCachedPVENode(targetNode)
	if targetPVENode == "" {
		targetPVENode = targetNode.Name
	}

	allNodes, err := s.nodeRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list nodes for fallback: %w", err)
	}

	var lastErr error
	for _, n := range allNodes {
		if n.ID == targetID || !n.IsOnline || n.Type != "pve" {
			continue
		}

		client, clientErr := s.clientFactory.CreateClient(&n)
		if clientErr != nil {
			slog.Debug("cluster fallback: skip node (client error)",
				slog.String("node", n.Name), slog.Any("error", clientErr))
			continue
		}

		// Try the expected target PVE node name first
		storage, storErr := client.GetStorage(ctx, targetPVENode)
		if storErr == nil {
			slog.Info("fetched storage via cluster fallback",
				slog.String("target", targetNode.Name), slog.String("via", n.Name))
			if getCachedPVENode(targetNode) == "" {
				go s.cachePVENode(context.Background(), targetNode, targetPVENode)
			}
			return storage, nil
		}
		lastErr = storErr

		// The target PVE name might be wrong - enumerate cluster nodes and try each
		pveNodes, nodesErr := client.GetNodes(ctx)
		if nodesErr != nil {
			continue
		}
		for _, pn := range pveNodes {
			if pn == targetPVENode {
				continue
			}
			storage, retryErr := client.GetStorage(ctx, pn)
			if retryErr == nil {
				slog.Info("fetched storage via cluster fallback with resolved name",
					slog.String("target", targetNode.Name),
					slog.String("via", n.Name),
					slog.String("resolved_pve_node", pn))
				go s.cachePVENode(context.Background(), targetNode, pn)
				return storage, nil
			}
		}
	}

	detail := "no other online PVE nodes available for fallback"
	if lastErr != nil {
		detail = fmt.Sprintf("all fallback attempts failed, last error: %v", lastErr)
	}
	return nil, fmt.Errorf("%w: %s", ErrNodeUnreachable, detail)
}

// ClusterStorageItem represents a storage pool with its owning node info.
type ClusterStorageItem struct {
	NodeID       string  `json:"node_id"`
	NodeName     string  `json:"node_name"`
	Storage      string  `json:"storage"`
	Type         string  `json:"type"`
	Content      string  `json:"content"`
	Total        int64   `json:"total"`
	Used         int64   `json:"used"`
	Available    int64   `json:"available"`
	UsagePercent float64 `json:"usage_percent"`
	Active       bool    `json:"active"`
	Shared       bool    `json:"shared"`
}

// GetClusterStorage aggregates storage from all online PVE nodes.
func (s *Service) GetClusterStorage(ctx context.Context) ([]ClusterStorageItem, error) {
	allNodes, err := s.nodeRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list nodes: %w", err)
	}

	type nodeResult struct {
		items []ClusterStorageItem
		err   error
	}

	results := make(chan nodeResult, len(allNodes))
	var wg sync.WaitGroup

	for i := range allNodes {
		n := allNodes[i]
		if !n.IsOnline || n.Type != "pve" {
			continue
		}
		wg.Add(1)
		go func(node model.Node) {
			defer wg.Done()
			storage, storErr := s.GetStorage(ctx, node.ID)
			if storErr != nil {
				slog.Warn("GetClusterStorage: failed for node",
					slog.String("node", node.Name), slog.Any("error", storErr))
				results <- nodeResult{err: storErr}
				return
			}
			items := make([]ClusterStorageItem, 0, len(storage))
			for _, st := range storage {
				items = append(items, ClusterStorageItem{
					NodeID:       node.ID.String(),
					NodeName:     node.Name,
					Storage:      st.Storage,
					Type:         st.Type,
					Content:      st.Content,
					Total:        st.Total,
					Used:         st.Used,
					Available:    st.Available,
					UsagePercent: st.UsagePercent,
					Active:       st.Active,
					Shared:       st.Shared,
				})
			}
			results <- nodeResult{items: items}
		}(n)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var allItems []ClusterStorageItem
	for res := range results {
		if res.err == nil {
			allItems = append(allItems, res.items...)
		}
	}

	if allItems == nil {
		allItems = []ClusterStorageItem{}
	}

	return allItems, nil
}

func (s *Service) GetNetworkInterfaces(ctx context.Context, id uuid.UUID) ([]NetworkInterfaceWithAlias, error) {
	_, client, pveNode, err := s.getClientAndNode(ctx, id)
	if err != nil {
		return nil, err
	}

	ifaces, err := client.GetNetworkInterfaces(ctx, pveNode)
	if err != nil {
		return nil, fmt.Errorf("%w: get network interfaces: %v", ErrNodeUnreachable, err)
	}

	// Get aliases for this node
	aliases, err := s.aliasRepo.GetByNode(ctx, id)
	if err != nil {
		slog.Warn("failed to get network aliases", slog.Any("error", err))
		aliases = nil
	}

	aliasMap := make(map[string]model.NetworkAlias)
	for _, a := range aliases {
		aliasMap[a.InterfaceName] = a
	}

	result := make([]NetworkInterfaceWithAlias, 0, len(ifaces))
	for _, iface := range ifaces {
		entry := NetworkInterfaceWithAlias{
			NetworkInterface: iface,
		}
		if alias, ok := aliasMap[iface.Iface]; ok {
			entry.DisplayName = alias.DisplayName
			entry.Description = alias.Description
			entry.Color = alias.Color
		}
		result = append(result, entry)
	}

	return result, nil
}

func (s *Service) SetAlias(ctx context.Context, nodeID uuid.UUID, ifaceName string, req model.UpsertAliasRequest) error {
	// Verify node exists
	_, err := s.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		return err
	}

	alias := &model.NetworkAlias{
		NodeID:        nodeID,
		InterfaceName: ifaceName,
		DisplayName:   req.DisplayName,
		Description:   req.Description,
		Color:         req.Color,
	}

	return s.aliasRepo.Upsert(ctx, alias)
}

func (s *Service) GetDisks(ctx context.Context, id uuid.UUID) ([]proxmox.DiskInfo, error) {
	_, client, pveNode, err := s.getClientAndNode(ctx, id)
	if err != nil {
		return nil, err
	}

	disks, err := client.GetDisks(ctx, pveNode)
	if err != nil {
		return nil, fmt.Errorf("%w: get disks: %v", ErrNodeUnreachable, err)
	}

	return disks, nil
}

func (s *Service) StartVM(ctx context.Context, nodeID uuid.UUID, vmid int, vmType string) (string, error) {
	_, client, pveNode, err := s.getClientAndNode(ctx, nodeID)
	if err != nil {
		return "", err
	}

	return client.StartVM(ctx, pveNode, vmid, vmType)
}

func (s *Service) StopVM(ctx context.Context, nodeID uuid.UUID, vmid int, vmType string) (string, error) {
	_, client, pveNode, err := s.getClientAndNode(ctx, nodeID)
	if err != nil {
		return "", err
	}

	return client.StopVM(ctx, pveNode, vmid, vmType)
}

func (s *Service) ShutdownVM(ctx context.Context, nodeID uuid.UUID, vmid int, vmType string) (string, error) {
	_, client, pveNode, err := s.getClientAndNode(ctx, nodeID)
	if err != nil {
		return "", err
	}

	return client.ShutdownVM(ctx, pveNode, vmid, vmType)
}

func (s *Service) SuspendVM(ctx context.Context, nodeID uuid.UUID, vmid int) (string, error) {
	_, client, pveNode, err := s.getClientAndNode(ctx, nodeID)
	if err != nil {
		return "", err
	}

	return client.SuspendVM(ctx, pveNode, vmid)
}

func (s *Service) ResumeVM(ctx context.Context, nodeID uuid.UUID, vmid int) (string, error) {
	_, client, pveNode, err := s.getClientAndNode(ctx, nodeID)
	if err != nil {
		return "", err
	}

	return client.ResumeVM(ctx, pveNode, vmid)
}

func (s *Service) ListSnapshots(ctx context.Context, nodeID uuid.UUID, vmid int, vmType string) ([]proxmox.SnapshotInfo, error) {
	_, client, pveNode, err := s.getClientAndNode(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	return client.ListSnapshots(ctx, pveNode, vmid, vmType)
}

func (s *Service) CreateSnapshot(ctx context.Context, nodeID uuid.UUID, vmid int, vmType string, name string, description string, includeRAM bool) (string, error) {
	_, client, pveNode, err := s.getClientAndNode(ctx, nodeID)
	if err != nil {
		return "", err
	}

	return client.CreateSnapshot(ctx, pveNode, vmid, vmType, name, description, includeRAM)
}

func (s *Service) DeleteSnapshot(ctx context.Context, nodeID uuid.UUID, vmid int, vmType string, snapname string) (string, error) {
	_, client, pveNode, err := s.getClientAndNode(ctx, nodeID)
	if err != nil {
		return "", err
	}

	return client.DeleteSnapshot(ctx, pveNode, vmid, vmType, snapname)
}

func (s *Service) RollbackSnapshot(ctx context.Context, nodeID uuid.UUID, vmid int, vmType string, snapname string) (string, error) {
	_, client, pveNode, err := s.getClientAndNode(ctx, nodeID)
	if err != nil {
		return "", err
	}

	return client.RollbackSnapshot(ctx, pveNode, vmid, vmType, snapname)
}

func (s *Service) GetVNCProxy(ctx context.Context, nodeID uuid.UUID, vmid int, vmType string) (*proxmox.VNCProxyResponse, error) {
	_, client, pveNode, err := s.getClientAndNode(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	return client.GetVNCProxy(ctx, pveNode, vmid, vmType)
}

func (s *Service) CreateVzdump(ctx context.Context, nodeID uuid.UUID, vmid int, opts proxmox.VzdumpOptions) (string, error) {
	_, client, pveNode, err := s.getClientAndNode(ctx, nodeID)
	if err != nil {
		return "", err
	}

	return client.CreateVzdump(ctx, pveNode, vmid, opts)
}

func (s *Service) RunSSHCommand(ctx context.Context, nodeID uuid.UUID, command string) (*ssh.CommandResult, error) {
	node, err := s.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	sshCfg, err := s.buildSSHConfig(node)
	if err != nil {
		return nil, fmt.Errorf("build ssh config: %w", err)
	}

	client, err := s.sshPool.Get(nodeID.String(), sshCfg)
	if err != nil {
		return nil, fmt.Errorf("get ssh client: %w", err)
	}
	defer s.sshPool.Return(nodeID.String(), client)

	return client.RunCommand(ctx, command)
}

func (s *Service) BulkVMAction(ctx context.Context, nodeID uuid.UUID, req model.BulkVMRequest) ([]model.BulkVMResult, error) {
	_, client, pveNode, err := s.getClientAndNode(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	// Get all VMs to determine types
	vms, err := client.GetVMs(ctx, pveNode)
	if err != nil {
		return nil, fmt.Errorf("get VMs: %w", err)
	}
	vmTypeMap := make(map[int]string, len(vms))
	for _, vm := range vms {
		vmTypeMap[vm.VMID] = vm.Type
	}

	results := make([]model.BulkVMResult, len(req.VMIDs))
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i, vmid := range req.VMIDs {
		wg.Add(1)
		go func(idx, id int) {
			defer wg.Done()

			vmType := vmTypeMap[id]
			if vmType == "" {
				vmType = "qemu"
			}

			var upid string
			var actionErr error
			switch req.Action {
			case "start":
				upid, actionErr = client.StartVM(ctx, pveNode, id, vmType)
			case "stop":
				upid, actionErr = client.StopVM(ctx, pveNode, id, vmType)
			case "shutdown":
				upid, actionErr = client.ShutdownVM(ctx, pveNode, id, vmType)
			default:
				actionErr = fmt.Errorf("unknown action: %s", req.Action)
			}

			mu.Lock()
			defer mu.Unlock()
			if actionErr != nil {
				results[idx] = model.BulkVMResult{VMID: id, Success: false, Error: actionErr.Error()}
			} else {
				results[idx] = model.BulkVMResult{VMID: id, Success: true, UPID: upid}
			}
		}(i, vmid)
	}

	wg.Wait()
	return results, nil
}

func (s *Service) SyncTagsFromProxmox(ctx context.Context, nodeID uuid.UUID) (int, error) {
	_, client, pveNode, err := s.getClientAndNode(ctx, nodeID)
	if err != nil {
		return 0, err
	}

	vms, err := client.GetVMs(ctx, pveNode)
	if err != nil {
		return 0, fmt.Errorf("get VMs: %w", err)
	}

	// Collect unique tags from all VMs
	tagSet := make(map[string]struct{})
	for _, vm := range vms {
		if vm.Tags == "" {
			continue
		}
		// Proxmox separates tags with semicolons
		for _, tag := range strings.Split(vm.Tags, ";") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tagSet[tag] = struct{}{}
			}
		}
	}

	// Get existing tags
	existingTags, err := s.tagRepo.List(ctx)
	if err != nil {
		return 0, fmt.Errorf("list existing tags: %w", err)
	}
	existingMap := make(map[string]struct{}, len(existingTags))
	for _, t := range existingTags {
		existingMap[strings.ToLower(t.Name)] = struct{}{}
	}

	// Create missing tags
	created := 0
	for tagName := range tagSet {
		if _, exists := existingMap[strings.ToLower(tagName)]; exists {
			continue
		}
		tag := &model.Tag{
			Name:  tagName,
			Color: "#3b82f6",
		}
		if err := s.tagRepo.Create(ctx, tag); err != nil {
			slog.Warn("failed to create synced tag", slog.String("tag", tagName), slog.Any("error", err))
			continue
		}
		created++
	}

	slog.Info("tags synced from proxmox", slog.String("node_id", nodeID.String()), slog.Int("created", created))
	return created, nil
}

// DiagnoseConnectivity performs detailed connectivity checks for a node and
// returns diagnostic information useful for troubleshooting 503 errors.
func (s *Service) DiagnoseConnectivity(ctx context.Context, id uuid.UUID) map[string]interface{} {
	result := map[string]interface{}{
		"node_id": id.String(),
	}

	node, err := s.nodeRepo.GetByID(ctx, id)
	if err != nil {
		result["error"] = fmt.Sprintf("node not found: %v", err)
		return result
	}
	result["node_name"] = node.Name
	result["hostname"] = node.Hostname
	result["port"] = node.Port
	result["is_online"] = node.IsOnline
	result["cached_pve_node"] = getCachedPVENode(node)

	client, err := s.clientFactory.CreateClient(node)
	if err != nil {
		result["client_error"] = fmt.Sprintf("failed to create client: %v", err)
		return result
	}
	result["client_created"] = true

	// Test basic connectivity
	version, err := client.GetVersion(ctx)
	if err != nil {
		result["version_error"] = err.Error()
		result["api_reachable"] = false
		return result
	}
	result["api_reachable"] = true
	result["pve_version"] = version.Version

	// List PVE nodes
	pveNodes, err := client.GetNodes(ctx)
	if err != nil {
		result["get_nodes_error"] = err.Error()
		return result
	}
	result["pve_cluster_nodes"] = pveNodes

	// Try storage for each PVE node
	storageResults := make(map[string]interface{})
	for _, pn := range pveNodes {
		storage, storErr := client.GetStorage(ctx, pn)
		if storErr != nil {
			storageResults[pn] = map[string]interface{}{"error": storErr.Error()}
		} else {
			names := make([]string, len(storage))
			for i, s := range storage {
				names[i] = fmt.Sprintf("%s (%s, %s)", s.Storage, s.Type, s.Content)
			}
			storageResults[pn] = map[string]interface{}{"count": len(storage), "storages": names}
		}
	}
	result["storage_by_pve_node"] = storageResults

	return result
}

// DebugStorage returns raw Proxmox API responses for storage endpoints.
// This bypasses all parsing/filtering so you can see exactly what PVE returns.
func (s *Service) DebugStorage(ctx context.Context, id uuid.UUID) (map[string]interface{}, error) {
	result := map[string]interface{}{
		"node_id":   id.String(),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	node, err := s.nodeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	result["node_name"] = node.Name
	result["hostname"] = node.Hostname
	result["port"] = node.Port
	result["cached_pve_node"] = getCachedPVENode(node)

	client, err := s.clientFactory.CreateClient(node)
	if err != nil {
		result["error"] = fmt.Sprintf("failed to create client: %v", err)
		return result, nil
	}

	// Get PVE node list
	pveNodes, err := client.GetNodes(ctx)
	if err != nil {
		result["get_nodes_error"] = err.Error()
		return result, nil
	}
	result["pve_nodes"] = pveNodes

	// For each PVE node, call GetStorageRaw
	perNodeRaw := make(map[string]interface{})
	for _, pn := range pveNodes {
		raw, rawErr := client.GetStorageRaw(ctx, pn)
		if rawErr != nil {
			perNodeRaw[pn] = map[string]interface{}{"error": rawErr.Error()}
		} else {
			// Parse into generic structure so it renders as proper JSON
			var parsed interface{}
			if jsonErr := json.Unmarshal(raw, &parsed); jsonErr != nil {
				perNodeRaw[pn] = map[string]interface{}{"raw_string": string(raw), "parse_error": jsonErr.Error()}
			} else {
				perNodeRaw[pn] = map[string]interface{}{"data": parsed, "raw_bytes": len(raw)}
			}
		}
	}
	result["storage_per_node_raw"] = perNodeRaw

	// Cluster-level storage raw
	clusterRaw, clusterErr := client.GetClusterResourcesRaw(ctx)
	if clusterErr != nil {
		result["cluster_resources_error"] = clusterErr.Error()
	} else {
		var parsed interface{}
		if jsonErr := json.Unmarshal(clusterRaw, &parsed); jsonErr != nil {
			result["cluster_resources_raw"] = map[string]interface{}{"raw_string": string(clusterRaw), "parse_error": jsonErr.Error()}
		} else {
			result["cluster_resources_raw"] = map[string]interface{}{"data": parsed, "raw_bytes": len(clusterRaw)}
		}
	}

	// Also include the parsed result from the normal GetStorage path for comparison
	storage, storErr := s.GetStorage(ctx, id)
	if storErr != nil {
		result["parsed_storage_error"] = storErr.Error()
	} else {
		result["parsed_storage_count"] = len(storage)
		result["parsed_storage"] = storage
	}

	return result, nil
}

func (s *Service) GetNodeRRDData(ctx context.Context, id uuid.UUID, timeframe string) ([]proxmox.RRDDataPoint, error) {
	_, client, pveNode, err := s.getClientAndNode(ctx, id)
	if err != nil {
		return nil, err
	}

	return client.GetNodeRRDData(ctx, pveNode, timeframe)
}

func (s *Service) GetVMRRDData(ctx context.Context, nodeID uuid.UUID, vmid int, vmType string, timeframe string) ([]proxmox.VMRRDDataPoint, error) {
	_, client, pveNode, err := s.getClientAndNode(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	return client.GetVMRRDData(ctx, pveNode, vmid, vmType, timeframe)
}

func (s *Service) buildSSHConfig(node *model.Node) (ssh.SSHConfig, error) {
	cfg := ssh.SSHConfig{
		Host: node.Hostname,
		Port: node.SSHPort,
		User: node.SSHUser,
	}

	if node.SSHPrivateKey != "" {
		decrypted, err := s.encryptor.Decrypt(node.SSHPrivateKey)
		if err != nil {
			return ssh.SSHConfig{}, fmt.Errorf("decrypt ssh private key: %w", err)
		}
		cfg.PrivateKey = decrypted
	}

	return cfg, nil
}
