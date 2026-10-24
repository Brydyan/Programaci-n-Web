package middleware

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// LocalUserKey es la clave usada en c.Locals(...) para guardar la JWTClaims
// validada. Compatible con cualquier handler downstream que la lea.
const LocalUserKey = "user"

// JWTSecretKey es la clave usada en c.Locals(...) para guardar el []byte
// con la clave de firma. Permite a los handlers emitir nuevos tokens
// (ej. Login) sin re-leer el entorno.
const JWTSecretKey = "jwt_secret"

// JWTClaims es el payload estándar de los tokens emitidos por la API.
//
// Campos:
//   - Sub  — id numérico del usuario (subject)
//   - Email — correo del usuario
//   - Rol  — "admin" o "cliente"
//   - Iat  — issued at (unix seconds)
//   - Exp  — expiration (unix seconds)
//
// Se serializa con camelCase (sub/email/rol/iat/exp) para alinearse con el
// openapi.yaml (bearerAuth + LoginResponse).
type JWTClaims struct {
	Sub   int    `json:"sub"`
	Email string `json:"email"`
	Rol   string `json:"rol"`
	Iat   int64  `json:"iat"`
	Exp   int64  `json:"exp"`
}

// RegisteredClaims expone los campos estándar (iat/exp) que exige jwt/v5.
func (c JWTClaims) RegisteredClaims() *jwt.RegisteredClaims {
	now := time.Now()
	if c.Iat == 0 {
		c.Iat = now.Unix()
	}
	return &jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(time.Unix(c.Iat, 0)),
		ExpiresAt: jwt.NewNumericDate(time.Unix(c.Exp, 0)),
	}
}

// SignJWT firma claims con HS256 y devuelve el token compacto listo para
// enviar al cliente. Devuelve error si la librería no logra serializar.
func SignJWT(secret []byte, claims JWTClaims) (string, error) {
	if claims.Iat == 0 {
		claims.Iat = time.Now().Unix()
	}
	payload := jwt.MapClaims{
		"sub":   claims.Sub,
		"email": claims.Email,
		"rol":   claims.Rol,
		"iat":   claims.Iat,
		"exp":   claims.Exp,
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	return tok.SignedString(secret)
}

// ParseJWT valida la firma y los claims estándar (exp) del token.
// Devuelve las JWTClaims reconstituidas o un error.
func ParseJWT(secret []byte, tokenString string) (JWTClaims, error) {
	var out JWTClaims
	parsed, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("método de firma inesperado")
		}
		return secret, nil
	})
	if err != nil {
		return out, err
	}
	if !parsed.Valid {
		return out, errors.New("token inválido")
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return out, errors.New("claims de tipo inesperado")
	}
	if subF, ok := claims["sub"].(float64); ok {
		out.Sub = int(subF)
	}
	if s, ok := claims["email"].(string); ok {
		out.Email = s
	}
	if s, ok := claims["rol"].(string); ok {
		out.Rol = s
	}
	if f, ok := claims["iat"].(float64); ok {
		out.Iat = int64(f)
	}
	if f, ok := claims["exp"].(float64); ok {
		out.Exp = int64(f)
	}
	return out, nil
}

// RequireAuth valida Authorization: Bearer <token> y, si es correcto,
// guarda claims en c.Locals(LocalUserKey). Devuelve 401 con el cuerpo
// uniforme de APIError en cualquier fallo (token ausente, mal formado,
// firma inválida, expirado).
//
// Uso en rutas:
//
//	api.Get("/perfil",
//	    middleware.RequireAuth(jwtSecret),
//	    handler,
//	)
func RequireAuth(secret []byte) fiber.Handler {
	return func(c *fiber.Ctx) error {
		header := c.Get("Authorization")
		if header == "" {
			return RespondError(c, fiber.StatusUnauthorized,
				string(CodeUnauthorized), "token de autenticación requerido")
		}
		// Aceptar "Bearer X" o sólo "X"
		const prefix = "Bearer "
		token := header
		if len(header) > len(prefix) && header[:len(prefix)] == prefix {
			token = header[len(prefix):]
		}
		if token == "" {
			return RespondError(c, fiber.StatusUnauthorized,
				string(CodeUnauthorized), "token vacío")
		}
		claims, err := ParseJWT(secret, token)
		if err != nil {
			return RespondError(c, fiber.StatusUnauthorized,
				string(CodeUnauthorized), "token inválido o expirado",
				err.Error())
		}
		c.Locals(LocalUserKey, claims)
		return c.Next()
	}
}

// RequireAdmin asume que RequireAuth ya corrió en la cadena (no lo
// invoca de nuevo). Lee claims de c.Locals y rechaza con 403 FORBIDDEN
// si rol != "admin".
//
// Uso típico en apilado:
//
//	api.Get("/admin/pedidos",
//	    middleware.RequireAuth(jwtSecret),
//	    middleware.RequireAdmin(),
//	    handler,
//	)
func RequireAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		v := c.Locals(LocalUserKey)
		claims, ok := v.(JWTClaims)
		if !ok {
			return RespondError(c, fiber.StatusUnauthorized,
				string(CodeUnauthorized), "autenticación requerida")
		}
		if claims.Rol != "admin" {
			return RespondError(c, fiber.StatusForbidden,
				string(CodeForbidden), "se requiere rol admin")
		}
		return c.Next()
	}
}
