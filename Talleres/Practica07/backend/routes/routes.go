package routes

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"multicatalogo-backend/controllers"
	"multicatalogo-backend/middleware"
)

type Controllers struct {
	Auth          *controllers.AuthController
	Producto      *controllers.ProductoController
	Network       *controllers.NetworkController
	Checkout      *controllers.CheckoutController
	Pedidos       *controllers.PedidosController
	Comisiones    *controllers.ComisionesController
	Liquidaciones *controllers.LiquidacionesController
	Reportes      *controllers.ReportesController
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
	// Rutas públicas — siguen abiertas: health, catálogo público.
	app.Get("/api/health", controllers.HealthCheck)
	app.Get("/api/productos", ctrl.Producto.GetProductos)
	app.Get("/api/productos/:id", ctrl.Producto.GetProductoPorID)
	app.Get("/api/producto/:id", ctrl.Producto.GetProductoPorID)

	// Checkout (F3): protegido. RateLimit 10/min igual que auth.
	app.Post("/api/checkout",
		middleware.RequireAuth(jwtSecret),
		middleware.RateLimit(10, time.Minute),
		ctrl.Checkout.Checkout,
	)

	// Rutas de auth — rate-limited a 10 req/min por IP (login + register).
	app.Post("/api/login",
		middleware.RateLimit(10, time.Minute),
		ctrl.Auth.Login,
	)
	app.Post("/api/register",
		middleware.RateLimit(10, time.Minute),
		ctrl.Auth.Register,
	)

	// Rutas del cliente autenticado. Requieren JWT pero NO rol admin: cada
	// usuario ve lo suyo (sus pedidos, su subárbol de red).
	auth := app.Group("/api", middleware.RequireAuth(jwtSecret))
	auth.Get("/pedidos/mios", ctrl.Pedidos.GetMisPedidos)
	auth.Get("/pedidos/:id", ctrl.Pedidos.GetPedidoDetalle)
	// Sprint 1: la red deja de servirse sin auth. Antes GetRed era público y
	// devolvía un mock; ahora /api/red/mia devuelve el subárbol del token.
	auth.Get("/red/mia", ctrl.Network.GetRedMia)
	// Ledger del usuario: el dinero que le corresponde. La identidad sale
	// del token, no de un query-param.
	auth.Get("/comisiones/mias", ctrl.Comisiones.GetMias)
	// Cobros del usuario: el lado "recibí mi dinero" del ciclo.
	auth.Get("/liquidaciones/mias", ctrl.Liquidaciones.ListarMias)

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
	// Auditoría del ledger: todo el dinero con filtro por estado y por
	// distribuidor. Es lo que usará la liquidación (Sprint 3).
	admin.Get("/comisiones", ctrl.Comisiones.GetAdmin)
	// Liquidaciones: consultar la deuda y pagarla. El POST es la operación
	// que mueve dinero, por eso va en su propio handler con validación.
	admin.Get("/liquidaciones/deuda", ctrl.Liquidaciones.Deuda)
	admin.Get("/liquidaciones", ctrl.Liquidaciones.Listar)
	admin.Post("/liquidaciones", ctrl.Liquidaciones.Crear)

	// Árbol global de la red. Admin only (decisión S10 de SPRINT_S1): la
	// estructura completa y las ventas de toda la organización no son datos
	// públicos. Cada distribuidor usa /api/red/mia. Vive fuera del grupo
	// /api/admin porque el contrato público lo expone como /api/red.
	app.Get("/api/red",
		middleware.RequireAuth(jwtSecret),
		middleware.RequireAdmin(),
		ctrl.Network.GetRed,
	)

	// Reportes del dashboard. Admin only por la misma razón que /api/red:
	// los agregados (facturación total, ranking de distribuidores, comisiones
	// devengadas) son de la organización, no de un usuario.
	//
	// Viven en su propio grupo y no bajo /api/admin porque openapi.yaml los
	// publica como /api/reportes/* (sin el infijo admin).
	reportes := app.Group("/api/reportes",
		middleware.RequireAuth(jwtSecret),
		middleware.RequireAdmin(),
	)
	reportes.Get("/resumen", ctrl.Reportes.GetResumen)
	reportes.Get("/top-productos", ctrl.Reportes.GetTopProductos)
	reportes.Get("/top-referidos", ctrl.Reportes.GetTopReferidos)
	reportes.Get("/comisiones-pendientes", ctrl.Reportes.GetComisionesPendientes)
}
