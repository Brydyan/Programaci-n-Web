package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"multicatalogo-backend/interfaces"
	"multicatalogo-backend/models"
)

type CheckoutController struct {
	repo  interfaces.DBRepository
	cache interfaces.CacheService
}

func NewCheckoutController(repo interfaces.DBRepository, cache interfaces.CacheService) *CheckoutController {
	return &CheckoutController{repo: repo, cache: cache}
}

// Checkout procesa la compra con una transacción ACID.
// Si el COMMIT es exitoso, lanza una Goroutine para calcular comisiones sin bloquear la respuesta.
func (cc *CheckoutController) Checkout(c *fiber.Ctx) error {
	var req models.CheckoutRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIError{
			Status:  fiber.StatusBadRequest,
			Message: "Cuerpo de petición inválido",
		})
	}
	if req.Email == "" || len(req.Items) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIError{
			Status:  fiber.StatusBadRequest,
			Message: "Email e items son obligatorios",
		})
	}

	pedido, err := cc.repo.ProcessCheckout(req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIError{
			Status:  fiber.StatusInternalServerError,
			Message: fmt.Sprintf("Error procesando la compra: %s", err.Error()),
		})
	}

	// Lanzar Goroutine — no bloquea la respuesta al cliente
	go cc.calcularComisiones(pedido.Total, req.Email)

	return c.Status(fiber.StatusCreated).JSON(pedido)
}

// calcularComisiones corre en segundo plano (Goroutine).
// Actualiza comisiones en DB e invalida el caché de la red.
func (cc *CheckoutController) calcularComisiones(total float64, emailUsuario string) {
	time.Sleep(200 * time.Millisecond) // Simula latencia de cálculo
	cc.repo.ActualizarComisiones(total, emailUsuario)
	ctx := context.Background()
	cc.cache.Delete(ctx, cacheKeyRed)
	fmt.Printf("[Goroutine] Cache 'red:tree:all' invalidado tras compra de %s\n", emailUsuario)
}
