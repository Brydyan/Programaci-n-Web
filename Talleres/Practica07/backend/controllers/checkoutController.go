package controllers

import (
	"context"
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"multicatalogo-backend/interfaces"
	"multicatalogo-backend/middleware"
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
//
// F3 migró este handler para usar la identidad del JWT (RequireAuth en
// routes.SetupRoutes) en vez del email del body. El usuario_id se
// inyecta en el INSERT de pedidos para que el pedido quede asociado al
// usuario y el listado GET /api/pedidos/mios pueda filtrar.
//
// Tras el COMMIT, lanza una Goroutine para:
//   - Calcular las comisiones (upline walk) enlazándolas al pedido_id
//     recién creado. Antes (F1) iban con pedido_id NULL; ahora pueden
//     revertirse al cancelar.
//   - Invalidar el caché de la red (las comisiones afectan al árbol).
//
// El cálculo de stock ocurre DENTRO de la tx del repo (FOR UPDATE), de
// modo que si la compra no es posible el ROLLBACK deja el stock intacto.
func (cc *CheckoutController) Checkout(c *fiber.Ctx) error {
	// Identidad del usuario desde el JWT — RequireAuth ya validó el token.
	claimsAny := c.Locals(middleware.LocalUserKey)
	claims, ok := claimsAny.(middleware.JWTClaims)
	if !ok {
		return middleware.RespondError(c, fiber.StatusUnauthorized,
			string(middleware.CodeUnauthorized), "autenticación requerida")
	}

	var req models.CheckoutRequest
	if err := c.BodyParser(&req); err != nil {
		return middleware.RespondError(c, fiber.StatusBadRequest,
			string(middleware.CodeValidationError), "Cuerpo de petición inválido")
	}
	if len(req.Items) == 0 {
		return middleware.RespondError(c, fiber.StatusBadRequest,
			string(middleware.CodeValidationError), "El carrito está vacío")
	}
	if req.MetodoPago == "" {
		return middleware.RespondError(c, fiber.StatusBadRequest,
			string(middleware.CodeValidationError), "metodo_pago requerido")
	}

	// Si el cliente no envió email, usar el del JWT (compatibilidad con
	// el FE legacy que lo mandaba en el body).
	if req.Email == "" {
		req.Email = claims.Email
	}

	detalle, err := cc.repo.CreatePedidoConUsuario(req, claims.Sub)
	if err != nil {
		if errors.Is(err, models.ErrProductoNoEncontrado) {
			return middleware.RespondError(c, fiber.StatusConflict,
				string(middleware.CodeStockInsufic),
				"uno o más productos no tienen stock suficiente", err.Error())
		}
		return middleware.RespondError(c, fiber.StatusInternalServerError,
			string(middleware.CodeInternal), "error procesando la compra", err.Error())
	}

	// Post-COMMIT: comisiones y caché.
	go cc.calcularComisiones(detalle.ID, detalle.Total, req.Email)

	return c.Status(fiber.StatusCreated).JSON(detalle)
}

// calcularComisiones corre en segundo plano (Goroutine).
// Actualiza comisiones en DB e invalida el caché de la red.
func (cc *CheckoutController) calcularComisiones(pedidoID int, total float64, emailUsuario string) {
	time.Sleep(200 * time.Millisecond) // Simula latencia de cálculo
	cc.repo.ActualizarComisiones(pedidoID, total, emailUsuario)
	ctx := context.Background()
	cc.cache.Delete(ctx, cacheKeyRed)
}