package service

import (
	"context"

	"github.com/n1207n/cache-query-aggregator/db/sqlc"
	"github.com/n1207n/cache-query-aggregator/internal/repository"
)

type PostService interface {
	CreatePost(ctx context.Context, params sqlc.CreatePostParams) (sqlc.Post, error)
	ListPostsByUser(ctx context.Context, params sqlc.ListPostsByUserParams) ([]sqlc.Post, error)
}

type postServiceImpl struct {
	postRepo repository.PostRepository
}

func NewPostService(postRepo repository.PostRepository) PostService {
	return &postServiceImpl{
		postRepo: postRepo,
	}
}

func (s *postServiceImpl) CreatePost(ctx context.Context, params sqlc.CreatePostParams) (sqlc.Post, error) {
	return s.postRepo.CreatePost(ctx, params)
}

func (s *postServiceImpl) ListPostsByUser(ctx context.Context, params sqlc.ListPostsByUserParams) ([]sqlc.Post, error) {
	return s.postRepo.ListPostsByUser(ctx, params)
}
