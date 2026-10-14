package redis

import (
	"github.com/d0hx4n2503/bento-backend-golang/config"
	"github.com/redis/go-redis/v9"
)

// Returns new redis client
func NewRedisClient(cfg *config.Config) *redis.Client {
	redisHost := cfg.Redis.Address

	if redisHost == "" {
		redisHost = ":6379"
	}

	client := redis.NewClient(&redis.Options{
		Addr:         redisHost,
		MinIdleConns: cfg.Redis.MinIdleConns,
		PoolSize:     cfg.Redis.PoolSize,
		PoolTimeout:  cfg.Redis.PoolTimeout,
		Password:     cfg.Redis.Password, // no password set
		DB:           cfg.Redis.DB,       // use default DB
	})

	return client
}
