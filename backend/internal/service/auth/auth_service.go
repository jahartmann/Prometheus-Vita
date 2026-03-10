package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserInactive       = errors.New("user account is inactive")
	ErrTokenRevoked       = errors.New("token has been revoked")
	ErrTokenExpired       = errors.New("token has expired")
	ErrAccountLocked      = errors.New("account is temporarily locked")
)

const (
	lockoutTTL      = 15 * time.Minute
	maxLoginAttempts = 5
)

type Service struct {
	userRepo  repository.UserRepository
	tokenRepo repository.TokenRepository
	jwt       *JWTService
	redis     *redis.Client
}

func NewService(userRepo repository.UserRepository, tokenRepo repository.TokenRepository, jwtSvc *JWTService, redisClient *redis.Client) *Service {
	return &Service{
		userRepo:  userRepo,
		tokenRepo: tokenRepo,
		jwt:       jwtSvc,
		redis:     redisClient,
	}
}

func (s *Service) Login(ctx context.Context, req model.LoginRequest) (*model.LoginResponse, error) {
	// Check if account is locked
	lockoutKey := fmt.Sprintf("lockout:%s", req.Username)
	exists, err := s.redis.Exists(ctx, lockoutKey).Result()
	if err != nil {
		slog.Warn("failed to check lockout status", slog.Any("error", err))
	} else if exists > 0 {
		return nil, ErrAccountLocked
	}

	user, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	if !user.IsActive {
		return nil, ErrUserInactive
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		// Increment failed login attempts
		attemptsKey := fmt.Sprintf("login_attempts:%s", req.Username)
		attempts, incrErr := s.redis.Incr(ctx, attemptsKey).Result()
		if incrErr != nil {
			slog.Warn("failed to increment login attempts", slog.Any("error", incrErr))
		} else {
			if attempts == 1 {
				s.redis.Expire(ctx, attemptsKey, lockoutTTL)
			}
			if attempts >= maxLoginAttempts {
				s.redis.Set(ctx, lockoutKey, "1", lockoutTTL)
			}
		}
		return nil, ErrInvalidCredentials
	}

	// Successful login: clear lockout and attempts
	attemptsKey := fmt.Sprintf("login_attempts:%s", req.Username)
	s.redis.Del(ctx, lockoutKey, attemptsKey)

	tokenPair, err := s.jwt.GenerateTokenPair(user)
	if err != nil {
		return nil, fmt.Errorf("generate tokens: %w", err)
	}

	tokenHash := repository.HashToken(tokenPair.RefreshToken)
	if err := s.tokenRepo.Create(ctx, user.ID, tokenHash, tokenPair.ExpiresAt); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	if err := s.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		slog.Warn("failed to update last login", slog.Any("error", err))
	}

	return &model.LoginResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		User:         user.ToInfo(),
	}, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (*model.RefreshResponse, error) {
	tokenHash := repository.HashToken(refreshToken)
	storedToken, err := s.tokenRepo.GetByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("get refresh token: %w", err)
	}

	if storedToken.Revoked {
		return nil, ErrTokenRevoked
	}

	if storedToken.ExpiresAt.Before(time.Now()) {
		return nil, ErrTokenExpired
	}

	// Revoke old token
	if err := s.tokenRepo.RevokeByHash(ctx, tokenHash); err != nil {
		return nil, fmt.Errorf("revoke old token: %w", err)
	}

	user, err := s.userRepo.GetByID(ctx, storedToken.UserID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	if !user.IsActive {
		return nil, ErrUserInactive
	}

	tokenPair, err := s.jwt.GenerateTokenPair(user)
	if err != nil {
		return nil, fmt.Errorf("generate new tokens: %w", err)
	}

	newTokenHash := repository.HashToken(tokenPair.RefreshToken)
	if err := s.tokenRepo.Create(ctx, user.ID, newTokenHash, tokenPair.ExpiresAt); err != nil {
		return nil, fmt.Errorf("store new refresh token: %w", err)
	}

	return &model.RefreshResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
	}, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := repository.HashToken(refreshToken)
	return s.tokenRepo.RevokeByHash(ctx, tokenHash)
}

func (s *Service) SeedAdmin(ctx context.Context, username, password string) error {
	count, err := s.userRepo.Count(ctx)
	if err != nil {
		return fmt.Errorf("count users: %w", err)
	}

	if count > 0 {
		slog.Info("users already exist, skipping admin seed")
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	user := &model.User{
		Username:           username,
		PasswordHash:       string(hash),
		Role:               model.RoleAdmin,
		IsActive:           true,
		MustChangePassword: password == "changeme",
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return fmt.Errorf("create admin user: %w", err)
	}

	slog.Info("admin user seeded", slog.String("username", username))
	return nil
}
