package repository

import (
	"context"

	"github.com/yourusername/yourprojectname/db/sqlc"
)

type PostRepository interface {
	CreatePost(ctx context.Context, arg sqlc.CreatePostParams) (sqlc.Post, error)
	GetPost(ctx context.Context, id int64) (sqlc.Post, error)
	ListPostsByUser(ctx context.Context, arg sqlc.ListPostsByUserParams) ([]sqlc.Post, error)
}

type DBPostRepository struct {
	q sqlc.Querier
}

func NewDBPostRepository(querier sqlc.Querier) PostRepository {
	return &DBPostRepository{q: querier}
}

func (r *DBPostRepository) CreatePost(ctx context.Context, arg sqlc.CreatePostParams) (sqlc.Post, error) {
	return r.q.CreatePost(ctx, arg)
}

func (r *DBPostRepository) GetPost(ctx context.Context, id int64) (sqlc.Post, error) {
	return r.q.GetPost(ctx, id)
}

func (r *DBPostRepository) ListPostsByUser(ctx context.Context, arg sqlc.ListPostsByUserParams) ([]sqlc.Post, error) {
	return r.q.ListPostsByUser(ctx, arg)
}
