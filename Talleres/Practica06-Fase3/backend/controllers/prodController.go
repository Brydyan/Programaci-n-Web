package controllers

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"multicatalogo-backend/config"
	"multicatalogo-backend/models"
	"multicatalogo-backend/repository"
)

const cacheKeyProductos = "productos:all"

// GetProductos implementa el patrón Cache-Aside para el catálogo completo.
// 1. Intenta leer desde Redis (Cache Hit → retorna sin tocar el repositorio).
// 2. Si no hay caché (Cache Miss), consulta el repositorio, guarda en Redis con TTL de 5 min y retorna.
func GetProductos(c *fiber.Ctx) error {
	ctx := context.Background()

	cached, err := config.RedisClient.Get(ctx, cacheKeyProductos).Result()
	if err == nil {
		var result []models.Producto
		if jsonErr := json.Unmarshal([]byte(cached), &result); jsonErr == nil {
			return c.JSON(result)
		}
	} else if err != redis.Nil {
		// Error de conexión con Redis: continúa hacia el repositorio (degradación elegante)
		_ = err
	}

	productos := repository.GetAllProducts()

	if data, jsonErr := json.Marshal(productos); jsonErr == nil {
		config.RedisClient.Set(ctx, cacheKeyProductos, data, 5*time.Minute)
	}

	return c.JSON(productos)
}

// GetProductoPorID obtiene un producto por su ID numérico.
func GetProductoPorID(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIError{
			Status:  fiber.StatusBadRequest,
			Message: "El ID del producto debe ser un número entero",
		})
	}

	prod, err := repository.GetProductByID(id)
	if err != nil || prod == nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIError{
			Status:  fiber.StatusNotFound,
			Message: "Producto no encontrado",
		})
	}

	return c.Status(fiber.StatusOK).JSON(prod)
}
