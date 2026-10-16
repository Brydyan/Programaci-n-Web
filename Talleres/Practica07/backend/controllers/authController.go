package controllers

import (
	"github.com/gofiber/fiber/v2"
	"multicatalogo-backend/interfaces"
	"multicatalogo-backend/models"
)

type AuthController struct {
	repo interfaces.DBRepository
}

func NewAuthController(repo interfaces.DBRepository) *AuthController {
	return &AuthController{repo: repo}
}

func (ac *AuthController) Login(c *fiber.Ctx) error {
	var req models.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIError{
			Status:  fiber.StatusBadRequest,
			Message: "Cuerpo de petición inválido",
		})
	}
	if req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIError{
			Status:  fiber.StatusBadRequest,
			Message: "El correo y la contraseña son obligatorios",
		})
	}
	ok, token := ac.repo.Authenticate(req.Email, req.Password)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(models.APIError{
			Status:  fiber.StatusUnauthorized,
			Message: "Credenciales incorrectas",
		})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"token": token,
		"email": req.Email,
	})
}
