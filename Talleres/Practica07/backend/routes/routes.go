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
	Pedidos  *controllers.PedidosController
}

// SetupRoutes registra las rutas del backend.
//
// Parametrización para S1:
//
//   - jwtSecret: clave HMAC usada por Login para firmar tokens y por
//     RequireAuth para validarlos. Se inyecta también como
//     c.Locals(middleware.JWTSecretKey) en cada request.
//
// F3 (gestión de pedidos + ciclo de vida):
//   - /api/checkout pasa a estar protegida con RequireAuth + RateLimit
//     (el cliente debe estar autenticado para comprar). Rate limit por
//     IP igual que /api/login: 10 req/min.
//   - /api/pedidos/mios y /api/pedidos/:id: cliente autenticado.
//   - /api/admin/pedidos (GET) y /api/admin/pedidos/:id/estado (PATCH):
//     admin only.
func SetupRoutes(app *fiber.App, ctrl Controllers, jwtSecret []byte) {
	// Rutas públicas — siguen abiertas: health, catálogo público, red.
	app.Get("/api/health", controllers.HealthCheck)
	app.Get("/api/productos", ctrl.Producto.GetProductos)
	app.Get("/api/productos/:id", ctrl.Producto.GetProductoPorID)
	app.Get("/api/producto/:id", ctrl.Producto.GetProductoPorID)
	app.Get("/api/red", ctrl.Network.GetRed)

	// Checkout (F3): ahora protegido. RateLimit 10/min igual que auth.
	app.Post("/api/checkout",
		middleware.RequireAuth(jwtSecret),
		middleware.RateLimit(10, time.Minute),
		ctrl.Checkout.Checkout,
	)

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

	// Rutas de pedidos del cliente autenticado (F3). Requieren JWT pero
	// NO rol admin: el cliente puede ver sus propios pedidos.
	auth := app.Group("/api", middleware.RequireAuth(jwtSecret))
	auth.Get("/pedidos/mios", ctrl.Pedidos.GetMisPedidos)
	auth.Get("/pedidos/:id", ctrl.Pedidos.GetPedidoDetalle)

	// Rutas admin (F2 + F3): requieren JWT válido (RequireAuth) y
	// rol=admin (RequireAdmin). El orden es importante — RequireAdmin
	// lee c.Locals(LocalUserKey) que RequireAuth pobla.
	admin := app.Group("/api/admin",
		middleware.RequireAuth(jwtSecret),
		middleware.RequireAdmin(),
	)
	admin.Get("/productos", ctrl.Producto.AdminList)
	admin.Post("/productos", ctrl.Producto.AdminCreate)
	admin.Put("/productos/:id", ctrl.Producto.AdminUpdate)
	admin.Delete("/productos/:id", ctrl.Producto.AdminDelete)
	// Pedidos admin (F3): listado global + cambio de estado con FSM.
	admin.Get("/pedidos", ctrl.Pedidos.GetPedidosAdmin)
	admin.Patch("/pedidos/:id/estado", ctrl.Pedidos.PatchEstado)
}