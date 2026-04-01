package sshkeys

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"encoding/pem"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/crypto"
	"github.com/antigravity/prometheus/internal/ssh"
	"github.com/google/uuid"
	gossh "golang.org/x/crypto/ssh"
)

// TrustResult describes the outcome of distributing a key to all nodes.
type TrustResult struct {
	DistributedTo []string    `json:"distributed_to"`
	Failed        []TrustFail `json:"failed,omitempty"`
}

// TrustFail holds one failed distribution attempt.
type TrustFail struct {
	Node  string `json:"node"`
	Error string `json:"error"`
}

type Service struct {
	keyRepo   repository.SSHKeyRepository
	nodeRepo  repository.NodeRepository
	encryptor *crypto.Encryptor
	sshPool   *ssh.Pool
}

func NewService(
	keyRepo repository.SSHKeyRepository,
	nodeRepo repository.NodeRepository,
	encryptor *crypto.Encryptor,
	sshPool *ssh.Pool,
) *Service {
	return &Service{
		keyRepo:   keyRepo,
		nodeRepo:  nodeRepo,
		encryptor: encryptor,
		sshPool:   sshPool,
	}
}

func (s *Service) GenerateKeyPair(ctx context.Context, nodeID uuid.UUID, req model.CreateSSHKeyRequest) (*model.SSHKey, error) {
	if _, err := s.nodeRepo.GetByID(ctx, nodeID); err != nil {
		return nil, fmt.Errorf("get node: %w", err)
	}

	keyType := req.KeyType
	if keyType == "" {
		keyType = "ed25519"
	}

	var pubKey, privKey string
	var genErr error
	switch keyType {
	case "rsa":
		pubKey, privKey, genErr = generateRSAKeyPair()
	default:
		keyType = "ed25519"
		pubKey, privKey, genErr = generateEd25519KeyPair()
	}
	if genErr != nil {
		return nil, fmt.Errorf("generate key pair: %w", genErr)
	}

	// Compute fingerprint
	parsed, _, _, _, err := gossh.ParseAuthorizedKey([]byte(pubKey))
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}
	fingerprint := gossh.FingerprintSHA256(parsed)

	// Encrypt private key for storage
	encryptedPrivKey, err := s.encryptor.Encrypt(privKey)
	if err != nil {
		return nil, fmt.Errorf("encrypt private key: %w", err)
	}

	var expiresAt *time.Time
	if req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			return nil, fmt.Errorf("parse expires_at: %w", err)
		}
		expiresAt = &t
	}

	key := &model.SSHKey{
		NodeID:      nodeID,
		Name:        req.Name,
		KeyType:     keyType,
		PublicKey:   pubKey,
		PrivateKey:  encryptedPrivKey,
		Fingerprint: fingerprint,
		IsDeployed:  false,
		ExpiresAt:   expiresAt,
	}

	if err := s.keyRepo.Create(ctx, key); err != nil {
		return nil, fmt.Errorf("create ssh key: %w", err)
	}

	if req.Deploy {
		if err := s.deployKey(ctx, nodeID, key); err != nil {
			slog.Warn("auto-deploy failed", slog.Any("error", err))
		}
	}

	return key, nil
}

func (s *Service) DeployKey(ctx context.Context, keyID uuid.UUID) error {
	key, err := s.keyRepo.GetByID(ctx, keyID)
	if err != nil {
		return fmt.Errorf("get key: %w", err)
	}
	return s.deployKey(ctx, key.NodeID, key)
}

func (s *Service) deployKey(ctx context.Context, nodeID uuid.UUID, key *model.SSHKey) error {
	node, err := s.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		return fmt.Errorf("get node: %w", err)
	}

	privateKey, err := s.encryptor.Decrypt(node.SSHPrivateKey)
	if err != nil {
		return fmt.Errorf("decrypt node ssh key: %w", err)
	}

	sshClient, err := s.sshPool.Get(nodeID.String(), ssh.SSHConfig{
		Host:       node.Hostname,
		Port:       node.SSHPort,
		User:       node.SSHUser,
		PrivateKey: privateKey,
		HostKey:    node.SSHHostKey,
	})
	if err != nil {
		return fmt.Errorf("ssh connection: %w", err)
	}
	defer s.sshPool.Return(nodeID.String(), sshClient)

	// Add public key to authorized_keys (idempotent – skip if already present)
	pubKey := strings.TrimSpace(key.PublicKey)
	cmd := fmt.Sprintf(
		`mkdir -p ~/.ssh && chmod 700 ~/.ssh && grep -qxF %q ~/.ssh/authorized_keys 2>/dev/null || echo %q >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys`,
		pubKey, pubKey,
	)
	result, err := sshClient.RunCommand(ctx, cmd)
	if err != nil {
		return fmt.Errorf("deploy key: %w", err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("deploy key failed: %s", result.Stderr)
	}

	now := time.Now().UTC()
	key.IsDeployed = true
	key.DeployedAt = &now
	return s.keyRepo.Update(ctx, key)
}

func (s *Service) RotateKey(ctx context.Context, nodeID uuid.UUID) (*model.SSHKey, error) {
	// Generate new key
	newKey, err := s.GenerateKeyPair(ctx, nodeID, model.CreateSSHKeyRequest{
		Name:   fmt.Sprintf("rotated-%s", time.Now().Format("2006-01-02")),
		Deploy: true,
	})
	if err != nil {
		return nil, fmt.Errorf("generate new key: %w", err)
	}

	slog.Info("key rotated",
		slog.String("node_id", nodeID.String()),
		slog.String("key_id", newKey.ID.String()),
	)

	return newKey, nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*model.SSHKey, error) {
	return s.keyRepo.GetByID(ctx, id)
}

// MutualTrustResult describes the outcome of establishing mutual SSH trust.
type MutualTrustResult struct {
	Distributed []string    `json:"distributed"`
	Failed      []TrustFail `json:"failed,omitempty"`
}

// EstablishMutualTrust SSHs into every node, reads (or generates) root's
// ed25519 public key, and distributes it to all other nodes' authorized_keys.
func (s *Service) EstablishMutualTrust(ctx context.Context) (*MutualTrustResult, error) {
	nodes, err := s.nodeRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list nodes: %w", err)
	}

	result := &MutualTrustResult{}

	type nodeKey struct {
		node   model.Node
		pubKey string
	}
	var nodeKeys []nodeKey

	for i := range nodes {
		node := &nodes[i]
		pubKey, err := s.getOrCreateNodePublicKey(ctx, node)
		if err != nil {
			slog.Warn("trust: get public key failed", slog.String("node", node.Name), slog.Any("error", err))
			result.Failed = append(result.Failed, TrustFail{
				Node:  node.Name,
				Error: fmt.Sprintf("Schlüssel auslesen: %v", err),
			})
			continue
		}
		nodeKeys = append(nodeKeys, nodeKey{node: *node, pubKey: pubKey})
	}

	for _, src := range nodeKeys {
		for _, tgt := range nodeKeys {
			if src.node.ID == tgt.node.ID {
				continue
			}
			if err := s.deployPublicKeyToNode(ctx, &tgt.node, src.pubKey); err != nil {
				label := fmt.Sprintf("%s → %s", src.node.Name, tgt.node.Name)
				slog.Warn("trust: distribute failed", slog.String("pair", label), slog.Any("error", err))
				result.Failed = append(result.Failed, TrustFail{Node: label, Error: err.Error()})
			} else {
				result.Distributed = append(result.Distributed, fmt.Sprintf("%s → %s", src.node.Name, tgt.node.Name))
			}
		}
	}

	return result, nil
}

// getOrCreateNodePublicKey SSHs into a node and returns root's ed25519 public
// key, generating a key pair first if none exists.
func (s *Service) getOrCreateNodePublicKey(ctx context.Context, node *model.Node) (string, error) {
	privateKey, err := s.encryptor.Decrypt(node.SSHPrivateKey)
	if err != nil {
		return "", fmt.Errorf("decrypt node ssh key: %w", err)
	}

	sshClient, err := s.sshPool.Get(node.ID.String(), ssh.SSHConfig{
		Host:       node.Hostname,
		Port:       node.SSHPort,
		User:       node.SSHUser,
		PrivateKey: privateKey,
		HostKey:    node.SSHHostKey,
	})
	if err != nil {
		return "", fmt.Errorf("ssh connection: %w", err)
	}
	defer s.sshPool.Return(node.ID.String(), sshClient)

	cmd := `[ -f ~/.ssh/id_ed25519.pub ] || ssh-keygen -t ed25519 -f ~/.ssh/id_ed25519 -N "" -q 2>/dev/null; cat ~/.ssh/id_ed25519.pub`
	res, err := sshClient.RunCommand(ctx, cmd)
	if err != nil {
		return "", fmt.Errorf("get public key: %w", err)
	}
	if res.ExitCode != 0 {
		return "", fmt.Errorf("get public key failed: %s", res.Stderr)
	}

	pubKey := strings.TrimSpace(res.Stdout)
	if pubKey == "" {
		return "", fmt.Errorf("empty public key returned from node")
	}
	return pubKey, nil
}

func (s *Service) ListByNode(ctx context.Context, nodeID uuid.UUID) ([]model.SSHKey, error) {
	return s.keyRepo.ListByNode(ctx, nodeID)
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.keyRepo.Delete(ctx, id)
}

func (s *Service) GetExpiringSoon(ctx context.Context, before time.Time) ([]model.SSHKey, error) {
	return s.keyRepo.GetExpiringSoon(ctx, before)
}

func (s *Service) CreateRotationSchedule(ctx context.Context, nodeID uuid.UUID, req model.CreateRotationScheduleRequest) (*model.SSHKeyRotationSchedule, error) {
	nextRotation := time.Now().UTC().Add(time.Duration(req.IntervalDays) * 24 * time.Hour)
	sched := &model.SSHKeyRotationSchedule{
		NodeID:         nodeID,
		IntervalDays:   req.IntervalDays,
		IsActive:       req.IsActive,
		NextRotationAt: &nextRotation,
	}
	if err := s.keyRepo.CreateRotationSchedule(ctx, sched); err != nil {
		return nil, fmt.Errorf("create rotation schedule: %w", err)
	}
	return sched, nil
}

func (s *Service) GetRotationSchedule(ctx context.Context, nodeID uuid.UUID) (*model.SSHKeyRotationSchedule, error) {
	return s.keyRepo.GetRotationScheduleByNode(ctx, nodeID)
}

func (s *Service) ListDueRotations(ctx context.Context) ([]model.SSHKeyRotationSchedule, error) {
	return s.keyRepo.ListDueRotations(ctx, time.Now().UTC())
}

func (s *Service) UpdateRotationSchedule(ctx context.Context, sched *model.SSHKeyRotationSchedule) error {
	return s.keyRepo.UpdateRotationSchedule(ctx, sched)
}

// TrustKeyOnAllNodes distributes the public key identified by keyID to the
// authorized_keys of every other node. Errors per node are collected in the
// result rather than aborting the whole operation.
func (s *Service) TrustKeyOnAllNodes(ctx context.Context, keyID uuid.UUID) (*TrustResult, error) {
	key, err := s.keyRepo.GetByID(ctx, keyID)
	if err != nil {
		return nil, fmt.Errorf("get key: %w", err)
	}

	nodes, err := s.nodeRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list nodes: %w", err)
	}

	result := &TrustResult{}
	for i := range nodes {
		node := &nodes[i]
		if node.ID == key.NodeID {
			continue // skip the key's own node
		}
		if err := s.deployPublicKeyToNode(ctx, node, key.PublicKey); err != nil {
			slog.Warn("trust: deploy to node failed",
				slog.String("node", node.Name),
				slog.Any("error", err),
			)
			result.Failed = append(result.Failed, TrustFail{Node: node.Name, Error: err.Error()})
		} else {
			result.DistributedTo = append(result.DistributedTo, node.Name)
		}
	}

	return result, nil
}

// deployPublicKeyToNode SSHs into target using its own stored credentials and
// appends publicKey to its authorized_keys (idempotent).
func (s *Service) deployPublicKeyToNode(ctx context.Context, node *model.Node, publicKey string) error {
	privateKey, err := s.encryptor.Decrypt(node.SSHPrivateKey)
	if err != nil {
		return fmt.Errorf("decrypt node ssh key: %w", err)
	}

	sshClient, err := s.sshPool.Get(node.ID.String(), ssh.SSHConfig{
		Host:       node.Hostname,
		Port:       node.SSHPort,
		User:       node.SSHUser,
		PrivateKey: privateKey,
		HostKey:    node.SSHHostKey,
	})
	if err != nil {
		return fmt.Errorf("ssh connection: %w", err)
	}
	defer s.sshPool.Return(node.ID.String(), sshClient)

	pubKey := strings.TrimSpace(publicKey)
	cmd := fmt.Sprintf(
		`mkdir -p ~/.ssh && chmod 700 ~/.ssh && grep -qxF %q ~/.ssh/authorized_keys 2>/dev/null || echo %q >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys`,
		pubKey, pubKey,
	)
	result, err := sshClient.RunCommand(ctx, cmd)
	if err != nil {
		return fmt.Errorf("run command: %w", err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("command failed: %s", result.Stderr)
	}

	return nil
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

func generateRSAKeyPair() (publicKey, privateKey string, err error) {
	priv, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return "", "", fmt.Errorf("generate rsa key: %w", err)
	}

	sshPub, err := gossh.NewPublicKey(&priv.PublicKey)
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
