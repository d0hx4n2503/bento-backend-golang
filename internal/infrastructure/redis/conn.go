package redis

import (
	"github.com/d0hx4n2503/bento-backend-golang/config"
	"github.com/redis/go-redis/v9"
)

// NewRedisClient constructs a client for the shared Redis deployment.
func NewRedisClient(cfg *config.Config) *redis.Client {
	redisHost := cfg.Redis.Address

	if redisHost == "" {
		redisHost = "localhost:6379"
	}

	client := redis.NewClient(&redis.Options{
		Addr:         redisHost,
		MinIdleConns: cfg.Redis.MinIdleConns,
		PoolSize:     cfg.Redis.PoolSize,
		PoolTimeout:  cfg.Redis.PoolTimeout,
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
	})

	return client
}
