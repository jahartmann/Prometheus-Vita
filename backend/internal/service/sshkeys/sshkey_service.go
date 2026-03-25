package sshkeys

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"log/slog"
	"time"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/crypto"
	"github.com/antigravity/prometheus/internal/ssh"
	"github.com/google/uuid"
	gossh "golang.org/x/crypto/ssh"
)

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

	pubKey, privKey, err := generateEd25519KeyPair()
	if err != nil {
		return nil, fmt.Errorf("generate key pair: %w", err)
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
		KeyType:     "ed25519",
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

	// Add public key to authorized_keys
	cmd := fmt.Sprintf(
		`mkdir -p ~/.ssh && chmod 700 ~/.ssh && echo %q >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys`,
		key.PublicKey,
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
