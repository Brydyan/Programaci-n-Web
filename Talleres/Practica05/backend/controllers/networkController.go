package controllers

import (
	"github.com/gofiber/fiber/v2"
	"multicatalogo-backend/repository"
)

// GetRed retorna la red de referidos obtenida del repositorio.
func GetRed(c *fiber.Ctx) error {
	red := repository.GetNetwork()
	return c.Status(fiber.StatusOK).JSON(red)
}
