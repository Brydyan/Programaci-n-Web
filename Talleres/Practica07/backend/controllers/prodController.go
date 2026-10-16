package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"

	"multicatalogo-backend/internal/dto"
	"multicatalogo-backend/interfaces"
	"multicatalogo-backend/middleware"
	"multicatalogo-backend/models"
)

// ProductoController expone los endpoints del catálogo público
// (/api/productos, /api/productos/:id) y el CRUD admin
// (/api/admin/productos) definido en routes.SetupRoutes.
//
// Patrón cache-aside sobre el listado público:
//
//   - GET /api/productos  → lee `productos:all` (TTL 5min)
//   - POST/PUT/DELETE admin → invalida `productos:all`
type ProductoController struct {
	repo  interfaces.DBRepository
	cache interfaces.CacheService
}

func NewProductoController(repo interfaces.DBRepository, cache interfaces.CacheService) *ProductoController {
	return &ProductoController{repo: repo, cache: cache}
}

// cacheKeyProductos es la clave Redis para el listado público cacheado.
// TTL definido en getCacheTTL().
const cacheKeyProductos = "productos:all"

// getCacheTTL devuelve 5 minutos — exportado lógicamente (no público) para
// que cualquier cambio de TTL quede centralizado.
const cacheTTLProductos = 5 * time.Minute

// ─────────────────────────── Vista pública ───────────────────────────

// GetProductos devuelve el catálogo activo. Lee de Redis primero (cache
// miss = repo + Set). Errores de Redis NO bloquean la respuesta — caemos
// al repo transparentemente.
func (pc *ProductoController) GetProductos(c *fiber.Ctx) error {
	ctx := context.Background()

	if cached, err := pc.cache.Get(ctx, cacheKeyProductos); err == nil && cached != "" {
		var result []models.Producto
		if jsonErr := json.Unmarshal([]byte(cached), &result); jsonErr == nil {
			return c.Type("json").SendString(cached)
		}
		// si falla Unmarshal (esquema viejo), seguimos al repo.
	} else if err != nil && !errors.Is(err, redis.Nil) {
		// error real de Redis: log mental, seguimos al repo.
		_ = err
	}

	productos, err := pc.repo.GetAllProductos()
	if err != nil {
		return middleware.RespondError(c, fiber.StatusInternalServerError,
			string(middleware.CodeInternal), "error consultando productos", err.Error())
	}

	if data, jsonErr := json.Marshal(productos); jsonErr == nil {
		_ = pc.cache.Set(ctx, cacheKeyProductos, string(data), cacheTTLProductos)
	}
	return c.JSON(productos)
}

// GetProductoPorID devuelve el producto por id (404 si no existe).
// El público lo consulta por id; el admin ya lo ve también (incluso
// inactivos) a través de este mismo handler.
func (pc *ProductoController) GetProductoPorID(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return middleware.RespondError(c, fiber.StatusBadRequest,
			string(middleware.CodeValidationError), "id inválido")
	}
	p, err := pc.repo.GetProductoByID(id)
	if errors.Is(err, models.ErrProductoNoEncontrado) {
		return middleware.RespondError(c, fiber.StatusNotFound,
			string(middleware.CodeNotFound), "producto no encontrado")
	}
	if err != nil {
		return middleware.RespondError(c, fiber.StatusInternalServerError,
			string(middleware.CodeInternal), "error consultando producto", err.Error())
	}
	return c.JSON(p)
}

// ─────────────────────────── Vista admin ───────────────────────────
//
// Estos handlers se montan detrás de middleware.RequireAuth + RequireAdmin
// en routes.SetupRoutes, así que ya sabemos que el usuario es admin. Aquí
// sólo validamos el body y los query params.

// AdminList implementa GET /api/admin/productos.
//
// Query params:
//
//	q       — substring a buscar en nombre y categoria
//	activo  — true | false | (omitido = todos)
//	page    — 1-based (default 1)
//	limit   — 1..100 (default 20)
//
// Respuesta: { data, page, limit, total }
func (pc *ProductoController) AdminList(c *fiber.Ctx) error {
	filtro := interfaces.ProductoFiltro{
		Q:    c.Query("q", ""),
		Page: parseIntDefault(c.Query("page", ""), 1),
		Limit: parseIntDefault(c.Query("limit", ""), 20),
	}
	if rawActivo := c.Query("activo", ""); rawActivo != "" {
		v, err := strconv.ParseBool(rawActivo)
		if err != nil {
			return middleware.RespondError(c, fiber.StatusBadRequest,
				string(middleware.CodeValidationError),
				"activo debe ser true o false",
			)
		}
		filtro.Activo = &v
	}

	data, total, err := pc.repo.GetProductosAdmin(filtro)
	if err != nil {
		return middleware.RespondError(c, fiber.StatusInternalServerError,
			string(middleware.CodeInternal), "error listando productos", err.Error())
	}

	return c.JSON(fiber.Map{
		"data":  data,
		"page":  filtro.Page,
		"limit": filtro.Limit,
		"total": total,
	})
}

// AdminCreate implementa POST /api/admin/productos.
// Valida el DTO, persiste en DB e invalida el caché público para que
// la siguiente GET /api/productos incluya el nuevo producto.
func (pc *ProductoController) AdminCreate(c *fiber.Ctx) error {
	var d dto.ProductoCreate
	if err := c.BodyParser(&d); err != nil {
		return middleware.RespondError(c, fiber.StatusBadRequest,
			string(middleware.CodeValidationError), "cuerpo inválido")
	}
	if err := dto.Validate.Struct(&d); err != nil {
		return middleware.RespondError(c, fiber.StatusBadRequest,
			string(middleware.CodeValidationError), "datos inválidos", err.Error())
	}

	p, err := pc.repo.CreateProducto(d)
	if err != nil {
		return middleware.RespondError(c, fiber.StatusInternalServerError,
			string(middleware.CodeInternal), "error creando producto", err.Error())
	}

	// Invalidar caché público — cualquier write debe limpiar la lista cacheada.
	_ = pc.cache.Delete(context.Background(), cacheKeyProductos)

	return c.Status(fiber.StatusCreated).JSON(p)
}

// AdminUpdate implementa PUT /api/admin/productos/:id. PUT parcial: sólo
// los campos enviados se modifican (gracias al COALESCE en el repo).
func (pc *ProductoController) AdminUpdate(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return middleware.RespondError(c, fiber.StatusBadRequest,
			string(middleware.CodeValidationError), "id inválido")
	}

	var d dto.ProductoUpdate
	if err := c.BodyParser(&d); err != nil {
		return middleware.RespondError(c, fiber.StatusBadRequest,
			string(middleware.CodeValidationError), "cuerpo inválido")
	}
	if err := dto.Validate.Struct(&d); err != nil {
		return middleware.RespondError(c, fiber.StatusBadRequest,
			string(middleware.CodeValidationError), "datos inválidos", err.Error())
	}

	p, err := pc.repo.UpdateProducto(id, d)
	if errors.Is(err, models.ErrProductoNoEncontrado) {
		return middleware.RespondError(c, fiber.StatusNotFound,
			string(middleware.CodeNotFound), "producto no encontrado")
	}
	if err != nil {
		return middleware.RespondError(c, fiber.StatusInternalServerError,
			string(middleware.CodeInternal), "error actualizando producto", err.Error())
	}

	_ = pc.cache.Delete(context.Background(), cacheKeyProductos)
	return c.JSON(p)
}

// AdminDelete implementa DELETE /api/admin/productos/:id. Es un soft-delete
// (activo=false) para preservar FK desde pedido_items y el histórico.
// Idempotente: borrar dos veces seguidas no genera error.
func (pc *ProductoController) AdminDelete(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return middleware.RespondError(c, fiber.StatusBadRequest,
			string(middleware.CodeValidationError), "id inválido")
	}
	if err := pc.repo.SoftDeleteProducto(id); err != nil {
		if errors.Is(err, models.ErrProductoNoEncontrado) {
			return middleware.RespondError(c, fiber.StatusNotFound,
				string(middleware.CodeNotFound), "producto no encontrado")
		}
		return middleware.RespondError(c, fiber.StatusInternalServerError,
			string(middleware.CodeInternal), "error eliminando producto", err.Error())
	}

	_ = pc.cache.Delete(context.Background(), cacheKeyProductos)
	return c.SendStatus(fiber.StatusNoContent)
}

// parseIntDefault devuelve el valor parseado o fallback si el string es
// vacío o no es un entero válido. Centralizado para no repetir el patrón.
func parseIntDefault(s string, fallback int) int {
	if s == "" {
		return fallback
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return v
}