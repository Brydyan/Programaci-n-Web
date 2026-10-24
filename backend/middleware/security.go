package middleware

import (
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// SecurityHeaders añade cabeceras de seguridad a TODAS las respuestas HTTP.
// Debe aplicarse con app.Use(...) antes de declarar rutas.
//
// Cabeceras siempre presentes:
//   - X-Content-Type-Options: nosniff
//   - X-Frame-Options: DENY
//   - Referrer-Policy: no-referrer
//   - Permissions-Policy: camera=(), microphone=(), geolocation=()
//   - Content-Security-Policy (dev/prod ajustada por ALLOWED_ORIGINS)
//
// Cabecera condicional:
//   - Strict-Transport-Security: sólo cuando APP_ENV=production.
//
// Política de CSP:
//   - En desarrollo (APP_ENV=dev) se permite el origen de Vite
//     (http://localhost:5173 y http://localhost:5174) y ws: en connect-src,
//     para que el frontend dev siga funcionando.
//   - En producción, connect-src queda limitado a 'self' y los orígenes
//     listados en ALLOWED_ORIGINS (CSV en env).
func SecurityHeaders() fiber.Handler {
	allowedOriginsCSV := strings.TrimSpace(os.Getenv("ALLOWED_ORIGINS"))
	connectSrc := buildConnectSrc(allowedOriginsCSV)

	return func(c *fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("Referrer-Policy", "no-referrer")
		c.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		if os.Getenv("APP_ENV") == "production" {
			c.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		}
		c.Set("Content-Security-Policy",
			"default-src 'self'; "+
				"img-src 'self' https://picsum.photos data:; "+
				"style-src 'self' 'unsafe-inline'; "+
				"script-src 'self'; "+
				"connect-src 'self' "+connectSrc+"; "+
				"frame-ancestors 'none'",
		)
		return c.Next()
	}
}

// buildConnectSrc construye el fragmento connect-src del CSP a partir
// del CSV ALLOWED_ORIGINS. En dev añade además los puertos típicos de Vite
// y la pareja ws://localhost:<puerto> para HMR sobre WebSockets.
func buildConnectSrc(allowedOriginsCSV string) string {
	parts := []string{"http://localhost:3000", "ws://localhost:3000"}

	if os.Getenv("APP_ENV") == "dev" {
		parts = append(parts,
			"http://localhost:5173",
			"ws://localhost:5173",
			"http://localhost:5174",
			"ws://localhost:5174",
		)
	}

	for _, raw := range strings.Split(allowedOriginsCSV, ",") {
		origin := strings.TrimSpace(raw)
		if origin == "" {
			continue
		}
		parts = append(parts, origin)
		// Variante WebSocket si el origen es http(s)
		if strings.HasPrefix(origin, "http://") {
			parts = append(parts, "ws://"+strings.TrimPrefix(origin, "http://"))
		} else if strings.HasPrefix(origin, "https://") {
			parts = append(parts, "wss://"+strings.TrimPrefix(origin, "https://"))
		}
	}

	return strings.Join(parts, " ")
}
