package controllers

import (
	"github.com/gofiber/fiber/v2"
	"multicatalogo-backend/models"
	"multicatalogo-backend/repository"
)

// Login es la función controladora que se ejecutará cuando el cliente envíe sus credenciales.
func Login(c *fiber.Ctx) error {
	var req models.LoginRequest

	// Intentamos parsear el cuerpo JSON entrante.
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIError{
			Status:  fiber.StatusBadRequest,
			Message: "Cuerpo de petición inválido",
		})
	}

	// Validamos que los campos obligatorios no estén vacíos.
	if req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIError{
			Status:  fiber.StatusBadRequest,
			Message: "El correo y la contraseña son obligatorios",
		})
	}

	// Autenticamos mediante el repositorio.
	ok, token := repository.Authenticate(req.Email, req.Password)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(models.APIError{
			Status:  fiber.StatusUnauthorized,
			Message: "Credenciales incorrectas",
		})
	}

	// Si las credenciales son válidas, respondemos con el token y el email.
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"token": token,
		"email": req.Email,
	})
}