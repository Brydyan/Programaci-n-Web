// Declaramos que este es el paquete principal (main), es decir, el archivo ejecutable que inicia el programa.
package main

import (
	// Importamos el framework principal Fiber.
	"github.com/gofiber/fiber/v2"
	// Importamos el middleware CORS para gestionar la seguridad entre distintos puertos/dominios.
	"github.com/gofiber/fiber/v2/middleware/cors"
	// Importamos nuestro propio paquete de rutas para delegarle la configuración de los endpoints.
	"multicatalogo-backend/routes"
)

// func main es el punto de entrada de la aplicación en Go. Todo comienza a ejecutarse aquí.
func main() {
	// Instanciamos una nueva aplicación de Fiber y la guardamos en la variable 'app'.
	app := fiber.New()

	// Implementamos el middleware CORS a nivel global usando app.Use() para interceptar todas las peticiones entrantes.
	app.Use(cors.New(cors.Config{
		// Configuramos el CORS para permitir peticiones desde localhost (desarrollo).
		AllowOrigins: "http://localhost:5173",
		// Declaramos de forma explícita qué cabeceras (Headers) se permitirán en la comunicación.
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		// Permitir métodos HTTP comunes.
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	// Llamamos a la función SetupRoutes de nuestro paquete 'routes', enviándole la instancia de nuestra 'app'.
	routes.SetupRoutes(app)

	// Ponemos a la aplicación a escuchar peticiones en el puerto 3000 de la máquina local. (Bloquea el hilo de ejecución).
	app.Listen(":3000")
}