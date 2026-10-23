package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"

	"multicatalogo-backend/interfaces"
	"multicatalogo-backend/middleware"
	"multicatalogo-backend/models"
)

// ReportesController alimenta el dashboard del admin.
//
// Los cuatro endpoints son de sólo lectura y agregación, pero no son
// baratos: cada resumen son dos agregaciones y cada ranking un GROUP BY con
// JOIN sobre las tablas más anchas del esquema. El caché de 15 minutos no es
// una optimización prematura, es la diferencia entre un dashboard instantáneo
// y uno que castiga a Postgres cada vez que el admin refresca.
//
// Todos los endpoints son admin only (ver routes.go): los agregados de la
// organización no son información pública ni de un distribuidor.
type ReportesController struct {
	repo  interfaces.DBRepository
	cache interfaces.CacheService
}

func NewReportesController(repo interfaces.DBRepository, cache interfaces.CacheService) *ReportesController {
	return &ReportesController{repo: repo, cache: cache}
}

// ── Caché versionada de reportes ────────────────────────────────────────────
//
// Mismo problema que la red (ver networkController.go) y la misma solución:
// un checkout crea un pedido, y un cambio de estado puede cancelar otro.
// Ningún controller sabe qué reporte quedó obsoleto, así que en vez de
// purgar claves se versionan.

const (
	reportesVersionKey = "reportes:ver"
	reportesTTL        = 15 * time.Minute

	// layoutFecha es el formato que declara openapi.yaml (`format: date`).
	layoutFecha = "2006-01-02"
)

// versionReportes lee el contador de versión. Si Redis está caído devuelve 0:
// el caché no funciona de todos modos, así que la API responde siempre desde
// PostgreSQL (degradación silenciosa, sin error al cliente).
func versionReportes(ctx context.Context, cache interfaces.CacheService) int64 {
	if cache == nil {
		return 0
	}
	raw, err := cache.Get(ctx, reportesVersionKey)
	if err != nil {
		return 0
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0
	}
	return v
}

// InvalidarReportes sube la versión del caché. Se llama tras cualquier
// escritura que altere los agregados: un checkout (pedido nuevo + comisiones)
// o un cambio de estado de pedido (sobre todo cancelar, que resta venta).
//
// Es función de paquete y no método porque los tres puntos que la necesitan
// —checkout, cambio de estado y el propio controller— no comparten receiver.
func InvalidarReportes(ctx context.Context, cache interfaces.CacheService) {
	if cache == nil {
		return
	}
	if _, err := cache.Increment(ctx, reportesVersionKey); err != nil {
		// Best-effort: si Redis falla, la API sigue correcta, sólo pierde el
		// caché y puede servir datos hasta 15 minutos old.
		_ = err
	}
}

// responderCache es el camino común de los cuatro endpoints: caché-aside
// sobre el JSON ya serializado.
//
// Es función de paquete y no método porque Go 1.26 no permite métodos
// genéricos sobre un receiver; el cache va como parámetro.
//
// La clave incluye TODOS los parámetros que cambian el resultado (periodo,
// limit). Sin eso, un `?limit=50` devolvería los datos cacheados de un
// `?limit=10` anterior, que es el tipo de bug que sólo aparece en producción
// y cuando alguien compara dos pestañas del dashboard.
func responderCache[T any](
	c *fiber.Ctx,
	cache interfaces.CacheService,
	scope string,
	cargar func() (T, error),
	mensajeError string,
) error {
	ctx := context.Background()
	key := fmt.Sprintf("reportes:v%d:%s", versionReportes(ctx, cache), scope)

	if cache != nil {
		if cached, err := cache.Get(ctx, key); err == nil {
			var out T
			if jsonErr := json.Unmarshal([]byte(cached), &out); jsonErr == nil {
				return c.Status(fiber.StatusOK).JSON(out)
			}
			// JSON corrupto: se ignora y se recalcula en vez de fallar.
		} else if err != redis.Nil {
			// Redis caído: seguimos a PostgreSQL sin quejarnos al cliente.
			_ = err
		}
	}

	datos, err := cargar()
	if err != nil {
		return middleware.RespondError(c,
			fiber.StatusInternalServerError,
			string(middleware.CodeInternal),
			mensajeError,
			err.Error(),
		)
	}

	if cache != nil {
		if raw, jsonErr := json.Marshal(datos); jsonErr == nil {
			//nolint:errcheck // best-effort: si falla, sólo se pierde caché
			cache.Set(ctx, key, raw, reportesTTL)
		}
	}
	return c.Status(fiber.StatusOK).JSON(datos)
}

// ── Handlers ────────────────────────────────────────────────────────────────

// GetResumen devuelve los KPIs del periodo.
//
//	GET /api/reportes/resumen?desde=2026-01-01&hasta=2026-12-31
func (rc *ReportesController) GetResumen(c *fiber.Ctx) error {
	desde, err := parseFecha(c.Query("desde"))
	if err != nil {
		return middleware.RespondError(c,
			fiber.StatusBadRequest,
			string(middleware.CodeValidationError),
			"el parámetro 'desde' no es una fecha válida (AAAA-MM-DD)",
		)
	}
	hasta, err := parseFechaHasta(c.Query("hasta"))
	if err != nil {
		return middleware.RespondError(c,
			fiber.StatusBadRequest,
			string(middleware.CodeValidationError),
			"el parámetro 'hasta' no es una fecha válida (AAAA-MM-DD)",
		)
	}
	if desde != nil && hasta != nil && hasta.Before(*desde) {
		return middleware.RespondError(c,
			fiber.StatusBadRequest,
			string(middleware.CodeValidationError),
			"'hasta' es anterior a 'desde'",
		)
	}

	filtro := interfaces.ReporteFiltro{Desde: desde, Hasta: hasta}
	scope := fmt.Sprintf("resumen:%s:%s", fechaOpcional(desde), fechaOpcional(hasta))

	return responderCache(c, rc.cache, scope,
		func() (*models.ReporteResumen, error) { return rc.repo.GetReporteResumen(filtro) },
		"error calculando el resumen de ventas",
	)
}

// GetTopProductos devuelve el ranking de productos más vendidos.
//
//	GET /api/reportes/top-productos?limit=10
func (rc *ReportesController) GetTopProductos(c *fiber.Ctx) error {
	limit, err := parseLimit(c.Query("limit"))
	if err != nil {
		return middleware.RespondError(c,
			fiber.StatusBadRequest,
			string(middleware.CodeValidationError),
			"el parámetro 'limit' debe ser un número entre 1 y 50",
		)
	}
	scope := fmt.Sprintf("top-productos:%d", limit)

	return responderCache(c, rc.cache, scope,
		func() ([]models.TopProducto, error) { return rc.repo.GetTopProductos(limit) },
		"error consultando los productos más vendidos",
	)
}

// GetTopReferidos devuelve el ranking de distribuidores.
//
//	GET /api/reportes/top-referidos?limit=10
func (rc *ReportesController) GetTopReferidos(c *fiber.Ctx) error {
	limit, err := parseLimit(c.Query("limit"))
	if err != nil {
		return middleware.RespondError(c,
			fiber.StatusBadRequest,
			string(middleware.CodeValidationError),
			"el parámetro 'limit' debe ser un número entre 1 y 50",
		)
	}
	scope := fmt.Sprintf("top-referidos:%d", limit)

	return responderCache(c, rc.cache, scope,
		func() ([]models.TopReferido, error) { return rc.repo.GetTopReferidos(limit) },
		"error consultando los distribuidores con más ventas",
	)
}

// GetComisionesPendientes devuelve el ledger pendiente de liquidar.
//
//	GET /api/reportes/comisiones-pendientes?limit=50
func (rc *ReportesController) GetComisionesPendientes(c *fiber.Ctx) error {
	limit, err := parseLimit(c.Query("limit"))
	if err != nil {
		return middleware.RespondError(c,
			fiber.StatusBadRequest,
			string(middleware.CodeValidationError),
			"el parámetro 'limit' debe ser un número entre 1 y 50",
		)
	}
	scope := fmt.Sprintf("comisiones-pendientes:%d", limit)

	return responderCache(c, rc.cache, scope,
		func() ([]models.Comision, error) { return rc.repo.GetComisionesPendientes(limit) },
		"error consultando las comisiones pendientes",
	)
}

// ── Parseo de query-params ──────────────────────────────────────────────────

// parseFecha lee un parámetro de fecha `AAAA-MM-DD`.
//
// `hasta` es INCLUSIVO para el usuario: pedir hasta el 31 debe incluir el 31,
// así que se devuelve el día siguiente como cota superior EXCLUSIVA
// (`< 2027-01-01`). La conversión vive aquí y no en el repository para que
// las tres capas que construyan un ReporteFiltro compartan la misma
// semántica.
func parseFecha(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	t, err := time.Parse(layoutFecha, raw)
	if err != nil {
		return nil, fmt.Errorf("fecha inválida %q: %w", raw, err)
	}
	// time.Parse con layout de sólo fecha devuelve UTC medianoche. Se pasa a
	// UTC explícito para que la comparación con created_at (TIMESTAMPTZ) no
	// dependa de la zona horaria local del servidor.
	return &t, nil
}

// parseFechaHasta es igual que parseFecha pero devuelve el día SIGUIENTE, para
// usar como cota `<` y así incluir el día pedido completo.
func parseFechaHasta(raw string) (*time.Time, error) {
	t, err := parseFecha(raw)
	if err != nil || t == nil {
		return t, err
	}
	siguiente := t.AddDate(0, 0, 1)
	return &siguiente, nil
}

// parseLimit valida `?limit=`. Default 10, rango 1..50, según openapi.yaml.
func parseLimit(raw string) (int, error) {
	if raw == "" {
		return 10, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("limit inválido %q: %w", raw, err)
	}
	if v < 1 || v > 50 {
		return 0, fmt.Errorf("limit %d fuera de rango [1,50]", v)
	}
	return v, nil
}

// fechaOpcional renderiza una fecha para construir la clave de caché. El nil
// se distingue del valor cero con "-", para que "sin filtro" y "01-01-0001"
// no compartan clave.
func fechaOpcional(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return t.Format(layoutFecha)
}
