package controllers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"multicatalogo-backend/models"
	"multicatalogo-backend/repository"
)

// GetProductos es la función controladora encargada de devolver el catálogo completo de productos.
func GetProductos(c *fiber.Ctx) error {
	productos := repository.GetAllProducts()
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