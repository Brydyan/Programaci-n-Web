package controllers

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"multicatalogo-backend/interfaces"
	"multicatalogo-backend/models"
)

type NetworkController struct {
	repo  interfaces.DBRepository
	cache interfaces.CacheService
}

func NewNetworkController(repo interfaces.DBRepository, cache interfaces.CacheService) *NetworkController {
	return &NetworkController{repo: repo, cache: cache}
}

const cacheKeyRed = "red:tree:all"

func (nc *NetworkController) GetRed(c *fiber.Ctx) error {
	ctx := context.Background()
	cached, err := nc.cache.Get(ctx, cacheKeyRed)
	if err == nil {
		var result []models.Referido
		if jsonErr := json.Unmarshal([]byte(cached), &result); jsonErr == nil {
			return c.Status(fiber.StatusOK).JSON(result)
		}
	} else if err != redis.Nil {
		_ = err
	}
	red := nc.repo.GetNetwork()
	if data, jsonErr := json.Marshal(red); jsonErr == nil {
		nc.cache.Set(ctx, cacheKeyRed, data, 10*time.Minute)
	}
	return c.Status(fiber.StatusOK).JSON(red)
}
