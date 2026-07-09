package cache

import (
	"context"
	"fmt"
	"time"

	"api-first/internal/config"
	"api-first/internal/logger"

	"github.com/redis/go-redis/v9"
)

var client *redis.Client

// Init creates a singleton Redis client configured for air-gapped private cloud deployment.
func Init(cfg *config.RedisConfig) error {
	log := logger.Logger()

	client = redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:       cfg.Password,
		DB:           cfg.DB,
		MinIdleConns: cfg.MinIdleConns,
		MaxIdleConns: cfg.MaxIdleConns,
		MaxIdleTime:  cfg.MaxIdleTime,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.WithError(err).Error("failed to connect to Redis")
		return fmt.Errorf("redis connection failed: %w", err)
	}

	log.WithField("addr", cfg.Addr).Info("successfully connected to Redis cache")
	return nil
}

// Get returns the underlying Redis client for direct operations.
func Get() *redis.Client {
	return client
}

// Set stores a value in Redis with the given TTL.
func Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return client.Set(ctx, key, value, ttl).Err()
}

// GetStr retrieves a string value from Redis.
func GetStr(ctx context.Context, key string) (string, error) {
	val, err := client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil // key not found is not an error
	}
	return val, err
}

// Delete removes a key from Redis.
func Delete(ctx context.Context, key string) error {
	return client.Del(ctx, key).Err()
}

// Exists checks if a key exists in Redis.
func Exists(ctx context.Context, key string) (bool, error) {
	result, err := client.Exists(ctx, key).Result()
	return result > 0, err
}

// Close shuts down the Redis client gracefully.
func Close() error {
	return client.Close()
}