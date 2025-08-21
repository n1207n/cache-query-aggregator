package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/n1207n/cache-query-aggregator/db/sqlc"
	"github.com/n1207n/cache-query-aggregator/internal/metrics"
	crc16_redis "github.com/sigurn/crc16"
)

const (
	userPostsKeyPattern   = "{user:%d}:posts"
	postKeyGenericPattern = "post:%d"
	cacheTTL              = 1 * time.Hour
)

var crc16Table = crc16_redis.MakeTable(crc16_redis.CRC16_XMODEM)

// CachedPostRepository is a cache decorator for PostRepository
type CachedPostRepository struct {
	nextRepo      PostRepository
	rdb           redis.Cmdable
	clusterClient *redis.ClusterClient
	slotMap       map[uint16]string // slot -> node address
	slotMapMux    sync.RWMutex
}

// NewCachedPostRepository creates a new instance of CachedPostRepository
func NewCachedPostRepository(next PostRepository, rdb redis.Cmdable) PostRepository {
	repo := &CachedPostRepository{
		nextRepo: next,
		rdb:      rdb,
	}

	if clusterClient, ok := rdb.(*redis.ClusterClient); ok {
		repo.clusterClient = clusterClient
		if err := repo.refreshSlotCache(context.Background()); err != nil {
			log.Printf("failed to initialize redis cluster slot cache: %v", err)
		} else {
			log.Println("Redis cluster slot cache initialized successfully.")
		}
	}

	return repo
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
	postKey := fmt.Sprintf(postKeyGenericPattern, id)
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
		posts, missedIDs := r.getPostsFromCache(ctx, arg.UserID, postIDStrs)
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

func (r *CachedPostRepository) getPostsFromCache(ctx context.Context, userID int64, postIDStrs []string) ([]sqlc.Post, []int64) {
	if len(postIDStrs) == 0 {
		return []sqlc.Post{}, nil
	}

	type result struct {
		post sqlc.Post
		err  error
		id   int64
	}

	var wg sync.WaitGroup
	ch := make(chan result, len(postIDStrs))

	for _, idStr := range postIDStrs {
		wg.Add(1)
		id, _ := strconv.ParseInt(idStr, 10, 64)

		go func(postID int64) {
			defer wg.Done()
			postKey := fmt.Sprintf(postKeyGenericPattern, postID)

			if r.clusterClient != nil {
				r.slotMapMux.RLock()
				// CRC16 Checksum as per Redis spec for cluster key hashing.
				slot := crc16_redis.Checksum([]byte(postKey), crc16Table) & 0x3FFF
				if nodeAddr, ok := r.slotMap[slot]; ok {
					metrics.RedisNodeReadsByUser.WithLabelValues(nodeAddr, strconv.FormatInt(userID, 10)).Inc()
				}
				r.slotMapMux.RUnlock()
			}

			val, err := r.rdb.Get(ctx, postKey).Result()
			if err != nil {
				ch <- result{id: postID, err: err}
				return
			}

			var post sqlc.Post
			if err = json.Unmarshal([]byte(val), &post); err != nil {
				ch <- result{id: postID, err: err}
				return
			}
			ch <- result{id: postID, post: post, err: nil}
		}(id)
	}

	wg.Wait()
	close(ch)

	posts := make([]sqlc.Post, 0, len(postIDStrs))
	missedIDs := make([]int64, 0)
	for res := range ch {
		if res.err != nil {
			missedIDs = append(missedIDs, res.id)
			if res.err != redis.Nil {
				log.Printf("failed to get or unmarshal post %d from cache: %v", res.id, res.err)
			}
			continue
		}
		posts = append(posts, res.post)
	}

	return posts, missedIDs
}

func (r *CachedPostRepository) refreshSlotCache(ctx context.Context) error {
	r.slotMapMux.Lock()
	defer r.slotMapMux.Unlock()

	slots, err := r.clusterClient.ClusterSlots(ctx).Result()
	if err != nil {
		return fmt.Errorf("failed to get cluster slots: %w", err)
	}

	newSlotMap := make(map[uint16]string)
	for _, slotRange := range slots {
		for i := slotRange.Start; i <= slotRange.End; i++ {
			// Map to the master node address
			if len(slotRange.Nodes) > 0 {
				newSlotMap[uint16(i)] = slotRange.Nodes[0].Addr
			}
		}
	}

	r.slotMap = newSlotMap
	return nil
}

// cachePost caches a single Post object and add it into user's post list as sorted set
func (r *CachedPostRepository) cachePost(ctx context.Context, post *sqlc.Post) error {
	postJSON, err := json.Marshal(post)
	if err != nil {
		return fmt.Errorf("failed to marshal post %d: %w", post.ID, err)
	}

	pipe := r.rdb.Pipeline()

	// Cache with a generic key for direct GetPost access
	postKeyGeneric := fmt.Sprintf(postKeyGenericPattern, post.ID)
	pipe.Set(ctx, postKeyGeneric, postJSON, cacheTTL)

	// Add post ID to the user's sorted set of posts
	userPostsKey := fmt.Sprintf(userPostsKeyPattern, post.UserID)
	pipe.ZAdd(ctx, userPostsKey, &redis.Z{
		Score:  float64(post.CreatedAt.Unix()),
		Member: post.ID,
	})
	pipe.Expire(ctx, userPostsKey, cacheTTL)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("pipeline execution failed for caching post %d: %w", post.ID, err)
	}

	return nil
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
		postJSON, err := json.Marshal(p)
		if err != nil {
			log.Printf("failed to marshal post %d for batch cache: %v", p.ID, err)
			continue
		}

		// Set generic key for direct access
		postKeyGeneric := fmt.Sprintf(postKeyGenericPattern, p.ID)
		pipe.Set(ctx, postKeyGeneric, postJSON, cacheTTL)

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
