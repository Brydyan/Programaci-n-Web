package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

// RateLimit es una factoría sobre el middleware `limiter` de Fiber.
//
// Por defecto limita por IP. Devuelve 429 cuando el cliente supera
// `max` peticiones dentro de la ventana `window`.
//
// Uso:
//
//	// antes de una ruta concreta, no global:
//	app.Post("/api/login",
//	    middleware.RateLimit(10, time.Minute),
//	    ctrl.Auth.Login,
//	)
//
// NOTA: aplicar RateLimit a una ruta no registrada no causa error — Fiber
// simplemente nunca la invoca.
func RateLimit(max int, window time.Duration) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        max,
		Expiration: window,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return RespondError(c,
				fiber.StatusTooManyRequests,
				string(CodeRateLimited),
				"demasiadas peticiones, intente más tarde",
			)
		},
	})
}
