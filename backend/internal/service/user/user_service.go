package user

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUsernameTaken    = errors.New("username already taken")
	ErrSelfDelete       = errors.New("cannot delete own account")
	ErrLastAdmin        = errors.New("cannot delete last admin")
	ErrWrongPassword    = errors.New("current password is incorrect")
	ErrPasswordRequired = errors.New("current password is required")
	ErrInvitationInvalid = errors.New("invitation is invalid")
	ErrInvitationExpired = errors.New("invitation is expired")
)

type Service struct {
	userRepo       repository.UserRepository
	tokenRepo      repository.TokenRepository
	apiTokenRepo   repository.APITokenRepository
	invitationRepo repository.UserInvitationRepository
	pwValidator    *PasswordValidator
}

func NewService(userRepo repository.UserRepository, pwValidator *PasswordValidator) *Service {
	return &Service{userRepo: userRepo, pwValidator: pwValidator}
}

func (s *Service) WithAccessRepositories(tokenRepo repository.TokenRepository, apiTokenRepo repository.APITokenRepository, invitationRepo repository.UserInvitationRepository) *Service {
	s.tokenRepo = tokenRepo
	s.apiTokenRepo = apiTokenRepo
	s.invitationRepo = invitationRepo
	return s
}

func (s *Service) List(ctx context.Context) ([]model.UserResponse, error) {
	users, err := s.userRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	responses := make([]model.UserResponse, 0, len(users))
	for _, u := range users {
		responses = append(responses, u.ToResponse())
	}
	return responses, nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*model.UserResponse, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := user.ToResponse()
	return &resp, nil
}

func (s *Service) Create(ctx context.Context, req model.CreateUserRequest) (*model.UserResponse, error) {
	// Validate unique username
	_, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err == nil {
		return nil, ErrUsernameTaken
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("check username: %w", err)
	}

	if !req.Role.IsValid() {
		return nil, fmt.Errorf("invalid role: %s", req.Role)
	}

	if s.pwValidator != nil {
		violations := s.pwValidator.Validate(ctx, req.Password, req.Username)
		if len(violations) > 0 {
			return nil, fmt.Errorf("password policy: %s", strings.Join(violations, "; "))
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &model.User{
		Username:      req.Username,
		Email:         req.Email,
		PasswordHash:  string(hash),
		Role:          req.Role,
		IsActive:      true,
		AutonomyLevel: model.AutonomyConfirm,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	resp := user.ToResponse()
	return &resp, nil
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, req model.UpdateUserRequest) (*model.UserResponse, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Username != nil && *req.Username != user.Username {
		existing, err := s.userRepo.GetByUsername(ctx, *req.Username)
		if err == nil && existing.ID != id {
			return nil, ErrUsernameTaken
		}
		if err != nil && !errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("check username: %w", err)
		}
		user.Username = *req.Username
	}
	if req.Email != nil {
		user.Email = *req.Email
	}
	if req.Role != nil {
		if !req.Role.IsValid() {
			return nil, fmt.Errorf("invalid role: %s", *req.Role)
		}
		user.Role = *req.Role
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}
	if req.AutonomyLevel != nil {
		level := *req.AutonomyLevel
		if level >= 0 && level <= 2 {
			user.AutonomyLevel = level
		}
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}
	if req.IsActive != nil && !*req.IsActive {
		if err := s.revokeUserAccess(ctx, id); err != nil {
			return nil, err
		}
	}

	resp := user.ToResponse()
	return &resp, nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID, currentUserID uuid.UUID) error {
	if id == currentUserID {
		return ErrSelfDelete
	}

	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if user.Role == model.RoleAdmin {
		count, err := s.userRepo.CountByRole(ctx, model.RoleAdmin)
		if err != nil {
			return fmt.Errorf("count admins: %w", err)
		}
		if count <= 1 {
			return ErrLastAdmin
		}
	}

	return s.userRepo.Delete(ctx, id)
}

func (s *Service) ChangePassword(ctx context.Context, id uuid.UUID, req model.ChangePasswordRequest, currentUserID uuid.UUID, currentRole model.UserRole) error {
	// Non-admins must provide current password and can only change their own
	if currentRole != model.RoleAdmin {
		if id != currentUserID {
			return fmt.Errorf("insufficient permissions")
		}
		if req.CurrentPassword == "" {
			return ErrPasswordRequired
		}
	}

	// Fetch user for current password verification and policy validation
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// If current password provided, verify it
	if req.CurrentPassword != "" {
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
			return ErrWrongPassword
		}
	}

	if s.pwValidator != nil {
		violations := s.pwValidator.Validate(ctx, req.NewPassword, user.Username)
		if len(violations) > 0 {
			return fmt.Errorf("password policy: %s", strings.Join(violations, "; "))
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	if err := s.userRepo.UpdatePassword(ctx, id, string(hash)); err != nil {
		return err
	}

	// Clear must_change_password flag
	if user != nil && user.MustChangePassword {
		user.MustChangePassword = false
		_ = s.userRepo.Update(ctx, user)
	}

	return nil
}

func (s *Service) CreateInvitation(ctx context.Context, req model.CreateUserInvitationRequest, createdBy uuid.UUID) (*model.CreateUserInvitationResponse, error) {
	if s.invitationRepo == nil {
		return nil, fmt.Errorf("invitation repository not configured")
	}
	if req.Username == "" {
		return nil, fmt.Errorf("username is required")
	}
	if !req.Role.IsValid() {
		return nil, fmt.Errorf("invalid role: %s", req.Role)
	}
	_, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err == nil {
		return nil, ErrUsernameTaken
	}
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("check username: %w", err)
	}

	token, err := generateInvitationToken()
	if err != nil {
		return nil, err
	}
	expiresIn := time.Duration(req.ExpiresInHours) * time.Hour
	if expiresIn <= 0 {
		expiresIn = 7 * 24 * time.Hour
	}
	if expiresIn > 30*24*time.Hour {
		expiresIn = 30 * 24 * time.Hour
	}

	invitation := &model.UserInvitation{
		Username:    req.Username,
		Email:       req.Email,
		Role:        req.Role,
		TokenHash:   repository.HashToken(token),
		TokenPrefix: token[:12],
		ExpiresAt:   time.Now().Add(expiresIn),
		CreatedBy:   createdBy,
	}
	if err := s.invitationRepo.Create(ctx, invitation); err != nil {
		return nil, err
	}
	return &model.CreateUserInvitationResponse{
		Invitation: invitation.ToResponse(),
		Token:      token,
	}, nil
}

func (s *Service) ListInvitations(ctx context.Context) ([]model.UserInvitationResponse, error) {
	if s.invitationRepo == nil {
		return nil, fmt.Errorf("invitation repository not configured")
	}
	invitations, err := s.invitationRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	responses := make([]model.UserInvitationResponse, 0, len(invitations))
	for _, invitation := range invitations {
		responses = append(responses, invitation.ToResponse())
	}
	return responses, nil
}

func (s *Service) DeleteInvitation(ctx context.Context, id uuid.UUID) error {
	if s.invitationRepo == nil {
		return fmt.Errorf("invitation repository not configured")
	}
	return s.invitationRepo.Delete(ctx, id)
}

func (s *Service) AcceptInvitation(ctx context.Context, req model.AcceptUserInvitationRequest) (*model.UserResponse, error) {
	if s.invitationRepo == nil {
		return nil, fmt.Errorf("invitation repository not configured")
	}
	if req.Token == "" || req.Password == "" {
		return nil, ErrInvitationInvalid
	}
	invitation, err := s.invitationRepo.GetByTokenHash(ctx, repository.HashToken(req.Token))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInvitationInvalid
		}
		return nil, err
	}
	if invitation.AcceptedAt != nil {
		return nil, ErrInvitationInvalid
	}
	if invitation.ExpiresAt.Before(time.Now()) {
		return nil, ErrInvitationExpired
	}

	resp, err := s.Create(ctx, model.CreateUserRequest{
		Username: invitation.Username,
		Email:    invitation.Email,
		Password: req.Password,
		Role:     invitation.Role,
	})
	if err != nil {
		return nil, err
	}
	if err := s.invitationRepo.MarkAccepted(ctx, invitation.ID, time.Now()); err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *Service) ListSessions(ctx context.Context, userID uuid.UUID) ([]model.UserSession, error) {
	if s.tokenRepo == nil {
		return nil, fmt.Errorf("token repository not configured")
	}
	return s.tokenRepo.ListByUser(ctx, userID)
}

func (s *Service) RevokeSession(ctx context.Context, userID, sessionID uuid.UUID) error {
	if s.tokenRepo == nil {
		return fmt.Errorf("token repository not configured")
	}
	return s.tokenRepo.RevokeByIDForUser(ctx, sessionID, userID)
}

func (s *Service) RevokeAllAccess(ctx context.Context, userID uuid.UUID) error {
	return s.revokeUserAccess(ctx, userID)
}

func (s *Service) ListAPITokens(ctx context.Context, userID uuid.UUID) ([]model.APIToken, error) {
	if s.apiTokenRepo == nil {
		return nil, fmt.Errorf("api token repository not configured")
	}
	return s.apiTokenRepo.ListByUser(ctx, userID)
}

func (s *Service) revokeUserAccess(ctx context.Context, userID uuid.UUID) error {
	if s.tokenRepo != nil {
		if err := s.tokenRepo.RevokeAllForUser(ctx, userID); err != nil {
			return fmt.Errorf("revoke user sessions: %w", err)
		}
	}
	if s.apiTokenRepo != nil {
		if err := s.apiTokenRepo.RevokeAllForUser(ctx, userID); err != nil {
			return fmt.Errorf("revoke user api tokens: %w", err)
		}
	}
	return nil
}

func generateInvitationToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate invitation token: %w", err)
	}
	return "pm_inv_" + hex.EncodeToString(b), nil
}
