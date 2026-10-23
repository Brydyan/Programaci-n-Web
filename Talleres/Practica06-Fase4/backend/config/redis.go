package config

import (
	"context"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisClient es el cliente global de conexión a Redis.
var RedisClient *redis.Client

func init() {
	addr := os.Getenv("REDIS_URL")
	if addr == "" {
		addr = "localhost:6379"
	}
	RedisClient = redis.NewClient(&redis.Options{
		Addr: addr,
	})
}

type RedisService struct {
	client *redis.Client
}

func NewRedisService() *RedisService {
	return &RedisService{client: RedisClient}
}

func (r *RedisService) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

func (r *RedisService) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

func (r *RedisService) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}
