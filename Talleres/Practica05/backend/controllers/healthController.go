package controllers

import (
	"github.com/gofiber/fiber/v2"
)

// HealthCheck verifica el estado operativo del backend.
func HealthCheck(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "ok",
		"message": "Servidor Go/Fiber operativo",
	})
}
