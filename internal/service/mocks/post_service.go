package mocks

import (
	"context"

	"github.com/n1207n/cache-query-aggregator/db/sqlc"
	"github.com/stretchr/testify/mock"
)

type PostService struct {
	mock.Mock
}

func (m *PostService) CreatePost(ctx context.Context, params sqlc.CreatePostParams) (sqlc.Post, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return sqlc.Post{}, args.Error(1)
	}
	return args.Get(0).(sqlc.Post), args.Error(1)
}

func (m *PostService) ListPostsByUser(ctx context.Context, params sqlc.ListPostsByUserParams) ([]sqlc.Post, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]sqlc.Post), args.Error(1)
}
