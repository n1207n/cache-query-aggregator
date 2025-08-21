package mocks

import (
	"context"

	"github.com/n1207n/cache-query-aggregator/db/sqlc"
	"github.com/stretchr/testify/mock"
)

type UserRepository struct {
	mock.Mock
}

func (m *UserRepository) CreateUser(ctx context.Context, arg sqlc.CreateUserParams) (sqlc.User, error) {
	args := m.Called(ctx, arg)
	if args.Get(0) == nil {
		return sqlc.User{}, args.Error(1)
	}
	return args.Get(0).(sqlc.User), args.Error(1)
}

func (m *UserRepository) GetUserByID(ctx context.Context, id int64) (sqlc.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return sqlc.User{}, args.Error(1)
	}
	return args.Get(0).(sqlc.User), args.Error(1)
}

func (m *UserRepository) GetUserByEmail(ctx context.Context, email string) (sqlc.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return sqlc.User{}, args.Error(1)
	}
	return args.Get(0).(sqlc.User), args.Error(1)
}

func (m *UserRepository) ListUsers(ctx context.Context, arg sqlc.ListUsersParams) ([]sqlc.User, error) {
	args := m.Called(ctx, arg)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]sqlc.User), args.Error(1)
}

func (m *UserRepository) UpdateUser(ctx context.Context, arg sqlc.UpdateUserParams) (sqlc.User, error) {
	args := m.Called(ctx, arg)
	if args.Get(0) == nil {
		return sqlc.User{}, args.Error(1)
	}
	return args.Get(0).(sqlc.User), args.Error(1)
}

func (m *UserRepository) DeleteUser(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
