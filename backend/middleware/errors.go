// Package middleware contiene middlewares reutilizables del backend:
// seguridad (headers), rate limit, autenticación JWT y helpers de error.
//
// Forma de uso típica en un handler:
//
//	if err := middleware.RespondError(c, fiber.StatusBadRequest,
//	    "VALIDATION_ERROR", "datos inválidos"); err != nil {
//	    return err
//	}
package middleware

import (
	"os"

	"github.com/gofiber/fiber/v2"
)

// APIError es la estructura de error estándar de la API.
// Coincide con el schema components.schemas.APIError del openapi.yaml.
//
// Campos:
//
//	status  — código HTTP devuelto al cliente
//	code    — código de error legible por máquina (ver tabla de códigos)
//	message — mensaje legible por humanos (seguro para mostrar)
//	details — opcional, sólo presente cuando APP_ENV=dev
type APIError struct {
	Status  int         `json:"status"`
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// Códigos de error canónicos (alineados con openapi.yaml).
//
//	VALIDATION_ERROR   — payload mal formado o falla validator (400)
//	UNAUTHORIZED       — token ausente / inválido / expirado (401)
//	FORBIDDEN          — autenticado pero sin permisos (rol) (403)
//	NOT_FOUND          — recurso inexistente (404)
//	CONFLICT           — duplicado / estado incompatible (409)
//	RATE_LIMITED       — demasiadas peticiones (429)
//	INVALID_TRANSITION — cambio de estado ilegal en FSM (409)
//	STOCK_INSUFICIENTE — sin inventario suficiente para checkout (409)
//	INTERNAL           — error inesperado del servidor (5xx)
type ErrorCode string

const (
	CodeValidationError ErrorCode = "VALIDATION_ERROR"
	CodeUnauthorized    ErrorCode = "UNAUTHORIZED"
	CodeForbidden       ErrorCode = "FORBIDDEN"
	CodeNotFound        ErrorCode = "NOT_FOUND"
	CodeConflict        ErrorCode = "CONFLICT"
	CodeRateLimited     ErrorCode = "RATE_LIMITED"
	CodeInvalidTrans    ErrorCode = "INVALID_TRANSITION"
	CodeStockInsufic    ErrorCode = "STOCK_INSUFICIENTE"
	CodeInternal        ErrorCode = "INTERNAL"
)

// RespondError escribe una respuesta JSON uniforme de error en el contexto
// de Fiber y devuelve el error para que el llamador pueda cortocircuitar
// la cadena con `return ...`.
//
// Comportamiento del campo details:
//   - Si APP_ENV=dev y se pasa al menos un detalle, details[0] se incluye.
//   - En production (o APP_ENV no establecido) details siempre se omite,
//     para no filtrar internals al cliente.
//
// Ejemplo:
//
//	return middleware.RespondError(c, fiber.StatusBadRequest,
//	    middleware.CodeValidationError, "email requerido")
//
//	return middleware.RespondError(c, fiber.StatusInternalServerError,
//	    middleware.CodeInternal, "fallo inesperado", err.Error())
func RespondError(c *fiber.Ctx, status int, code string, message string, details ...interface{}) error {
	body := APIError{
		Status:  status,
		Code:    code,
		Message: message,
	}
	if isDevEnv() && len(details) > 0 {
		body.Details = details[0]
	}
	return c.Status(status).JSON(body)
}

// isDevEnv devuelve true cuando APP_ENV=dev (case-insensitive). Todo lo
// demás (production, vacío, otros) se trata como producción.
func isDevEnv() bool {
	return os.Getenv("APP_ENV") == "dev"
}
