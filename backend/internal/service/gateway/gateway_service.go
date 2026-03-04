package gateway

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/google/uuid"
)

type Service struct {
	tokenRepo repository.APITokenRepository
	userRepo  repository.UserRepository
}

func NewService(tokenRepo repository.APITokenRepository, userRepo repository.UserRepository) *Service {
	return &Service{tokenRepo: tokenRepo, userRepo: userRepo}
}

func (s *Service) CreateToken(ctx context.Context, userID uuid.UUID, req model.CreateAPITokenRequest) (*model.CreateAPITokenResponse, error) {
	// Verify user exists
	if _, err := s.userRepo.GetByID(ctx, userID); err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	// Generate random token
	rawToken, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	tokenHash := hashToken(rawToken)
	prefix := rawToken[:8]

	permsJSON, _ := json.Marshal(req.Permissions)

	var expiresAt *time.Time
	if req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			return nil, fmt.Errorf("parse expires_at: %w", err)
		}
		expiresAt = &t
	}

	token := &model.APIToken{
		UserID:      userID,
		Name:        req.Name,
		TokenHash:   tokenHash,
		TokenPrefix: prefix,
		Permissions: permsJSON,
		IsActive:    true,
		ExpiresAt:   expiresAt,
	}

	if err := s.tokenRepo.Create(ctx, token); err != nil {
		return nil, fmt.Errorf("create token: %w", err)
	}

	return &model.CreateAPITokenResponse{
		Token:   rawToken,
		TokenID: token.ID,
		Name:    token.Name,
		Prefix:  prefix,
	}, nil
}

func (s *Service) ValidateToken(ctx context.Context, rawToken string) (*model.APIToken, error) {
	tokenHash := hashToken(rawToken)
	token, err := s.tokenRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if !token.IsActive {
		return nil, fmt.Errorf("token is revoked")
	}

	if token.ExpiresAt != nil && token.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("token has expired")
	}

	// Update last used (fire and forget)
	go func() {
		_ = s.tokenRepo.UpdateLastUsed(context.Background(), token.ID)
	}()

	return token, nil
}

func (s *Service) ListTokens(ctx context.Context, userID uuid.UUID) ([]model.APIToken, error) {
	return s.tokenRepo.ListByUser(ctx, userID)
}

func (s *Service) RevokeToken(ctx context.Context, tokenID uuid.UUID) error {
	token, err := s.tokenRepo.GetByID(ctx, tokenID)
	if err != nil {
		return err
	}
	token.IsActive = false
	return s.tokenRepo.Update(ctx, token)
}

func (s *Service) DeleteToken(ctx context.Context, tokenID uuid.UUID) error {
	return s.tokenRepo.Delete(ctx, tokenID)
}

func (s *Service) GetUserForToken(ctx context.Context, token *model.APIToken) (*model.User, error) {
	return s.userRepo.GetByID(ctx, token.UserID)
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "pm_" + hex.EncodeToString(b), nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
