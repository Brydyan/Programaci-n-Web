package controllers

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"multicatalogo-backend/config"
	"multicatalogo-backend/models"
	"multicatalogo-backend/repository"
)

const cacheKeyRed = "red:tree:all"

// GetRed implementa el patrón Cache-Aside para la red de referidos.
// La reconstrucción del árbol es una operación costosa, por eso se cachea 10 minutos.
// 1. Intenta leer desde Redis (Cache Hit → retorna sin tocar el repositorio).
// 2. Si no hay caché (Cache Miss), construye el árbol desde el repositorio, guarda en Redis con TTL de 10 min y retorna.
func GetRed(c *fiber.Ctx) error {
	ctx := context.Background()

	cached, err := config.RedisClient.Get(ctx, cacheKeyRed).Result()
	if err == nil {
		var result []models.Referido
		if jsonErr := json.Unmarshal([]byte(cached), &result); jsonErr == nil {
			return c.Status(fiber.StatusOK).JSON(result)
		}
	} else if err != redis.Nil {
		// Error de conexión con Redis: continúa hacia el repositorio (degradación elegante)
		_ = err
	}

	red := repository.GetNetwork()

	if data, jsonErr := json.Marshal(red); jsonErr == nil {
		config.RedisClient.Set(ctx, cacheKeyRed, data, 10*time.Minute)
	}

	return c.Status(fiber.StatusOK).JSON(red)
}
