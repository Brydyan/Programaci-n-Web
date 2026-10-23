package controllers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"multicatalogo-backend/interfaces"
	"multicatalogo-backend/internal/dto"
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

	var req dto.CheckoutRequest
	if err := c.BodyParser(&req); err != nil {
		return middleware.RespondError(c, fiber.StatusBadRequest,
			string(middleware.CodeValidationError), "Cuerpo de petición inválido")
	}
	// Validación de forma contra components.schemas.CheckoutRequest de
	// openapi.yaml: direccion/ciudad/telefono/metodo_pago obligatorios,
	// items con al menos una línea. Antes se comprobaban a mano sólo el
	// carrito vacío y el método de pago, y ni dirección ni ciudad ni teléfono
	// se exigían — pese a que el contrato sí los exige.
	if err := dto.Validate.Struct(&req); err != nil {
		return middleware.RespondError(c, fiber.StatusBadRequest,
			string(middleware.CodeValidationError), "Datos de envío inválidos",
			detallesValidacion(err))
	}

	// Identidad: SIEMPRE del token. El DTO declara EmailUsuario con `json:"-"`,
	// así que un email en el body se descarta en el parseo.
	req.EmailUsuario = claims.Email

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

	// Post-COMMIT: el contador `referidos.ventas` y el caché de la red.
	//
	// El ledger de comisiones YA se escribió dentro de la transacción del
	// pedido (Sprint 2): aquí no queda nada que pueda perder dinero, sólo un
	// contador de lectura rápida y una caché que invalidar.
	go cc.actualizarContadores(req.EmailUsuario, detalle.Total)

	return c.Status(fiber.StatusCreated).JSON(detalle)
}

// detallesValidacion traduce los errores de go-playground/validator a una
// lista de mensajes legibles para el cliente.
//
// `fe.Field()` devuelve el nombre del campo Go ("Ciudad"), no el tag JSON, así
// que `camelCase` sólo actúa cuando el campo está en snake_case. Se mantiene
// porque el DTO mezcla ambas convenciones y así el mensaje es consistente
// pase lo que pase.
func detallesValidacion(err error) []string {
	var verrs validator.ValidationErrors
	if !errors.As(err, &verrs) {
		return nil
	}
	detalles := make([]string, 0, len(verrs))
	for _, fe := range verrs {
		msg := fmt.Sprintf("%s %s", campoLegible(fe.Field()), verbijo(fe.Tag()))
		// Param viene vacío en reglas como `required`; "ciudad es requerido ()"
		// se lee como un error de plantilla mal resuelta.
		if fe.Param() != "" {
			msg += " (" + fe.Param() + ")"
		}
		detalles = append(detalles, msg)
	}
	return detalles
}

func campoLegible(campo string) string {
	partes := strings.Split(campo, ".")
	nombre := partes[len(partes)-1]
	return camelCase(nombre)
}

// verbijo traduce el nombre de la regla a un verbo legible. El cliente sólo
// muestra esto; la decisión sigue siendo del validator.
func verbijo(tag string) string {
	switch tag {
	case "required":
		return "es requerido"
	case "min":
		return "debe tener al menos"
	case "max":
		return "no puede superar"
	case "gt":
		return "debe ser mayor que"
	case "gte":
		return "no puede ser menor que"
	case "email":
		return "no tiene formato de email válido"
	default:
		return "es inválido"
	}
}

// camelCase convierte snake_case en camelCase sin dependencias externas.
func camelCase(s string) string {
	var b strings.Builder
	upperNext := false
	for _, r := range s {
		if r == '_' {
			upperNext = true
			continue
		}
		if upperNext {
			b.WriteString(strings.ToUpper(string(r)))
			upperNext = false
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

//  1. Suma la venta al contador `ventas` de los nodos del upline. Es cache de
//     lectura; si falla, el ledger sigue siendo correcto.
//  2. Sube la versión del caché de la red: el checkout acredita a hasta 3
//     nodos y sus árboles cacheados quedaron obsoletos. Versionar es más
//     simple que intentar borrar claves que no conocemos.
func (cc *CheckoutController) actualizarContadores(emailUsuario string, total float64) {
	if err := cc.repo.ActualizarVentasUpline(emailUsuario, total); err != nil {
		log.Printf("[Checkout] contador de ventas no actualizado para %s: %v", emailUsuario, err)
	}
	if cc.cache != nil {
		if _, err := cc.cache.Increment(context.Background(), redVersionKey); err != nil {
			log.Printf("[Checkout] caché de red no invalidado: %v", err)
		}
	}
	// El checkout crea un pedido y su ledger de comisiones: los cuatro
	// agregados del dashboard quedan obsoletos en el mismo instante.
	InvalidarReportes(context.Background(), cc.cache)
}
