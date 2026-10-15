package routes

import (
	"github.com/gofiber/fiber/v2"
	"multicatalogo-backend/controllers"
)

// SetupRoutes registra todos los endpoints de la API en la aplicación Fiber.
func SetupRoutes(app *fiber.App) {
	// Endpoint de comprobación de salud del servidor
	app.Get("/api/health", controllers.HealthCheck)

	// Endpoint para Login y autenticación
	app.Post("/api/login", controllers.Login)

	// Endpoints para Catálogo de Productos
	app.Get("/api/productos", controllers.GetProductos)
	app.Get("/api/productos/:id", controllers.GetProductoPorID)
	app.Get("/api/producto/:id", controllers.GetProductoPorID)

	// Endpoint para Red de Referidos
	app.Get("/api/red", controllers.GetRed)
}