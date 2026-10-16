package routes

import (
	"github.com/gofiber/fiber/v2"
	"multicatalogo-backend/controllers"
)

type Controllers struct {
	Auth     *controllers.AuthController
	Producto *controllers.ProductoController
	Network  *controllers.NetworkController
	Checkout *controllers.CheckoutController
}

func SetupRoutes(app *fiber.App, ctrl Controllers) {
	app.Get("/api/health", controllers.HealthCheck)
	app.Post("/api/login", ctrl.Auth.Login)
	app.Get("/api/productos", ctrl.Producto.GetProductos)
	app.Get("/api/productos/:id", ctrl.Producto.GetProductoPorID)
	app.Get("/api/producto/:id", ctrl.Producto.GetProductoPorID)
	app.Get("/api/red", ctrl.Network.GetRed)
	app.Post("/api/checkout", ctrl.Checkout.Checkout)
}
