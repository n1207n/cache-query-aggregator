package config

import (
	"os"
	"strconv"
)

// Config holds all configuration for the application
type Config struct {
	AppEnv    string
	AppPort   int
	DbURL     string
	RedisURL  string
	SecretKey string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	return &Config{
		AppEnv:    getEnv("APP_ENV", "development"),
		AppPort:   getEnvAsInt("APP_PORT", 8080),
		DbURL:     getEnv("POSTGRES_URL", "postgres://user:password@db:5432/mydatabase?sslmode=disable"),
		RedisURL:  getEnv("REDIS_CLUSTER_URLS", "redis-1:7001,redis-2:7002,redis-3:7003,redis-4:7004,redis-5:7005"),
		SecretKey: getEnv("SECRET_KEY", "supersecret"),
	}, nil
}

// Helper function to get an environment variable or return a default value
func getEnv(key string, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// Helper function to get an environment variable as int or return a default value
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}
