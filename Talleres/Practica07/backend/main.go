// MultiCatálogo — entrypoint del backend Go/Fiber.
//
// Sprint S1 introduce: godotenv, validación de entorno, middleware de
// seguridad, CORS configurable por env, JWT real.
package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"

	"multicatalogo-backend/config"
	"multicatalogo-backend/controllers"
	"multicatalogo-backend/middleware"
	"multicatalogo-backend/repository"
	"multicatalogo-backend/routes"
)

func main() {
	// 1) Carga .env si existe. En producción se ignora silenciosamente
	//    cuando el archivo no está, así que es seguro llamarlo siempre.
	if err := godotenv.Load(); err != nil {
		log.Printf("[.env] archivo .env no encontrado, usando variables del entorno real")
	}

	// 2) Validar variables de entorno críticas — falla rápido si faltan.
	if err := validateEnv(); err != nil {
		log.Fatalf("[env] configuración inválida: %v", err)
	}
	port := defaultEnv("PORT", "3000")
	jwtSecret := []byte(os.Getenv("JWT_SECRET"))
	appEnv := defaultEnv("APP_ENV", "production")

	// 3) Inicializar conexiones (DB + Redis).
	config.InitDB()

	// 4) Inyección de dependencias en controllers.
	repo := repository.NewRepository(config.DB)
	cache := config.NewRedisService()
	ctrl := routes.Controllers{
		Auth:          controllers.NewAuthController(repo, cache),
		Producto:      controllers.NewProductoController(repo, cache),
		Network:       controllers.NewNetworkController(repo, cache),
		Checkout:      controllers.NewCheckoutController(repo, cache),
		Pedidos:       controllers.NewPedidosController(repo, cache),
		Comisiones:    controllers.NewComisionesController(repo),
		Liquidaciones: controllers.NewLiquidacionesController(repo),
		Reportes:      controllers.NewReportesController(repo, cache),
	}

	// 5) App + middlewares globales.
	app := fiber.New()
	app.Use(middleware.SecurityHeaders())
	app.Use(cors.New(cors.Config{
		AllowOrigins: buildAllowedOrigins(appEnv),
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	// Inyecta el secret JWT en c.Locals para que Login (y futuros handlers
	// que firmen tokens) puedan leerlo sin tocar os.Getenv por request.
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(middleware.JWTSecretKey, jwtSecret)
		return c.Next()
	})

	// 6) Rutas.
	routes.SetupRoutes(app, ctrl, jwtSecret)

	// 6.5) Servir frontend (SPA static files + fallback a index.html).
	// En Heroku: ./frontend/dist (generado en el Dockerfile)
	// En desarrollo: ../frontend/dist (desde raíz del proyecto)
	app.Static("/assets", "./frontend/dist/assets")
	app.Static("/", "./frontend/dist")
	app.Get("/*", func(c *fiber.Ctx) error {
		// SPA routing: cualquier ruta no encontrada → index.html
		return c.SendFile("./frontend/dist/index.html")
	})

	// 7) Listen.
	log.Printf("[boot] APP_ENV=%s puerto=%s", appEnv, port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("[boot] error iniciando listener: %v", err)
	}
}

// validateEnv exige JWT_SECRET (>=32 bytes) y ALLOWED_ORIGINS (no vacío).
// Otros vars (DATABASE_URL, REDIS_URL) son opcionales y caen a defaults.
func validateEnv() error {
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if len(secret) < 32 {
		return fmt.Errorf("JWT_SECRET debe tener >= 32 bytes (actual: %d)", len(secret))
	}
	if strings.TrimSpace(os.Getenv("ALLOWED_ORIGINS")) == "" {
		return fmt.Errorf("ALLOWED_ORIGINS debe estar definido (CSV de orígenes permitidos)")
	}
	return nil
}

// defaultEnv devuelve el valor de env `key` o `fallback` si está vacío.
func defaultEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

// buildAllowedOrigins devuelve los orígenes para CORS en formato CSV.
// En dev añade los puertos típicos de Vite (5173/5174) si no están ya.
func buildAllowedOrigins(appEnv string) string {
	parts := []string{}
	for _, p := range strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",") {
		if s := strings.TrimSpace(p); s != "" {
			parts = append(parts, s)
		}
	}
	if appEnv == "dev" {
		for _, vp := range []string{"http://localhost:5173", "http://localhost:5174"} {
			found := false
			for _, existing := range parts {
				if existing == vp {
					found = true
					break
				}
			}
			if !found {
				parts = append(parts, vp)
			}
		}
	}
	return strings.Join(parts, ",")
}
