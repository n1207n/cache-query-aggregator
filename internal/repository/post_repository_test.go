//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/n1207n/cache-query-aggregator/db/sqlc"
	"github.com/n1207n/cache-query-aggregator/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper to create a user for tests that need a valid user_id
func createTestUser(t *testing.T, ctx context.Context) sqlc.User {
	userRepo := NewDBUserRepository(testQueries)
	hashedPassword, err := util.HashPassword("password123")
	require.NoError(t, err)

	user, err := userRepo.CreateUser(ctx, sqlc.CreateUserParams{
		FirstName:      "Test",
		LastName:       "User",
		Email:          "test." + util.RandomString(6) + "@example.com",
		HashedPassword: hashedPassword,
	})
	require.NoError(t, err)
	require.NotEmpty(t, user)
	return user
}

func TestDBPostRepository_CreatePost(t *testing.T) {
	ctx := context.Background()
	user := createTestUser(t, ctx)
	postRepo := NewDBPostRepository(testQueries)

	params := sqlc.CreatePostParams{
		UserID:  user.ID,
		Content: "This is a test post.",
	}
	post, err := postRepo.CreatePost(ctx, params)

	assert.NoError(t, err)
	require.NotEmpty(t, post)
	assert.Equal(t, params.UserID, post.UserID)
	assert.Equal(t, params.Content, post.Content)
	assert.NotZero(t, post.ID)
	assert.NotZero(t, post.CreatedAt)
	assert.NotZero(t, post.UpdatedAt)

	// Test GetPost as well to verify creation
	fetchedPost, err := postRepo.GetPost(ctx, post.ID)
	assert.NoError(t, err)
	require.NotEmpty(t, fetchedPost)
	assert.Equal(t, post.ID, fetchedPost.ID)
	assert.Equal(t, post.Content, fetchedPost.Content)
}

func TestDBPostRepository_ListPostsByUser(t *testing.T) {
	ctx := context.Background()
	user := createTestUser(t, ctx)
	postRepo := NewDBPostRepository(testQueries)

	// Create some posts for the user
	for i := 0; i < 5; i++ {
		content := fmt.Sprintf("Post %d", i)
		_, err := postRepo.CreatePost(ctx, sqlc.CreatePostParams{
			UserID:  user.ID,
			Content: content,
		})
		require.NoError(t, err)
	}

	// Create a post for another user to ensure it's not fetched
	otherUser := createTestUser(t, ctx)
	_, err := postRepo.CreatePost(ctx, sqlc.CreatePostParams{
		UserID:  otherUser.ID,
		Content: "Another user's post",
	})
	require.NoError(t, err)

	params := sqlc.ListPostsByUserParams{
		UserID: user.ID,
		Limit:  10,
		Offset: 0,
	}
	posts, err := postRepo.ListPostsByUser(ctx, params)

	assert.NoError(t, err)
	assert.Len(t, posts, 5)

	// Test with limit and offset
	params.Limit = 2
	params.Offset = 1
	paginatedPosts, err := postRepo.ListPostsByUser(ctx, params)
	assert.NoError(t, err)
	assert.Len(t, paginatedPosts, 2)
}
