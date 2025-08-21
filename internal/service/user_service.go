package service

import (
	"context"

	"github.com/n1207n/cache-query-aggregator/db/sqlc"
	"github.com/n1207n/cache-query-aggregator/internal/repository"
)

// UserService defines the interface for user-related business logic.
type UserService interface {
	CreateUser(ctx context.Context, params sqlc.CreateUserParams) (sqlc.User, error)
	GetUserByID(ctx context.Context, id int64) (sqlc.User, error)
}

type userServiceImpl struct {
	userRepo repository.UserRepository
}

// NewUserService creates a new instance of UserService.
func NewUserService(userRepo repository.UserRepository) UserService {
	return &userServiceImpl{
		userRepo: userRepo,
	}
}

// CreateUser creates a new user.
func (s *userServiceImpl) CreateUser(ctx context.Context, params sqlc.CreateUserParams) (sqlc.User, error) {
	return s.userRepo.CreateUser(ctx, params)
}

// GetUserByID retrieves a user by their ID.
func (s *userServiceImpl) GetUserByID(ctx context.Context, id int64) (sqlc.User, error) {
	return s.userRepo.GetUserByID(ctx, id)
}
