package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/n1207n/cache-query-aggregator/db/sqlc"
	"github.com/n1207n/cache-query-aggregator/internal/metrics"
)

const (
	userPostsKeyPattern = "{user:%d}:posts"
	postKeyPattern      = "post:%d"
	cacheTTL            = 1 * time.Hour
)

// CachedPostRepository is a cache decorator for PostRepository
type CachedPostRepository struct {
	nextRepo PostRepository
	rdb      redis.Cmdable
}

// NewCachedPostRepository creates a new instance of CachedPostRepository
func NewCachedPostRepository(next PostRepository, rdb redis.Cmdable) PostRepository {
	return &CachedPostRepository{
		nextRepo: next,
		rdb:      rdb,
	}
}

// CreatePost creates a Post table record and pushes it into cache
func (r *CachedPostRepository) CreatePost(ctx context.Context, arg sqlc.CreatePostParams) (sqlc.Post, error) {
	post, err := r.nextRepo.CreatePost(ctx, arg)
	if err != nil {
		return sqlc.Post{}, err
	}

	if err := r.cachePost(ctx, &post); err != nil {
		log.Printf("failed to cache created post %d: %v", post.ID, err)
	}

	return post, nil
}

// GetPost reads Post from cache first then DB
func (r *CachedPostRepository) GetPost(ctx context.Context, id int64) (sqlc.Post, error) {
	postKey := fmt.Sprintf(postKeyPattern, id)
	val, err := r.rdb.Get(ctx, postKey).Result()

	if err == nil {
		// Cache hit
		var post sqlc.Post
		if err := json.Unmarshal([]byte(val), &post); err == nil {
			log.Printf("cache hit for post %d", id)
			metrics.PostCacheHits.Inc()
			return post, nil
		}
		log.Printf("failed to unmarshal cached post %d: %v", id, err)
	}

	if err != redis.Nil {
		log.Printf("redis error on getting post %d: %v", id, err)
	}

	// Cache miss
	log.Printf("cache miss for post %d, fetching from db", id)
	metrics.PostCacheMisses.Inc()
	metrics.PostDBQueries.Inc()
	post, err := r.nextRepo.GetPost(ctx, id)
	if err != nil {
		return sqlc.Post{}, err
	}

	if err := r.cachePost(ctx, &post); err != nil {
		log.Printf("failed to cache post %d after db fetch: %v", post.ID, err)
	}

	return post, nil
}

// ListPostsByUser queries a list of Posts from cache first then DB
func (r *CachedPostRepository) ListPostsByUser(ctx context.Context, arg sqlc.ListPostsByUserParams) ([]sqlc.Post, error) {
	userPostsKey := fmt.Sprintf(userPostsKeyPattern, arg.UserID)
	start, stop := int64(arg.Offset), int64(arg.Offset+arg.Limit-1)
	postIDStrs, err := r.rdb.ZRevRange(ctx, userPostsKey, start, stop).Result()

	if err == nil && len(postIDStrs) > 0 {
		posts, missedIDs := r.getPostsFromCache(ctx, postIDStrs)
		if len(missedIDs) == 0 {
			log.Printf("full cache hit for user %d posts list (offset: %d, limit: %d)", arg.UserID, arg.Offset, arg.Limit)
			metrics.PostCacheHits.Inc()
			return posts, nil
		}
		// Partial cache hit, a.k.a shard join
		log.Printf("partial cache hit for user %d. Missed %d posts. Fetching full list from DB.", arg.UserID, len(missedIDs))
		metrics.PostCacheShardJoins.Inc()
	}

	if err != nil && err != redis.Nil {
		log.Printf("redis error on getting post list for user %d: %v", arg.UserID, err)
	}

	// Full cache miss
	if err == redis.Nil || len(postIDStrs) == 0 {
		log.Printf("full cache miss for user %d posts list, fetching from db", arg.UserID)
		metrics.PostCacheMisses.Inc()
	}

	metrics.PostDBQueries.Inc()
	posts, err := r.nextRepo.ListPostsByUser(ctx, arg)
	if err != nil {
		return nil, err
	}

	if len(posts) > 0 {
		if err := r.cachePostList(ctx, arg.UserID, posts); err != nil {
			log.Printf("failed to cache post list for user %d: %v", arg.UserID, err)
		}
	}

	return posts, nil
}

func (r *CachedPostRepository) getPostsFromCache(ctx context.Context, postIDStrs []string) ([]sqlc.Post, []int64) {
	if len(postIDStrs) == 0 {
		return []sqlc.Post{}, nil
	}

	postKeys := make([]string, len(postIDStrs))
	postIDMap := make(map[string]int64, len(postIDStrs))
	for i, idStr := range postIDStrs {
		id, _ := strconv.ParseInt(idStr, 10, 64)
		postKeys[i] = fmt.Sprintf(postKeyPattern, id)
		postIDMap[postKeys[i]] = id
	}

	vals, err := r.rdb.MGet(ctx, postKeys...).Result()
	if err != nil {
		log.Printf("MGet failed for keys %v: %v", postKeys, err)
		var allIDs []int64
		for _, id := range postIDMap {
			allIDs = append(allIDs, id)
		}
		return nil, allIDs
	}

	posts := make([]sqlc.Post, 0, len(vals))
	var missedIDs []int64

	for i, val := range vals {
		if val == nil {
			missedIDs = append(missedIDs, postIDMap[postKeys[i]])
			continue
		}
		var post sqlc.Post
		if err := json.Unmarshal([]byte(val.(string)), &post); err != nil {
			log.Printf("failed to unmarshal cached post for key %s: %v", postKeys[i], err)
			missedIDs = append(missedIDs, postIDMap[postKeys[i]])
			continue
		}
		posts = append(posts, post)
	}

	return posts, missedIDs
}

// cachePost caches a single Post object and add it into user's post list as sorted set
func (r *CachedPostRepository) cachePost(ctx context.Context, post *sqlc.Post) error {
	postKey := fmt.Sprintf(postKeyPattern, post.ID)
	postJSON, err := json.Marshal(post)
	if err != nil {
		return fmt.Errorf("failed to marshal post %d: %w", post.ID, err)
	}

	if err := r.rdb.Set(ctx, postKey, postJSON, cacheTTL).Err(); err != nil {
		return fmt.Errorf("failed to SET post %d: %w", post.ID, err)
	}

	userPostsKey := fmt.Sprintf(userPostsKeyPattern, post.UserID)
	if err := r.rdb.ZAdd(ctx, userPostsKey, &redis.Z{
		Score:  float64(post.CreatedAt.Unix()),
		Member: post.ID,
	}).Err(); err != nil {
		return fmt.Errorf("failed to ZADD post %d to user %d list: %w", post.ID, post.UserID, err)
	}
	return r.rdb.Expire(ctx, userPostsKey, cacheTTL).Err()
}

// cachePostList caches multiple Posts and their ids
func (r *CachedPostRepository) cachePostList(ctx context.Context, userID int64, posts []sqlc.Post) error {
	if len(posts) == 0 {
		return nil
	}

	userPostsKey := fmt.Sprintf(userPostsKeyPattern, userID)
	pipe := r.rdb.Pipeline()

	redisZMembers := make([]*redis.Z, len(posts))
	for i, p := range posts {
		postKey := fmt.Sprintf(postKeyPattern, p.ID)
		postJSON, err := json.Marshal(p)
		if err != nil {
			log.Printf("failed to marshal post %d for batch cache: %v", p.ID, err)
			continue
		}
		pipe.Set(ctx, postKey, postJSON, cacheTTL)
		redisZMembers[i] = &redis.Z{Score: float64(p.CreatedAt.Unix()), Member: p.ID}
	}

	if len(redisZMembers) > 0 {
		pipe.ZAdd(ctx, userPostsKey, redisZMembers...)
		pipe.Expire(ctx, userPostsKey, cacheTTL)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("pipeline execution failed for user %d: %w", userID, err)
	}

	return nil
}
