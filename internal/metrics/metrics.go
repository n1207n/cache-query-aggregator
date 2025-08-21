package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// PostCacheHits calculates # of cache hits
	PostCacheHits = promauto.NewCounter(prometheus.CounterOpts{
		Name: "post_repository_cache_hits_total",
		Help: "The total number of cache hits for post repository",
	})

	// PostCacheMisses calculates # of cache misses
	PostCacheMisses = promauto.NewCounter(prometheus.CounterOpts{
		Name: "post_repository_cache_misses_total",
		Help: "The total number of cache misses for post repository",
	})

	// PostCacheShardJoins calculates # of partial cache hits
	PostCacheShardJoins = promauto.NewCounter(prometheus.CounterOpts{
		Name: "post_repository_cache_shard_joins_total",
		Help: "The total number of times a partial cache hit required a DB query (shard join).",
	})

	// PostDBQueries는 calculates # of Post queries
	PostDBQueries = promauto.NewCounter(prometheus.CounterOpts{
		Name: "post_repository_db_queries_total",
		Help: "The total number of queries made to the DB from post repository.",
	})

	// RedisNodeReadsByUser tells # of nodes accessed by userId
	RedisNodeReadsByUser = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "redis_node_reads_by_user_total",
		Help: "Total number of reads from a specific Redis node, partitioned by user.",
	}, []string{"node_addr", "user_id"})
)
