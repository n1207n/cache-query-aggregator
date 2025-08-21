package mocks

import (
	"context"

	"github.com/n1207n/cache-query-aggregator/db/sqlc"
	"github.com/stretchr/testify/mock"
)

type UserService struct {
	mock.Mock
}

func (m *UserService) CreateUser(ctx context.Context, params sqlc.CreateUserParams) (sqlc.User, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return sqlc.User{}, args.Error(1)
	}
	return args.Get(0).(sqlc.User), args.Error(1)
}

func (m *UserService) GetUserByID(ctx context.Context, id int64) (sqlc.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return sqlc.User{}, args.Error(1)
	}
	return args.Get(0).(sqlc.User), args.Error(1)
}
