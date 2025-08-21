package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/n1207n/cache-query-aggregator/config"
	"github.com/n1207n/cache-query-aggregator/db/sqlc"
	"github.com/n1207n/cache-query-aggregator/internal/handler"
	"github.com/n1207n/cache-query-aggregator/internal/repository"
	approuter "github.com/n1207n/cache-query-aggregator/internal/router"
	"github.com/n1207n/cache-query-aggregator/internal/service"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Configuration loaded successfully. App Env: %s, Server: %d", cfg.AppEnv, cfg.AppPort)

	dbPool, err := initDB(cfg.DbURL)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer dbPool.Close()
	log.Println("Database connection pool established.")

	redisAddrs := strings.Split(cfg.RedisURL, ",")
	if len(redisAddrs) == 0 || redisAddrs[0] == "" {
		log.Fatalf("Failed to initialize Redis: %v", fmt.Errorf("redis address is not configured"))
	}

	if len(redisAddrs) == 1 {
		rdb, err := initSingleRedis(redisAddrs[0])
		if err != nil {
			log.Fatalf("Failed to initialize Single Redis: %v", err)
		}

		defer rdb.Close()
		log.Println("Redis Single client initialized.")
	} else {
		rdb, err := initClusterRedis(redisAddrs)
		if err != nil {
			log.Fatalf("Failed to initialize Cluster Redis: %v", err)
		}

		defer rdb.Close()
		log.Println("Redis Cluster client initialized.")
	}

	sqlcQuerier := sqlc.New(dbPool)
	log.Println("SQLC Querier initialized.")

	userRepo := repository.NewDBUserRepository(sqlcQuerier)
	log.Println("User repository initialized.")
	postRepo := repository.NewDBPostRepository(sqlcQuerier)
	log.Println("Post repository initialized.")

	// Initialize Services
	userService := service.NewUserService(userRepo) // Example
	log.Println("User service initialized.")
	postService := service.NewPostService(postRepo)
	log.Println("Post service initialized.")

	// Initialize Gin router
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.Default()

	// Initialize Handlers
	userHandler := handler.NewUserHandler(userService)
	log.Println("User handler initialized.")
	postHandler := handler.NewPostHandler(postService)
	log.Println("Post handler initialized.")

	// Setup routes
	v1 := router.Group("/api/v1")
	{
		approuter.SetupUserRoutes(v1, userHandler)
		approuter.SetupPostRoutes(v1, postHandler)
	}

	// Ping route for health check
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.AppPort),
		Handler: router,
	}

	go func() {
		log.Printf("Server listening on %d", cfg.AppPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}

func initDB(databaseURL string) (*pgxpool.Pool, error) {
	pgxpoolCfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database URL: %w", err)
	}

	// You can configure pool settings here, e.g.,
	// pgxpool_cfg.MaxConns = 10
	// pgxpool_cfg.MinConns = 2
	// pgxpool_cfg.MaxConnLifetime = time.Hour
	// pgxpool_cfg.MaxConnIdleTime = 30 * time.Minute
	// pgxpool_cfg.HealthCheckPeriod = time.Minute
	// pgxpool_cfg.ConnConfig.ConnectTimeout = 5 * time.Second

	dbPool, err := pgxpool.NewWithConfig(context.Background(), pgxpoolCfg)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Ping the database to verify connection
	if err = dbPool.Ping(context.Background()); err != nil {
		dbPool.Close() // Close the pool if ping fails
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	return dbPool, nil
}

func initSingleRedis(redisURL string) (*redis.Client, error) {
	var rdb *redis.Client
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("could not parse Redis URL: %w", err)
	}

	rdb = redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := rdb.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("could not connect to Redis: %w", err)
	}

	return rdb, nil
}

func initClusterRedis(redisAddrs []string) (*redis.ClusterClient, error) {
	var rdb *redis.ClusterClient
	rdb = redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: redisAddrs,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := rdb.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("could not connect to Redis: %w", err)
	}

	return rdb, nil
}
