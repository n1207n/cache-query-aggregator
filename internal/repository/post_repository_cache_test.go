package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/go-redis/redismock/v8"
	"github.com/n1207n/cache-query-aggregator/db/sqlc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockPostRepository mocks PostRepository interface
type mockPostRepository struct {
	mock.Mock
}

func (m *mockPostRepository) CreatePost(ctx context.Context, arg sqlc.CreatePostParams) (sqlc.Post, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(sqlc.Post), args.Error(1)
}

func (m *mockPostRepository) GetPost(ctx context.Context, id int64) (sqlc.Post, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(sqlc.Post), args.Error(1)
}

func (m *mockPostRepository) ListPostsByUser(ctx context.Context, arg sqlc.ListPostsByUserParams) ([]sqlc.Post, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]sqlc.Post), args.Error(1)
}

func TestGetPost_CacheHit(t *testing.T) {
	db, rdbMock := redismock.NewClientMock()
	mockRepo := new(mockPostRepository)
	repo := NewCachedPostRepository(mockRepo, db)

	post := sqlc.Post{ID: 1, UserID: 1, Content: "test content"}
	postJSON, _ := json.Marshal(post)
	postKey := fmt.Sprintf(postKeyGenericPattern, post.ID)

	rdbMock.ExpectGet(postKey).SetVal(string(postJSON))

	result, err := repo.GetPost(context.Background(), post.ID)

	require.NoError(t, err)
	assert.Equal(t, post, result)
	mockRepo.AssertNotCalled(t, "GetPost")
	require.NoError(t, rdbMock.ExpectationsWereMet())
}

func TestGetPost_CacheMiss(t *testing.T) {
	db, rdbMock := redismock.NewClientMock()
	mockRepo := new(mockPostRepository)
	repo := NewCachedPostRepository(mockRepo, db)

	post := sqlc.Post{ID: 1, UserID: 1, Content: "test content", CreatedAt: time.Now()}
	postKeyGeneric := fmt.Sprintf(postKeyGenericPattern, post.ID)
	userPostsKey := fmt.Sprintf(userPostsKeyPattern, post.UserID)

	rdbMock.ExpectGet(postKeyGeneric).SetErr(redis.Nil)
	mockRepo.On("GetPost", mock.Anything, post.ID).Return(post, nil)

	postJSON, _ := json.Marshal(post)
	rdbMock.ExpectSet(postKeyGeneric, postJSON, cacheTTL).SetVal("OK")
	rdbMock.ExpectZAdd(userPostsKey, &redis.Z{Score: float64(post.CreatedAt.Unix()), Member: post.ID}).SetVal(1)
	rdbMock.ExpectExpire(userPostsKey, cacheTTL).SetVal(true)

	result, err := repo.GetPost(context.Background(), post.ID)

	require.NoError(t, err)
	assert.Equal(t, post, result)
	mockRepo.AssertCalled(t, "GetPost", mock.Anything, post.ID)
	require.NoError(t, rdbMock.ExpectationsWereMet())
}

func TestListPostsByUser_FullCacheHit(t *testing.T) {
	db, rdbMock := redismock.NewClientMock()
	mockRepo := new(mockPostRepository)
	repo := NewCachedPostRepository(mockRepo, db)

	params := sqlc.ListPostsByUserParams{UserID: 1, Limit: 10, Offset: 0}
	userPostsKey := fmt.Sprintf(userPostsKeyPattern, params.UserID)

	postIDs := []string{"1", "2"}
	posts := []sqlc.Post{
		{ID: 1, UserID: 1, Content: "post 1"},
		{ID: 2, UserID: 1, Content: "post 2"},
	}
	postJSONs := make([]interface{}, len(posts))
	postKeys := make([]string, len(posts))
	for i, p := range posts {
		jsonBytes, _ := json.Marshal(p)
		postJSONs[i] = string(jsonBytes)
		postKeys[i] = fmt.Sprintf(postKeyGenericPattern, p.ID)
	}

	rdbMock.MatchExpectationsInOrder(false)
	rdbMock.ExpectZRevRange(userPostsKey, 0, 9).SetVal(postIDs)

	for i := range posts {
		rdbMock.ExpectGet(postKeys[i]).SetVal(postJSONs[i].(string))
	}

	result, err := repo.ListPostsByUser(context.Background(), params)

	require.NoError(t, err)

	slices.Reverse(result)
	assert.Equal(t, posts, result)
	mockRepo.AssertNotCalled(t, "ListPostsByUser")
	require.NoError(t, rdbMock.ExpectationsWereMet())
}

func TestListPostsByUser_PartialCacheHit(t *testing.T) {
	db, rdbMock := redismock.NewClientMock()
	mockRepo := new(mockPostRepository)
	repo := NewCachedPostRepository(mockRepo, db)

	params := sqlc.ListPostsByUserParams{UserID: 1, Limit: 2, Offset: 0}
	userPostsKey := fmt.Sprintf(userPostsKeyPattern, params.UserID)

	postIDs := []string{"1", "2"}
	post1 := sqlc.Post{ID: 1, UserID: 1, Content: "post 1", CreatedAt: time.Now()}
	post2 := sqlc.Post{ID: 2, UserID: 1, Content: "post 2", CreatedAt: time.Now().Add(-time.Minute)}
	post1JSON, _ := json.Marshal(post1)

	// MGet returns a value first time then nil
	rdbMock.MatchExpectationsInOrder(false)
	rdbMock.ExpectZRevRange(userPostsKey, 0, 1).SetVal(postIDs)
	rdbMock.ExpectGet(fmt.Sprintf(postKeyGenericPattern, 1)).SetVal(string(post1JSON))
	rdbMock.ExpectGet(fmt.Sprintf(postKeyGenericPattern, 2)).SetErr(redis.Nil)

	// For partial hit, return all data from DB
	dbPosts := []sqlc.Post{post1, post2}
	mockRepo.On("ListPostsByUser", mock.Anything, params).Return(dbPosts, nil)

	post2JSON, _ := json.Marshal(post2)

	// Post 1
	rdbMock.ExpectSet(fmt.Sprintf(postKeyGenericPattern, post1.ID), post1JSON, cacheTTL).SetVal("OK")
	// Post 2
	rdbMock.ExpectSet(fmt.Sprintf(postKeyGenericPattern, post2.ID), post2JSON, cacheTTL).SetVal("OK")
	members := []*redis.Z{
		{Score: float64(post1.CreatedAt.Unix()), Member: post1.ID},
		{Score: float64(post2.CreatedAt.Unix()), Member: post2.ID},
	}
	rdbMock.ExpectZAdd(userPostsKey, members...).SetVal(2)
	rdbMock.ExpectExpire(userPostsKey, cacheTTL).SetVal(true)

	_, err := repo.ListPostsByUser(context.Background(), params)
	require.NoError(t, err)
	mockRepo.AssertCalled(t, "ListPostsByUser", mock.Anything, params)
	require.NoError(t, rdbMock.ExpectationsWereMet())
}

func TestListPostsByUser_CacheMiss(t *testing.T) {
	db, rdbMock := redismock.NewClientMock()
	mockRepo := new(mockPostRepository)
	repo := NewCachedPostRepository(mockRepo, db)

	params := sqlc.ListPostsByUserParams{UserID: 1, Limit: 10, Offset: 0}
	userPostsKey := fmt.Sprintf(userPostsKeyPattern, params.UserID)

	rdbMock.ExpectZRevRange(userPostsKey, 0, 9).SetErr(redis.Nil)

	dbPost := sqlc.Post{ID: 1, UserID: 1, Content: "db post", CreatedAt: time.Now()}
	dbPosts := []sqlc.Post{dbPost}
	mockRepo.On("ListPostsByUser", mock.Anything, params).Return(dbPosts, nil)

	postJSON, _ := json.Marshal(dbPost)
	rdbMock.ExpectSet(fmt.Sprintf(postKeyGenericPattern, dbPost.ID), postJSON, cacheTTL).SetVal("OK")
	rdbMock.ExpectZAdd(userPostsKey, &redis.Z{Score: float64(dbPost.CreatedAt.Unix()), Member: dbPost.ID}).SetVal(1)
	rdbMock.ExpectExpire(userPostsKey, cacheTTL).SetVal(true)

	_, err := repo.ListPostsByUser(context.Background(), params)
	require.NoError(t, err)
	mockRepo.AssertCalled(t, "ListPostsByUser", mock.Anything, params)
	require.NoError(t, rdbMock.ExpectationsWereMet())
}

func TestCreatePost(t *testing.T) {
	db, rdbMock := redismock.NewClientMock()
	mockRepo := new(mockPostRepository)
	repo := NewCachedPostRepository(mockRepo, db)

	createParams := sqlc.CreatePostParams{UserID: 1, Content: "new post"}
	createdPost := sqlc.Post{ID: 100, UserID: 1, Content: "new post", CreatedAt: time.Now()}

	mockRepo.On("CreatePost", mock.Anything, createParams).Return(createdPost, nil)

	postKeyGeneric := fmt.Sprintf(postKeyGenericPattern, createdPost.ID)
	userPostsKey := fmt.Sprintf(userPostsKeyPattern, createdPost.UserID)
	postJSON, _ := json.Marshal(createdPost)

	rdbMock.ExpectSet(postKeyGeneric, postJSON, cacheTTL).SetVal("OK")
	rdbMock.ExpectZAdd(userPostsKey, &redis.Z{Score: float64(createdPost.CreatedAt.Unix()), Member: createdPost.ID}).SetVal(1)
	rdbMock.ExpectExpire(userPostsKey, cacheTTL).SetVal(true)

	result, err := repo.CreatePost(context.Background(), createParams)
	require.NoError(t, err)
	assert.Equal(t, createdPost, result)
	mockRepo.AssertCalled(t, "CreatePost", mock.Anything, createParams)
	require.NoError(t, rdbMock.ExpectationsWereMet())
}
