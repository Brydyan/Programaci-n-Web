package controllers

import (
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"multicatalogo-backend/domain"
	"multicatalogo-backend/internal/dto"
	"multicatalogo-backend/interfaces"
	"multicatalogo-backend/middleware"
	"multicatalogo-backend/models"
)

type AuthController struct {
	repo interfaces.DBRepository
}

func NewAuthController(repo interfaces.DBRepository) *AuthController {
	return &AuthController{repo: repo}
}

// LoginResponse es la forma canónica de la respuesta de POST /api/login,
// alineada con openapi.yaml — components.schemas.LoginResponse.
type LoginResponse struct {
	Token string `json:"token"`
	Email string `json:"email"`
	Rol   string `json:"rol"`
}

// Login valida el cuerpo con el DTO + validator, autentica contra la DB
// (bcrypt + flags activo) y emite un JWT firmado con HS256.
//
// Mapeo de errores del repository a HTTP:
//
//	ErrCredencialesInvalidas → 401 UNAUTHORIZED
//	ErrCuentaDesactivada     → 403 FORBIDDEN
//
// La clave de firma se obtiene de c.Locals(middleware.JWTSecretKey).
// En main.go se setea vía app.Use(...) antes de cualquier ruta.
func (ac *AuthController) Login(c *fiber.Ctx) error {
	var req dto.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return middleware.RespondError(c,
			fiber.StatusBadRequest,
			string(middleware.CodeValidationError),
			"cuerpo inválido",
		)
	}
	if err := dto.Validate.Struct(req); err != nil {
		return middleware.RespondError(c,
			fiber.StatusBadRequest,
			string(middleware.CodeValidationError),
			"datos de login inválidos",
			err.Error(),
		)
	}

	usuario, err := ac.repo.Login(req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrCuentaDesactivada):
			return middleware.RespondError(c,
				fiber.StatusForbidden,
				string(middleware.CodeForbidden),
				"cuenta desactivada",
			)
		case errors.Is(err, models.ErrCredencialesInvalidas):
			return middleware.RespondError(c,
				fiber.StatusUnauthorized,
				string(middleware.CodeUnauthorized),
				"credenciales incorrectas",
			)
		default:
			return middleware.RespondError(c,
				fiber.StatusInternalServerError,
				string(middleware.CodeInternal),
				"error interno en login",
				err.Error(),
			)
		}
	}

	secretAny := c.Locals(middleware.JWTSecretKey)
	secret, _ := secretAny.([]byte)

	claims := middleware.JWTClaims{
		Sub:   usuario.ID,
		Email: usuario.Email,
		Rol:   usuario.Rol,
		Iat:   time.Now().Unix(),
		Exp:   time.Now().Add(2 * time.Hour).Unix(),
	}
	token, err := middleware.SignJWT(secret, claims)
	if err != nil {
		return middleware.RespondError(c,
			fiber.StatusInternalServerError,
			string(middleware.CodeInternal),
			"no se pudo emitir el token",
			err.Error(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(LoginResponse{
		Token: token,
		Email: usuario.Email,
		Rol:   claims.Rol,
	})
}

// Register crea un usuario nuevo en la DB (con bcrypt + código de
// referido único). Validaciones:
//
//   - DTO validator (nombre/email/password/ref formato).
//   - Si trae `ref`, debe pasar domain.CodigoEsValido (alfabeto
//     restringido — el validador `alphanum` del DTO acepta 0/1/I).
//
// Errores mapeados:
//
//	ErrEmailDuplicado    → 409 CONFLICT
//	ErrCodigoRefInvalido → 400 VALIDATION_ERROR
//
// On success → 201 con {id, email, codigo_ref}.
//
// NO emite JWT: el contrato pide que el cliente haga login después del
// registro para autenticarse (F1). F4 podría cambiar a "auto-login".
func (ac *AuthController) Register(c *fiber.Ctx) error {
	var req dto.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return middleware.RespondError(c,
			fiber.StatusBadRequest,
			string(middleware.CodeValidationError),
			"cuerpo inválido",
		)
	}
	if err := dto.Validate.Struct(req); err != nil {
		return middleware.RespondError(c,
			fiber.StatusBadRequest,
			string(middleware.CodeValidationError),
			"datos de registro inválidos",
			err.Error(),
		)
	}

	// Validación extra del código de referido: el DTO sólo exige alphanum,
	// pero el alfabeto del sistema excluye 0/1/I para legibilidad.
	if req.Ref != "" && !domain.CodigoEsValido(strings.ToUpper(req.Ref)) {
		return middleware.RespondError(c,
			fiber.StatusBadRequest,
			string(middleware.CodeValidationError),
			"código de referido inválido",
		)
	}

	// Adaptamos al modelo de repository (mismos campos, distinto paquete).
	modelReq := models.RegisterRequest{
		Nombre:   req.Nombre,
		Email:    req.Email,
		Password: req.Password,
		Ref:      strings.ToUpper(req.Ref),
	}

	result, err := ac.repo.Registrar(modelReq)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrEmailDuplicado):
			return middleware.RespondError(c,
				fiber.StatusConflict,
				string(middleware.CodeConflict),
				"email ya registrado",
			)
		case errors.Is(err, models.ErrCodigoRefInvalido):
			return middleware.RespondError(c,
				fiber.StatusBadRequest,
				string(middleware.CodeValidationError),
				"código de referido inválido",
			)
		default:
			return middleware.RespondError(c,
				fiber.StatusInternalServerError,
				string(middleware.CodeInternal),
				"error interno en registro",
				err.Error(),
			)
		}
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id":         result.UsuarioID,
		"email":      result.Email,
		"codigo_ref": result.CodigoRef,
	})
}