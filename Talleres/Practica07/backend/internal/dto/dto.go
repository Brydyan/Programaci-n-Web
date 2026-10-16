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
