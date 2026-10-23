package controllers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"multicatalogo-backend/interfaces"
	"multicatalogo-backend/middleware"
	"multicatalogo-backend/models"
)

// ComisionesController expone el ledger de comisiones.
//
// El ledger es la fuente de verdad del dinero del MLM: cada fila dice quién
// cobró, cuánto, en qué nivel y por qué pedido. `referidos.ventas` es sólo
// un contador de lectura rápida derivado de él.
type ComisionesController struct {
	repo interfaces.DBRepository
}

func NewComisionesController(repo interfaces.DBRepository) *ComisionesController {
	return &ComisionesController{repo: repo}
}

// estadosValidos refleja el CHECK de la tabla comisiones. Se valida aquí y
// no en SQL para poder devolver 400 con un mensaje claro en vez de un 500 por
// violación de restricción.
var estadosValidos = map[string]bool{
	"pendiente": true,
	"aprobada":  true,
	"pagada":    true,
	"revertida": true,
}

// GetMias devuelve el ledger del usuario del token.
//
// No acepta un referido_id por query: la identidad se toma del claim `sub`.
// Si el usuario acepta un id por parámetro sería trivial leer las comisiones
// de otro distribuidor.
func (cc *ComisionesController) GetMias(c *fiber.Ctx) error {
	claimsAny := c.Locals(middleware.LocalUserKey)
	claims, ok := claimsAny.(middleware.JWTClaims)
	if !ok {
		return middleware.RespondError(c,
			fiber.StatusUnauthorized,
			string(middleware.CodeUnauthorized),
			"token sin identidad de usuario",
		)
	}

	estado, err := estadoQuery(c)
	if err != nil {
		return middleware.RespondError(c,
			fiber.StatusBadRequest,
			string(middleware.CodeValidationError),
			"filtro de estado inválido",
			err.Error(),
		)
	}

	// El ledger cuelga del nodo de red, no del usuario: se traduce el
	// usuario a su referido. Si no tiene nodo, no hay comisiones suyas.
	referidoID, err := cc.referidoDeUsuario(claims.Sub)
	if err != nil {
		return middleware.RespondError(c,
			fiber.StatusInternalServerError,
			string(middleware.CodeInternal),
			"error consultando el nodo de red del usuario",
			err.Error(),
		)
	}
	if referidoID == nil {
		return c.Status(fiber.StatusOK).JSON(models.ComisionResumen{
			Data: []models.Comision{}, Page: 1, Limit: 20, Total: 0,
			Totales: []models.ComisionTotal{},
		})
	}

	return cc.responder(c, interfaces.ComisionFiltro{
		ReferidoID: referidoID,
		Estado:     estado,
		Page:       queryInt(c, "page"),
		Limit:      queryInt(c, "limit"),
	})
}

// GetAdmin devuelve el ledger completo con filtro por estado. Admin only.
func (cc *ComisionesController) GetAdmin(c *fiber.Ctx) error {
	estado, err := estadoQuery(c)
	if err != nil {
		return middleware.RespondError(c,
			fiber.StatusBadRequest,
			string(middleware.CodeValidationError),
			"filtro de estado inválido",
			err.Error(),
		)
	}

	var referidoID *int
	if raw := c.Query("referido_id"); raw != "" {
		id, convErr := strconv.Atoi(raw)
		if convErr != nil {
			return middleware.RespondError(c,
				fiber.StatusBadRequest,
				string(middleware.CodeValidationError),
				"referido_id debe ser un número",
			)
		}
		referidoID = &id
	}

	return cc.responder(c, interfaces.ComisionFiltro{
		ReferidoID: referidoID,
		Estado:     estado,
		Page:       queryInt(c, "page"),
		Limit:      queryInt(c, "limit"),
	})
}

// referidoDeUsuario traduce usuario → nodo de red. Nil significa "todavía no
// tiene nodo", que es un estado válido (alta sin código de referido).
//
// Devuelve el error CRUDO a propósito, no una respuesta ya escrita.
// middleware.RespondError devuelve nil cuando la escritura del JSON tiene
// éxito (sólo falla si la serialización revienta), así que usarlo como valor
// de retorno hacía que `if err != nil` nunca disparase y el handler cayera en
// la rama de éxito: un fallo de base de datos respondía 200 con lista vacía en
// vez de 500. Quien llama decide cómo mapear el error.
func (cc *ComisionesController) referidoDeUsuario(usuarioID int) (*int, error) {
	return cc.repo.GetReferidoIDByUsuario(usuarioID)
}

// responder ejecuta la consulta paginada y añade los totales por estado.
func (cc *ComisionesController) responder(c *fiber.Ctx, filtro interfaces.ComisionFiltro) error {
	data, total, err := cc.repo.GetComisiones(filtro)
	if err != nil {
		return middleware.RespondError(c,
			fiber.StatusInternalServerError,
			string(middleware.CodeInternal),
			"error consultando comisiones",
			err.Error(),
		)
	}
	totales, err := cc.repo.GetTotalesComisiones(filtro.ReferidoID)
	if err != nil {
		return middleware.RespondError(c,
			fiber.StatusInternalServerError,
			string(middleware.CodeInternal),
			"error consultando totales de comisiones",
			err.Error(),
		)
	}

	page, limit := filtro.Page, filtro.Limit
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	return c.Status(fiber.StatusOK).JSON(models.ComisionResumen{
		Data: data, Page: page, Limit: limit, Total: total, Totales: totales,
	})
}

// estadoQuery lee y valida ?estado=.
func estadoQuery(c *fiber.Ctx) (string, error) {
	estado := c.Query("estado")
	if estado == "" {
		return "", nil
	}
	if !estadosValidos[estado] {
		return "", &estadoInvalidoError{estado: estado}
	}
	return estado, nil
}

type estadoInvalidoError struct{ estado string }

func (e *estadoInvalidoError) Error() string {
	return "estado '" + e.estado + "' no es válido (pendiente, aprobada, pagada, revertida)"
}

// queryInt lee un query-param entero; si no es número devuelve 0 (que el
// repository normaliza a sus defaults).
func queryInt(c *fiber.Ctx, key string) int {
	raw := c.Query(key)
	if raw == "" {
		return 0
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return v
}
