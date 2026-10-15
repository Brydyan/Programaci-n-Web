package controllers

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"multicatalogo-backend/interfaces"
	"multicatalogo-backend/models"
)

type ProductoController struct {
	repo  interfaces.DBRepository
	cache interfaces.CacheService
}

func NewProductoController(repo interfaces.DBRepository, cache interfaces.CacheService) *ProductoController {
	return &ProductoController{repo: repo, cache: cache}
}

const cacheKeyProductos = "productos:all"

func (pc *ProductoController) GetProductos(c *fiber.Ctx) error {
	ctx := context.Background()
	cached, err := pc.cache.Get(ctx, cacheKeyProductos)
	if err == nil {
		var result []models.Producto
		if jsonErr := json.Unmarshal([]byte(cached), &result); jsonErr == nil {
			return c.JSON(result)
		}
	} else if err != redis.Nil {
		_ = err
	}
	productos := pc.repo.GetAllProducts()
	if data, jsonErr := json.Marshal(productos); jsonErr == nil {
		pc.cache.Set(ctx, cacheKeyProductos, data, 5*time.Minute)
	}
	return c.JSON(productos)
}

func (pc *ProductoController) GetProductoPorID(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIError{
			Status:  fiber.StatusBadRequest,
			Message: "El ID del producto debe ser un número entero",
		})
	}
	prod, err := pc.repo.GetProductByID(id)
	if err != nil || prod == nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIError{
			Status:  fiber.StatusNotFound,
			Message: "Producto no encontrado",
		})
	}
	return c.Status(fiber.StatusOK).JSON(prod)
}
