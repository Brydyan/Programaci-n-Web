// Package dto centraliza los Data Transfer Objects de request del backend.
// Mantener los tags de validación aquí evita que los handlers se acoplen a
// la librería validator.
package dto

import (
	"reflect"

	"github.com/go-playground/validator/v10"
)

// Validate es el singleton del validador. Está inicializado en init()
// para que cualquier paquete lo use sin bootstrap adicional.
//
// Uso típico en un handler:
//
//	if err := c.BodyParser(&dto.LoginRequest{}); err != nil { ... }
//	err := dto.Validate.Struct(req)  // req es el puntero parseado
//
// También existe el método Validate() embebido en cada DTO:
//
//	if err := req.Validate(); err != nil { ... }
var Validate *validator.Validate

func init() {
	Validate = validator.New(validator.WithRequiredStructEnabled())
}

// Validator es un embed que expone el método Validate() sobre cualquier DTO.
// Para usarlo embebe anonymous en tu struct:
//
//	type LoginRequest struct {
//	    Validator
//	    Email    string `validate:"required,email"`
//	    Password string `validate:"required,min=8"`
//	}
//
//	y luego: req.Validate() == dto.Validate.Struct(req)
type Validator struct{}

func (Validator) Validate(target interface{}) error {
	return Validate.Struct(target)
}

// ─────────────────────────── Login ───────────────────────────

// LoginRequest es el cuerpo de POST /api/login.
// Alineado con openapi.yaml — components.schemas.LoginRequest.
type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=8,max=128"`
}

// ─────────────────────────── Register ───────────────────────────

// RegisterRequest es el cuerpo de POST /api/register.
// Ref es opcional y debe ser un código alfanumérico de 2-8 caracteres cuando
// se proporciona (códigos de referido).
type RegisterRequest struct {
	Nombre   string `json:"nombre"   validate:"required,min=2,max=120"`
	Email    string `json:"email"    validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=8,max=128"`
	Ref      string `json:"ref"      validate:"omitempty,alphanum,min=2,max=8"`
}

// ─────────────────────────── Producto ───────────────────────────

// ProductoCreate es el cuerpo de POST /api/admin/productos.
type ProductoCreate struct {
	Nombre    string  `json:"nombre"             validate:"required,min=2,max=200"`
	Precio    float64 `json:"precio"             validate:"required,gt=0"`
	Categoria string  `json:"categoria"          validate:"required,min=2,max=80"`
	Stock     int     `json:"stock"              validate:"gte=0"`
	Img       string  `json:"img,omitempty"      validate:"omitempty,url"`
}

// ProductoUpdate es el cuerpo de PUT /api/admin/productos/{id}.
// Todos los campos son opcionales (PUT parcial). Stock y Activo son
// punteros para distinguir "no enviado" de "enviado con valor cero".
type ProductoUpdate struct {
	Nombre    string  `json:"nombre"             validate:"omitempty,min=2,max=200"`
	Precio    float64 `json:"precio"             validate:"omitempty,gt=0"`
	Categoria string  `json:"categoria"          validate:"omitempty,max=80"`
	Stock     *int    `json:"stock"              validate:"omitempty,gte=0"`
	Img       string  `json:"img,omitempty"      validate:"omitempty,url"`
	Activo    *bool   `json:"activo"             validate:"omitempty"`
}

// ─────────────────────────── Pedido (F3) ───────────────────────────

// PedidoEstadoUpdate es el cuerpo de PATCH /api/admin/pedidos/:id/estado.
// Sólo acepta los 5 estados válidos del CHECK del schema (alineado con
// domain.EstadoPedido). El validator `oneof` rechaza cualquier valor
// fuera de la lista antes de llegar al repository, que a su vez valida
// la transición con la FSM (domain.EsTransicionValida).
type PedidoEstadoUpdate struct {
	Estado string `json:"estado" validate:"required,oneof=pendiente preparando enviado entregado cancelado"`
}

// ─────────────────────────── Checkout ────────────────────────────────────────

// ItemCarrito es una línea del carrito tal como la manda el cliente.
//
// El precio NO se valida contra el catálogo a propósito: es un dato no
// confiable y el repository lo ignora, recalculando el total desde la tabla
// `productos`. Validarlo aquí sólo daría una falsa sensación de seguridad.
type ItemCarrito struct {
	ID       int     `json:"id"       validate:"required,gt=0"`
	Nombre   string  `json:"nombre"   validate:"required,max=200"`
	Precio   float64 `json:"precio"   validate:"gte=0"`
	Cantidad int     `json:"cantidad" validate:"required,gt=0"`
}

// CheckoutRequest es el cuerpo de POST /api/checkout.
// Refleja components.schemas.CheckoutRequest de openapi.yaml, incluidos sus
// `required` y sus longitudes.
//
// La identidad no viaja en el body: EmailUsuario lo inyecta el controller desde
// el JWT (RequireAuth corre antes). Sin `json:"-"` el cliente podría pedir un
// pedido a nombre de otro simplemente poniendo su email en el JSON.
type CheckoutRequest struct {
	// EmailUsuario lo rellena el controller desde el claim `sub`/`email`.
	// `-` lo hace inmutable desde el exterior.
	EmailUsuario string `json:"-"`

	Nombre     string        `json:"nombre"      validate:"omitempty,max=120"`
	Direccion  string        `json:"direccion"   validate:"required,min=5,max=300"`
	Ciudad     string        `json:"ciudad"      validate:"required,min=2,max=120"`
	Telefono   string        `json:"telefono"    validate:"required,min=6,max=30"`
	MetodoPago string        `json:"metodo_pago" validate:"required,min=2,max=60"`
	Items      []ItemCarrito `json:"items"       validate:"required,min=1,dive"`
}

// Validate es un helper de cortesía que usa `reflect` para auto-detectar
// si el target es un puntero, requerido por validator.Struct.
//
// Uso:
//
//	if err := dto.ValidateStruct(&req); err != nil { ... }
func ValidateStruct(target interface{}) error {
	v := reflect.ValueOf(target)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return validator.New().Struct(target)
	}
	return Validate.Struct(target)
}
