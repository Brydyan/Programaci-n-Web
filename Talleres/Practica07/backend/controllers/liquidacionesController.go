package controllers

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"multicatalogo-backend/interfaces"
	"multicatalogo-backend/middleware"
	"multicatalogo-backend/models"
)

// LiquidacionesController cierra el ciclo del dinero: el admin ve a quién debe,
// paga, y el distribuidor ve sus cobros.
//
// La operación de liquidar es idempotente por diseño: si el distribuidor no
// tiene comisiones 'pendiente' responde 409, nunca un 201 vacío. Así un doble
// clic o dos admins pulsando a la vez no producen pagos fantasma.
type LiquidacionesController struct {
	repo interfaces.DBRepository
}

func NewLiquidacionesController(repo interfaces.DBRepository) *LiquidacionesController {
	return &LiquidacionesController{repo: repo}
}

type liquidacionRequest struct {
	ReferidoID int `json:"referido_id"`
}

// Crear liquida las comisiones pendientes del distribuidor indicado.
// POST /api/admin/liquidaciones → 201 con la liquidación creada.
func (lc *LiquidacionesController) Crear(c *fiber.Ctx) error {
	var req liquidacionRequest
	if err := c.BodyParser(&req); err != nil {
		return middleware.RespondError(c,
			fiber.StatusBadRequest,
			string(middleware.CodeValidationError),
			"cuerpo inválido",
		)
	}
	if req.ReferidoID <= 0 {
		return middleware.RespondError(c,
			fiber.StatusBadRequest,
			string(middleware.CodeValidationError),
			"referido_id debe ser un número positivo",
		)
	}

	liq, err := lc.repo.Liquidar(req.ReferidoID)
	if err != nil {
		return mapErrorLiquidacion(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(liq)
}

// Listar devuelve el historial de pagos. Admin only.
// GET /api/admin/liquidaciones?page=&limit=
func (lc *LiquidacionesController) Listar(c *fiber.Ctx) error {
	return lc.listar(c, nil)
}

// ListarMias devuelve los cobros del usuario del token.
// GET /api/liquidaciones/mias?page=&limit=
func (lc *LiquidacionesController) ListarMias(c *fiber.Ctx) error {
	claimsAny := c.Locals(middleware.LocalUserKey)
	claims, ok := claimsAny.(middleware.JWTClaims)
	if !ok {
		return middleware.RespondError(c,
			fiber.StatusUnauthorized,
			string(middleware.CodeUnauthorized),
			"token sin identidad de usuario",
		)
	}

	referidoID, err := lc.repo.GetReferidoIDByUsuario(claims.Sub)
	if err != nil {
		return middleware.RespondError(c,
			fiber.StatusInternalServerError,
			string(middleware.CodeInternal),
			"error consultando el nodo de red del usuario",
			err.Error(),
		)
	}
	// Sin nodo todavía: no hay pagos que mostrar, no es un error.
	if referidoID == nil {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"data": []models.Liquidacion{}, "page": 1, "limit": 20, "total": 0,
		})
	}
	return lc.listar(c, referidoID)
}

// Deuda devuelve cuánto se le debe a cada distribuidor. Admin only.
// GET /api/admin/liquidaciones/deuda
func (lc *LiquidacionesController) Deuda(c *fiber.Ctx) error {
	deudas, err := lc.repo.GetDeuda()
	if err != nil {
		return middleware.RespondError(c,
			fiber.StatusInternalServerError,
			string(middleware.CodeInternal),
			"error consultando la deuda",
			err.Error(),
		)
	}
	return c.Status(fiber.StatusOK).JSON(deudas)
}

func (lc *LiquidacionesController) listar(c *fiber.Ctx, referidoID *int) error {
	page, limit := queryInt(c, "page"), queryInt(c, "limit")

	var (
		liquidaciones []models.Liquidacion
		total         int
		err           error
	)
	if referidoID != nil {
		liquidaciones, total, err = lc.repo.GetLiquidacionesDe(*referidoID, page, limit)
	} else {
		liquidaciones, total, err = lc.repo.GetLiquidaciones(page, limit)
	}
	if err != nil {
		return middleware.RespondError(c,
			fiber.StatusInternalServerError,
			string(middleware.CodeInternal),
			"error consultando liquidaciones",
			err.Error(),
		)
	}

	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": liquidaciones, "page": page, "limit": limit, "total": total,
	})
}

// mapErrorLiquidacion traduce los sentinelas del dominio a códigos HTTP.
func mapErrorLiquidacion(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, models.ErrNadaQueLiquidar):
		return middleware.RespondError(c,
			fiber.StatusConflict,
			string(middleware.CodeConflict),
			"no hay comisiones pendientes que liquidar",
		)
	case errors.Is(err, models.ErrReferidoInvalido):
		return middleware.RespondError(c,
			fiber.StatusNotFound,
			string(middleware.CodeNotFound),
			"distribuidor no encontrado",
		)
	default:
		return middleware.RespondError(c,
			fiber.StatusInternalServerError,
			string(middleware.CodeInternal),
			"error liquidando comisiones",
			err.Error(),
		)
	}
}
