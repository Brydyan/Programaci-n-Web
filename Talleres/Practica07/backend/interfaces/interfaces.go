package interfaces

import (
	"context"
	"time"

	"multicatalogo-backend/models"
)

// DBRepository es la fachada de datos del backend. Mantenerla en un
// paquete dedicado permite que controllers y middleware dependan de la
// abstracción, no de un *Repository concreto (sustituible por un mock
// o una implementación alternativa en tests).
//
// Sprint F1 reemplaza el antiguo `Authenticate(email, password)` (mock
// hardcoded) por tres métodos reales contra PostgreSQL:
//
//   - Login: valida credenciales (bcrypt) y devuelve el usuario.
//   - Registrar: crea usuario + nodo de referido con código único.
//   - GetUsuarioByCodigoRef: consulta el árbol por código (utilidad).
type DBRepository interface {
	// Auth (F1)
	Login(email, password string) (*models.Usuario, error)
	Registrar(req models.RegisterRequest) (*models.RegisterResult, error)
	GetUsuarioByCodigoRef(codigo string) (*models.Referido, error)

	// Catálogo (aún mock hasta F2)
	GetAllProducts() []models.Producto
	GetProductByID(id int) (*models.Producto, error)

	// Red (aún mock hasta F2)
	GetNetwork() []models.Referido

	// Checkout (real DB desde S1)
	ProcessCheckout(req models.CheckoutRequest) (*models.Pedido, error)

	// Comisiones — ahora hace upline walk real (F1)
	ActualizarComisiones(total float64, emailUsuario string)
}

type CacheService interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}