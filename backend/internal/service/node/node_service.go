package node

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"encoding/pem"
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

	// 5. Save node via existing Create flow
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
	node, err := s.nodeRepo.GetByID(ctx, id)
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

	return client.GetNodeStatus(ctx, nodes[0])
}

func (s *Service) GetVMs(ctx context.Context, id uuid.UUID) ([]proxmox.VMInfo, error) {
	node, err := s.nodeRepo.GetByID(ctx, id)
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

	return client.GetVMs(ctx, nodes[0])
}

func (s *Service) GetStorage(ctx context.Context, id uuid.UUID) ([]proxmox.StorageInfo, error) {
	node, err := s.nodeRepo.GetByID(ctx, id)
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

	return client.GetStorage(ctx, nodes[0])
}

func (s *Service) GetNetworkInterfaces(ctx context.Context, id uuid.UUID) ([]NetworkInterfaceWithAlias, error) {
	node, err := s.nodeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	client, err := s.clientFactory.CreateClient(node)
	if err != nil {
		return nil, fmt.Errorf("create proxmox client: %w", err)
	}

	pveNodes, err := client.GetNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("get nodes: %w", err)
	}

	if len(pveNodes) == 0 {
		return nil, fmt.Errorf("no nodes found")
	}

	ifaces, err := client.GetNetworkInterfaces(ctx, pveNodes[0])
	if err != nil {
		return nil, fmt.Errorf("get network interfaces: %w", err)
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
	node, err := s.nodeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	client, err := s.clientFactory.CreateClient(node)
	if err != nil {
		return nil, fmt.Errorf("create proxmox client: %w", err)
	}

	pveNodes, err := client.GetNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("get nodes: %w", err)
	}

	if len(pveNodes) == 0 {
		return nil, fmt.Errorf("no nodes found")
	}

	return client.GetDisks(ctx, pveNodes[0])
}

func (s *Service) StartVM(ctx context.Context, nodeID uuid.UUID, vmid int, vmType string) (string, error) {
	node, err := s.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		return "", err
	}

	client, err := s.clientFactory.CreateClient(node)
	if err != nil {
		return "", fmt.Errorf("create proxmox client: %w", err)
	}

	nodes, err := client.GetNodes(ctx)
	if err != nil {
		return "", fmt.Errorf("get nodes: %w", err)
	}

	if len(nodes) == 0 {
		return "", fmt.Errorf("no nodes found")
	}

	return client.StartVM(ctx, nodes[0], vmid, vmType)
}

func (s *Service) StopVM(ctx context.Context, nodeID uuid.UUID, vmid int, vmType string) (string, error) {
	node, err := s.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		return "", err
	}

	client, err := s.clientFactory.CreateClient(node)
	if err != nil {
		return "", fmt.Errorf("create proxmox client: %w", err)
	}

	nodes, err := client.GetNodes(ctx)
	if err != nil {
		return "", fmt.Errorf("get nodes: %w", err)
	}

	if len(nodes) == 0 {
		return "", fmt.Errorf("no nodes found")
	}

	return client.StopVM(ctx, nodes[0], vmid, vmType)
}

func (s *Service) ShutdownVM(ctx context.Context, nodeID uuid.UUID, vmid int, vmType string) (string, error) {
	node, err := s.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		return "", err
	}

	client, err := s.clientFactory.CreateClient(node)
	if err != nil {
		return "", fmt.Errorf("create proxmox client: %w", err)
	}

	nodes, err := client.GetNodes(ctx)
	if err != nil {
		return "", fmt.Errorf("get nodes: %w", err)
	}

	if len(nodes) == 0 {
		return "", fmt.Errorf("no nodes found")
	}

	return client.ShutdownVM(ctx, nodes[0], vmid, vmType)
}

func (s *Service) SuspendVM(ctx context.Context, nodeID uuid.UUID, vmid int) (string, error) {
	node, err := s.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		return "", err
	}

	client, err := s.clientFactory.CreateClient(node)
	if err != nil {
		return "", fmt.Errorf("create proxmox client: %w", err)
	}

	nodes, err := client.GetNodes(ctx)
	if err != nil {
		return "", fmt.Errorf("get nodes: %w", err)
	}

	if len(nodes) == 0 {
		return "", fmt.Errorf("no nodes found")
	}

	return client.SuspendVM(ctx, nodes[0], vmid)
}

func (s *Service) ResumeVM(ctx context.Context, nodeID uuid.UUID, vmid int) (string, error) {
	node, err := s.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		return "", err
	}

	client, err := s.clientFactory.CreateClient(node)
	if err != nil {
		return "", fmt.Errorf("create proxmox client: %w", err)
	}

	nodes, err := client.GetNodes(ctx)
	if err != nil {
		return "", fmt.Errorf("get nodes: %w", err)
	}

	if len(nodes) == 0 {
		return "", fmt.Errorf("no nodes found")
	}

	return client.ResumeVM(ctx, nodes[0], vmid)
}

func (s *Service) ListSnapshots(ctx context.Context, nodeID uuid.UUID, vmid int, vmType string) ([]proxmox.SnapshotInfo, error) {
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

	return client.ListSnapshots(ctx, nodes[0], vmid, vmType)
}

func (s *Service) CreateSnapshot(ctx context.Context, nodeID uuid.UUID, vmid int, vmType string, name string, description string, includeRAM bool) (string, error) {
	node, err := s.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		return "", err
	}

	client, err := s.clientFactory.CreateClient(node)
	if err != nil {
		return "", fmt.Errorf("create proxmox client: %w", err)
	}

	nodes, err := client.GetNodes(ctx)
	if err != nil {
		return "", fmt.Errorf("get nodes: %w", err)
	}

	if len(nodes) == 0 {
		return "", fmt.Errorf("no nodes found")
	}

	return client.CreateSnapshot(ctx, nodes[0], vmid, vmType, name, description, includeRAM)
}

func (s *Service) DeleteSnapshot(ctx context.Context, nodeID uuid.UUID, vmid int, vmType string, snapname string) (string, error) {
	node, err := s.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		return "", err
	}

	client, err := s.clientFactory.CreateClient(node)
	if err != nil {
		return "", fmt.Errorf("create proxmox client: %w", err)
	}

	nodes, err := client.GetNodes(ctx)
	if err != nil {
		return "", fmt.Errorf("get nodes: %w", err)
	}

	if len(nodes) == 0 {
		return "", fmt.Errorf("no nodes found")
	}

	return client.DeleteSnapshot(ctx, nodes[0], vmid, vmType, snapname)
}

func (s *Service) RollbackSnapshot(ctx context.Context, nodeID uuid.UUID, vmid int, vmType string, snapname string) (string, error) {
	node, err := s.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		return "", err
	}

	client, err := s.clientFactory.CreateClient(node)
	if err != nil {
		return "", fmt.Errorf("create proxmox client: %w", err)
	}

	nodes, err := client.GetNodes(ctx)
	if err != nil {
		return "", fmt.Errorf("get nodes: %w", err)
	}

	if len(nodes) == 0 {
		return "", fmt.Errorf("no nodes found")
	}

	return client.RollbackSnapshot(ctx, nodes[0], vmid, vmType, snapname)
}

func (s *Service) GetVNCProxy(ctx context.Context, nodeID uuid.UUID, vmid int, vmType string) (*proxmox.VNCProxyResponse, error) {
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

	return client.GetVNCProxy(ctx, nodes[0], vmid, vmType)
}

func (s *Service) CreateVzdump(ctx context.Context, nodeID uuid.UUID, vmid int, opts proxmox.VzdumpOptions) (string, error) {
	node, err := s.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		return "", err
	}

	client, err := s.clientFactory.CreateClient(node)
	if err != nil {
		return "", fmt.Errorf("create proxmox client: %w", err)
	}

	nodes, err := client.GetNodes(ctx)
	if err != nil {
		return "", fmt.Errorf("get nodes: %w", err)
	}

	if len(nodes) == 0 {
		return "", fmt.Errorf("no nodes found")
	}

	return client.CreateVzdump(ctx, nodes[0], vmid, opts)
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
	pveNode := nodes[0]

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
	node, err := s.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		return 0, err
	}

	client, err := s.clientFactory.CreateClient(node)
	if err != nil {
		return 0, fmt.Errorf("create proxmox client: %w", err)
	}

	pveNodes, err := client.GetNodes(ctx)
	if err != nil {
		return 0, fmt.Errorf("get nodes: %w", err)
	}
	if len(pveNodes) == 0 {
		return 0, fmt.Errorf("no nodes found")
	}

	vms, err := client.GetVMs(ctx, pveNodes[0])
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
