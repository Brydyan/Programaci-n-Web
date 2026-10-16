package routes

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"multicatalogo-backend/controllers"
	"multicatalogo-backend/middleware"
)

type Controllers struct {
	Auth     *controllers.AuthController
	Producto *controllers.ProductoController
	Network  *controllers.NetworkController
	Checkout *controllers.CheckoutController
}

// SetupRoutes registra las rutas del backend.
//
// Parametrización para S1:
//   - jwtSecret: clave HMAC usada por Login para firmar tokens y por
//     RequireAuth (F1+) para validarlos. Se inyecta también como
//     c.Locals(middleware.JWTSecretKey) en cada request para que
//     AuthController.Login pueda emitir tokens sin re-leer el entorno.
//
// IMPORTANTE para sprints futuros (F1–F7):
//   - Las rutas NUEVAS admin/privadas deben protegerse con:
//
//     app.Get("/api/admin/...",
//         middleware.RequireAuth(jwtSecret),
//         middleware.RequireAdmin(),
//         handler,
//     )
//
//   - NO añadimos RequireAuth a las rutas existentes (/api/productos,
//     /api/red, /api/checkout) en este sprint para no romper el contrato
//     actual ni los tests existentes. Se hará en F4 (admin) y F5/F6
//     (checkout, pedidos del usuario).
func SetupRoutes(app *fiber.App, ctrl Controllers, jwtSecret []byte) {
	// Rutas públicas — S1 las mantiene abiertas por diseño.
	app.Get("/api/health", controllers.HealthCheck)
	app.Get("/api/productos", ctrl.Producto.GetProductos)
	app.Get("/api/productos/:id", ctrl.Producto.GetProductoPorID)
	app.Get("/api/producto/:id", ctrl.Producto.GetProductoPorID)
	app.Get("/api/red", ctrl.Network.GetRed)
	app.Post("/api/checkout", ctrl.Checkout.Checkout)

	// Rutas de auth — rate-limited a 10 req/min por IP (login + register).
	// Aplicamos RateLimit SÓLO a estas dos (no global), tal como pide el sprint.
	app.Post("/api/login",
		middleware.RateLimit(10, time.Minute),
		ctrl.Auth.Login,
	)
	app.Post("/api/register",
		middleware.RateLimit(10, time.Minute),
		ctrl.Auth.Register, // placeholder F1: devuelve 501
	)
}
