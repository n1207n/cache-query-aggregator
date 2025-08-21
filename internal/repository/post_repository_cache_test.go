package repository

import (
	"context"
	"encoding/json"
	"fmt"
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
	postKey := fmt.Sprintf(postKeyPattern, post.ID)

	rdbMock.ExpectGet(postKey).SetVal(string(postJSON))

	result, err := repo.GetPost(context.Background(), post.ID)

	require.NoError(t, err)
	assert.Equal(t, post, result)
	mockRepo.AssertNotCalled(t, "GetPost") // DB는 호출되지 않아야 함
	require.NoError(t, rdbMock.ExpectationsWereMet())
}

func TestGetPost_CacheMiss(t *testing.T) {
	db, rdbMock := redismock.NewClientMock()
	mockRepo := new(mockPostRepository)
	repo := NewCachedPostRepository(mockRepo, db)

	post := sqlc.Post{ID: 1, UserID: 1, Content: "test content", CreatedAt: time.Now()}
	postKey := fmt.Sprintf(postKeyPattern, post.ID)
	userPostsKey := fmt.Sprintf(userPostsKeyPattern, post.UserID)

	rdbMock.ExpectGet(postKey).SetErr(redis.Nil)
	mockRepo.On("GetPost", mock.Anything, post.ID).Return(post, nil)

	postJSON, _ := json.Marshal(post)
	rdbMock.ExpectSet(postKey, postJSON, cacheTTL).SetVal("OK")
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
		postKeys[i] = fmt.Sprintf(postKeyPattern, p.ID)
	}

	rdbMock.ExpectZRevRange(userPostsKey, 0, 9).SetVal(postIDs)
	rdbMock.ExpectMGet(postKeys...).SetVal(postJSONs)

	result, err := repo.ListPostsByUser(context.Background(), params)

	require.NoError(t, err)
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
	rdbMock.ExpectZRevRange(userPostsKey, 0, 1).SetVal(postIDs)
	rdbMock.ExpectMGet(fmt.Sprintf(postKeyPattern, 1), fmt.Sprintf(postKeyPattern, 2)).SetVal([]interface{}{string(post1JSON), nil})

	// For partial hit, return all data from DB
	dbPosts := []sqlc.Post{post1, post2}
	mockRepo.On("ListPostsByUser", mock.Anything, params).Return(dbPosts, nil)

	// DB에서 가져온 데이터를 캐시에 채우는 파이프라인 명령 모의
	post2JSON, _ := json.Marshal(post2)
	rdbMock.ExpectSet(fmt.Sprintf(postKeyPattern, post1.ID), post1JSON, cacheTTL).SetVal("OK")
	rdbMock.ExpectSet(fmt.Sprintf(postKeyPattern, post2.ID), post2JSON, cacheTTL).SetVal("OK")
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

	// ZRevRange가 redis.Nil 또는 빈 슬라이스를 반환하여 캐시 미스 발생
	rdbMock.ExpectZRevRange(userPostsKey, 0, 9).SetErr(redis.Nil)

	dbPost := sqlc.Post{ID: 1, UserID: 1, Content: "db post", CreatedAt: time.Now()}
	dbPosts := []sqlc.Post{dbPost}
	mockRepo.On("ListPostsByUser", mock.Anything, params).Return(dbPosts, nil)

	// 파이프라인으로 캐시 채우는 로직 모의
	postJSON, _ := json.Marshal(dbPost)
	rdbMock.ExpectSet(fmt.Sprintf(postKeyPattern, dbPost.ID), postJSON, cacheTTL).SetVal("OK")
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

	postKey := fmt.Sprintf(postKeyPattern, createdPost.ID)
	userPostsKey := fmt.Sprintf(userPostsKeyPattern, createdPost.UserID)
	postJSON, _ := json.Marshal(createdPost)

	rdbMock.ExpectSet(postKey, postJSON, cacheTTL).SetVal("OK")
	rdbMock.ExpectZAdd(userPostsKey, &redis.Z{Score: float64(createdPost.CreatedAt.Unix()), Member: createdPost.ID}).SetVal(1)
	rdbMock.ExpectExpire(userPostsKey, cacheTTL).SetVal(true)

	result, err := repo.CreatePost(context.Background(), createParams)
	require.NoError(t, err)
	assert.Equal(t, createdPost, result)
	mockRepo.AssertCalled(t, "CreatePost", mock.Anything, createParams)
	require.NoError(t, rdbMock.ExpectationsWereMet())
}
