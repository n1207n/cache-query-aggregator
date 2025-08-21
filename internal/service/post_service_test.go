//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/n1207n/cache-query-aggregator/db/sqlc"
	"github.com/n1207n/cache-query-aggregator/internal/repository/mocks"
	"github.com/stretchr/testify/assert"
)

func TestPostServiceImpl_CreatePost(t *testing.T) {
	mockRepo := new(mocks.PostRepository)
	postService := NewPostService(mockRepo)

	ctx := context.Background()
	params := sqlc.CreatePostParams{
		UserID:  1,
		Content: "Test content",
	}
	expectedPost := sqlc.Post{
		ID:        1,
		UserID:    params.UserID,
		Content:   params.Content,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockRepo.On("CreatePost", ctx, params).Return(expectedPost, nil)

	post, err := postService.CreatePost(ctx, params)

	assert.NoError(t, err)
	assert.Equal(t, expectedPost, post)
	mockRepo.AssertExpectations(t)
}

func TestPostServiceImpl_ListPostsByUser(t *testing.T) {
	mockRepo := new(mocks.PostRepository)
	postService := NewPostService(mockRepo)

	ctx := context.Background()
	params := sqlc.ListPostsByUserParams{
		UserID: 1,
		Limit:  10,
		Offset: 0,
	}
	expectedPosts := []sqlc.Post{
		{ID: 1, UserID: 1, Content: "Post 1"},
		{ID: 2, UserID: 1, Content: "Post 2"},
	}

	mockRepo.On("ListPostsByUser", ctx, params).Return(expectedPosts, nil)

	posts, err := postService.ListPostsByUser(ctx, params)

	assert.NoError(t, err)
	assert.Equal(t, expectedPosts, posts)
	assert.Len(t, posts, 2)
	mockRepo.AssertExpectations(t)
}
