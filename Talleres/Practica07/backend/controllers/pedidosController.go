package controllers

import (
	"context"
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"multicatalogo-backend/domain"
	"multicatalogo-backend/interfaces"
	"multicatalogo-backend/internal/dto"
	"multicatalogo-backend/middleware"
	"multicatalogo-backend/models"
)

// PedidosController agrupa los endpoints de gestión de pedidos del
// cliente autenticado y del admin:
//
//	GET    /api/pedidos/mios             — lista pedidos del usuario
//	GET    /api/pedidos/:id              — detalle (dueño o admin)
//	GET    /api/admin/pedidos            — lista global con filtro estado
//	PATCH  /api/admin/pedidos/:id/estado — cambia estado con FSM
type PedidosController struct {
	repo  interfaces.DBRepository
	cache interfaces.CacheService
}

func NewPedidosController(repo interfaces.DBRepository, cache interfaces.CacheService) *PedidosController {
	return &PedidosController{repo: repo, cache: cache}
}

// ─────────────────────────── Cliente autenticado ───────────────────────────

// GetMisPedidos implementa GET /api/pedidos/mios.
//
// Devuelve los pedidos del usuario autenticado (claims.Sub) paginados.
// Query params:
//
//	page  — 1-based (default 1)
//	limit — 1..100 (default 20)
//
// Respuesta: { data, page, limit, total } — mismo formato que
// /api/admin/productos para consistencia con el resto del backend.
func (pc *PedidosController) GetMisPedidos(c *fiber.Ctx) error {
	claims, ok := c.Locals(middleware.LocalUserKey).(middleware.JWTClaims)
	if !ok {
		return middleware.RespondError(c, fiber.StatusUnauthorized,
			string(middleware.CodeUnauthorized), "autenticación requerida")
	}
	page, limit := parsePagination(c)

	rows, total, err := pc.repo.GetMisPedidos(claims.Sub, page, limit)
	if err != nil {
		return middleware.RespondError(c, fiber.StatusInternalServerError,
			string(middleware.CodeInternal), "error consultando pedidos", err.Error())
	}
	return c.JSON(fiber.Map{
		"data":  rows,
		"page":  page,
		"limit": limit,
		"total": total,
	})
}

// GetPedidoDetalle implementa GET /api/pedidos/:id.
//
// Autorización:
//   - admin: ve cualquier pedido.
//   - cliente: sólo si el pedido.UsuarioID == claims.Sub.
//
// Si no se cumple la autorización devolvemos 404 NOT_FOUND (no 403) —
// decisión de seguridad para no exponer la existencia del pedido a
// terceros.
func (pc *PedidosController) GetPedidoDetalle(c *fiber.Ctx) error {
	claims, ok := c.Locals(middleware.LocalUserKey).(middleware.JWTClaims)
	if !ok {
		return middleware.RespondError(c, fiber.StatusUnauthorized,
			string(middleware.CodeUnauthorized), "autenticación requerida")
	}
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return middleware.RespondError(c, fiber.StatusBadRequest,
			string(middleware.CodeValidationError), "id inválido")
	}

	det, err := pc.repo.GetPedidoDetalle(id)
	if errors.Is(err, models.ErrPedidoNoEncontrado) {
		return middleware.RespondError(c, fiber.StatusNotFound,
			string(middleware.CodeNotFound), "pedido no encontrado")
	}
	if err != nil {
		return middleware.RespondError(c, fiber.StatusInternalServerError,
			string(middleware.CodeInternal), "error consultando pedido", err.Error())
	}

	if claims.Rol != "admin" {
		if det.UsuarioID == nil || *det.UsuarioID != claims.Sub {
			// 404 (no exponer existencia) en vez de 403.
			return middleware.RespondError(c, fiber.StatusNotFound,
				string(middleware.CodeNotFound), "pedido no encontrado")
		}
	}
	return c.JSON(det)
}

// ─────────────────────────── Admin ───────────────────────────

// GetPedidosAdmin implementa GET /api/admin/pedidos.
//
// Query params:
//
//	estado — opcional: pendiente | preparando | enviado | entregado | cancelado
//	page   — 1-based (default 1)
//	limit  — 1..100 (default 20)
//
// Respuesta: { data, page, limit, total }.
func (pc *PedidosController) GetPedidosAdmin(c *fiber.Ctx) error {
	page, limit := parsePagination(c)
	filtro := interfaces.PedidoFiltro{
		Estado: c.Query("estado", ""),
		Page:   page,
		Limit:  limit,
	}

	rows, total, err := pc.repo.GetPedidosAdmin(filtro)
	if err != nil {
		return middleware.RespondError(c, fiber.StatusInternalServerError,
			string(middleware.CodeInternal), "error listando pedidos", err.Error())
	}
	return c.JSON(fiber.Map{
		"data":  rows,
		"page":  page,
		"limit": limit,
		"total": total,
	})
}

// PatchEstado implementa PATCH /api/admin/pedidos/:id/estado.
//
// Valida el DTO (validator oneof) y delega al repository, que aplica la
// FSM en una transacción con lock FOR UPDATE. Side-effects al cancelar:
// devuelve stock reservado + revierte comisiones pendientes/aprobadas.
//
// Tras el commit invalida el caché del catálogo público (porque el
// stock cambió si fue cancelación).
func (pc *PedidosController) PatchEstado(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return middleware.RespondError(c, fiber.StatusBadRequest,
			string(middleware.CodeValidationError), "id inválido")
	}

	var d dto.PedidoEstadoUpdate
	if err := c.BodyParser(&d); err != nil {
		return middleware.RespondError(c, fiber.StatusBadRequest,
			string(middleware.CodeValidationError), "cuerpo inválido")
	}
	if err := dto.Validate.Struct(&d); err != nil {
		return middleware.RespondError(c, fiber.StatusBadRequest,
			string(middleware.CodeValidationError), "estado inválido", err.Error())
	}

	if err := pc.repo.CambiarEstadoPedido(id, domain.EstadoPedido(d.Estado)); err != nil {
		switch {
		case errors.Is(err, models.ErrPedidoNoEncontrado):
			return middleware.RespondError(c, fiber.StatusNotFound,
				string(middleware.CodeNotFound), "pedido no encontrado")
		case errors.Is(err, domain.ErrTransicionInvalida):
			return middleware.RespondError(c, fiber.StatusConflict,
				string(middleware.CodeInvalidTrans), err.Error())
		default:
			return middleware.RespondError(c, fiber.StatusInternalServerError,
				string(middleware.CodeInternal), "error cambiando estado", err.Error())
		}
	}

	// Cualquier cambio de estado altera los agregados del dashboard (cancelar
	// resta venta; entregado la suma). Se sube la versión en vez de purgar
	// claves: las viejas dejan de direccionarse y expiran por TTL.
	InvalidarReportes(context.Background(), pc.cache)

	// Si fue cancelación, el stock cambió → invalidar caché público.
	if domain.EstadoPedido(d.Estado) == domain.EstadoCancelado {
		_ = pc.cache.Delete(context.Background(), cacheKeyProductos)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// parsePagination extrae page/limit de los query params con defaults
// sensatos: page=1 (min 1), limit=20 (min 1, max 100). Se usa tanto en
// GetMisPedidos como en GetPedidosAdmin para homogeneizar respuestas.
func parsePagination(c *fiber.Ctx) (page, limit int) {
	page = parseIntDefault(c.Query("page", ""), 1)
	if page < 1 {
		page = 1
	}
	limit = parseIntDefault(c.Query("limit", ""), 20)
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}
