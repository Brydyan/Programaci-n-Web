package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"multicatalogo-backend/config"
	"multicatalogo-backend/controllers"
	"multicatalogo-backend/repository"
	"multicatalogo-backend/routes"
)

func main() {
	// Inicializar conexiones
	config.InitDB()

	// Instanciar repositorio y servicio de caché con inyección de dependencias
	repo := repository.NewRepository(config.DB)
	cache := config.NewRedisService()

	// Construir controladores inyectando dependencias
	ctrl := routes.Controllers{
		Auth:     controllers.NewAuthController(repo),
		Producto: controllers.NewProductoController(repo, cache),
		Network:  controllers.NewNetworkController(repo, cache),
		Checkout: controllers.NewCheckoutController(repo, cache),
	}

	app := fiber.New()
	app.Use(cors.New(cors.Config{
		AllowOrigins: "http://localhost:5173",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	routes.SetupRoutes(app, ctrl)
	app.Listen(":3000")
}
