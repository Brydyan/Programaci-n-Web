package interfaces

import (
	"context"
	"time"

	"multicatalogo-backend/internal/dto"
	"multicatalogo-backend/models"
)

// DBRepository es la fachada de datos del backend. Mantenerla en un
// paquete dedicado permite que controllers y middleware dependan de la
// abstracción, no de un *Repository concreto (sustituible por un mock
// o una implementación alternativa en tests).
//
// F2 reemplaza los mocks de catálogo por métodos reales contra la tabla
// `productos` (PostgreSQL). Las firmas públicas mantienen la misma forma
// que antes pero ahora devuelven `(slice, error)` / `(ptr, error)` para
// propagar fallos de DB al controller.
//
// Diferencias clave:
//   - GetAllProductos: sólo productos activo=true (vista pública).
//   - GetProductoByID:  incluye activos e inactivos (404 si no existe).
//   - GetProductosAdmin: listado paginado + búsqueda + filtro activo,
//                        usado por /api/admin/productos.
//   - CreateProducto / UpdateProducto / SoftDeleteProducto: CRUD admin.
type DBRepository interface {
	// Auth (F1)
	Login(email, password string) (*models.Usuario, error)
	Registrar(req models.RegisterRequest) (*models.RegisterResult, error)
	GetUsuarioByCodigoRef(codigo string) (*models.Referido, error)

	// Catálogo público (F2 — antes mock, ahora DB)
	GetAllProductos() ([]models.Producto, error)
	GetProductoByID(id int) (*models.Producto, error)

	// Catálogo admin (F2): paginación + búsqueda + filtro activo.
	GetProductosAdmin(filtro ProductoFiltro) ([]models.Producto, int, error)
	CreateProducto(d dto.ProductoCreate) (*models.Producto, error)
	UpdateProducto(id int, d dto.ProductoUpdate) (*models.Producto, error)
	SoftDeleteProducto(id int) error

	// Red (aún mock hasta F2)
	GetNetwork() []models.Referido

	// Checkout (real DB desde S1)
	ProcessCheckout(req models.CheckoutRequest) (*models.Pedido, error)

	// Comisiones — ahora hace upline walk real (F1)
	ActualizarComisiones(total float64, emailUsuario string)
}

// ProductoFiltro encapsula los query-params de GET /api/admin/productos.
// Página y Limit se normalizan en el repository (defaults + cap=100).
type ProductoFiltro struct {
	Q     string // ILIKE sobre nombre y categoria; "" = sin búsqueda
	Activo *bool  // nil=todos, &true=v*, &false=inactivos
	Page  int    // 1-based
	Limit int    // 1..100, default 20
}

type CacheService interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}