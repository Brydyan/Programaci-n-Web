package config

import (
	"os"

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
