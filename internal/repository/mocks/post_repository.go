package mocks

import (
	"context"

	"github.com/n1207n/cache-query-aggregator/db/sqlc"
	"github.com/stretchr/testify/mock"
)

type PostRepository struct {
	mock.Mock
}

func (m *PostRepository) CreatePost(ctx context.Context, arg sqlc.CreatePostParams) (sqlc.Post, error) {
	args := m.Called(ctx, arg)
	if args.Get(0) == nil {
		return sqlc.Post{}, args.Error(1)
	}
	return args.Get(0).(sqlc.Post), args.Error(1)
}

func (m *PostRepository) GetPost(ctx context.Context, id int64) (sqlc.Post, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return sqlc.Post{}, args.Error(1)
	}
	return args.Get(0).(sqlc.Post), args.Error(1)
}

func (m *PostRepository) ListPostsByUser(ctx context.Context, arg sqlc.ListPostsByUserParams) ([]sqlc.Post, error) {
	args := m.Called(ctx, arg)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]sqlc.Post), args.Error(1)
}
