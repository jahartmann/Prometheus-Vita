package user

import (
	"context"
	"errors"
	"fmt"

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
)

type Service struct {
	userRepo repository.UserRepository
}

func NewService(userRepo repository.UserRepository) *Service {
	return &Service{userRepo: userRepo}
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

	// If current password provided, verify it
	if req.CurrentPassword != "" {
		user, err := s.userRepo.GetByID(ctx, id)
		if err != nil {
			return err
		}
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
			return ErrWrongPassword
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	return s.userRepo.UpdatePassword(ctx, id, string(hash))
}
