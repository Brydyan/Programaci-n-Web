package interfaces

import (
	"context"
	"time"

	"multicatalogo-backend/domain"
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
//     usado por /api/admin/productos.
//   - CreateProducto / UpdateProducto / SoftDeleteProducto: CRUD admin.
type DBRepository interface {
	// Auth (F1)
	Login(email, password string) (*models.Usuario, error)
	Registrar(req models.RegisterRequest) (*models.RegisterResult, error)
	GetUsuarioByCodigoRef(codigo string) (*models.Referido, error)
	// GetReferidoIDByUsuario traduce usuario → nodo de red. nil sin error
	// significa que el usuario todavía no tiene nodo (alta sin referido).
	GetReferidoIDByUsuario(usuarioID int) (*int, error)

	// Catálogo público (F2 — antes mock, ahora DB)
	GetAllProductos() ([]models.Producto, error)
	GetProductoByID(id int) (*models.Producto, error)

	// Catálogo admin (F2): paginación + búsqueda + filtro activo.
	GetProductosAdmin(filtro ProductoFiltro) ([]models.Producto, int, error)
	CreateProducto(d dto.ProductoCreate) (*models.Producto, error)
	UpdateProducto(id int, d dto.ProductoUpdate) (*models.Producto, error)
	SoftDeleteProducto(id int) error

	// Red multinivel (Sprint 1) — SQL real con CTE recursiva, ya no hay
	// slice en memoria.
	//
	//   - GetRedGlobal:   todos los nodos (vista admin de GET /api/red).
	//   - GetRedPropia:   subárbol del nodo del usuario (GET /api/red/mia).
	//
	// Ambas devuelven la lista PLANA; el árbol anidado lo arma
	// domain.ConstruirArbol. Un usuario sin nodo devuelve slice vacío + nil
	// (todavía no tiene red: no es un error).
	GetRedGlobal() ([]domain.NodoPlano, error)
	GetRedPropia(usuarioID int) ([]domain.NodoPlano, error)

	// Checkout — CreatePedidoConUsuario inserta el pedido con usuario_id
	// (no sólo email_usuario), descuenta stock y escribe el ledger de
	// comisiones DENTRO de la misma transacción. El total se recalcula en
	// servidor. Devuelve el detalle completo para que el FE no necesite un
	// GET extra.
	CreatePedidoConUsuario(req dto.CheckoutRequest, usuarioID int) (*models.PedidoDetalle, error)

	// Pedidos — F3 expone gestión de pedidos y ciclo de vida con FSM.
	GetMisPedidos(usuarioID int, page, limit int) ([]models.PedidoResumen, int, error)
	GetPedidosAdmin(filtro PedidoFiltro) ([]models.PedidoResumen, int, error)
	GetPedidoDetalle(id int) (*models.PedidoDetalle, error)
	// CambiarEstadoPedido aplica la transición FSM en una transacción.
	// Side effects al cancelar: devuelve el stock reservado y revierte
	// comisiones pendientes/aprobadas del pedido. Devuelve
	// domain.ErrTransicionInvalida si el cambio no es legal, y
	// models.ErrPedidoNoEncontrado si el id no existe.
	CambiarEstadoPedido(id int, nuevoEstado domain.EstadoPedido) error

	// Ledger de comisiones (Sprint 2).
	//
	// El INSERT del ledger ocurre DENTRO de la transacción de
	// CreatePedidoConUsuario (ver domain.RepartirComisiones para el reparto).
	// Lo que queda fuera es el contador de conveniencia `referidos.ventas`,
	// que es best-effort y puede ir un instante desfasado.
	GetComisiones(filtro ComisionFiltro) ([]models.Comision, int, error)
	GetTotalesComisiones(referidoID *int) ([]models.ComisionTotal, error)
	ActualizarVentasUpline(emailUsuario string, total float64) error

	// Liquidaciones (Sprint 3) — cierran el ciclo del dinero.
	//
	//   - Liquidar:            transacción que paga todas las comisiones
	//                          'pendiente' de un distribuidor y devuelve
	//                          models.ErrNadaQueLiquidar si no debe nada.
	//   - GetLiquidaciones:    historial paginado (admin).
	//   - GetLiquidacionesDe:  historial de un distribuidor (propio).
	//   - GetDeuda:            qué se le debe a cada distribuidor.
	Liquidar(referidoID int) (*models.Liquidacion, error)
	GetLiquidaciones(page, limit int) ([]models.Liquidacion, int, error)
	GetLiquidacionesDe(referidoID, page, limit int) ([]models.Liquidacion, int, error)
	GetDeuda() ([]models.DeudaDistribuidor, error)

	// Reportes (Sprint 4) — agregados para el dashboard del admin.
	//
	// Son consultas de lectura y agregación, no filtered lists: por eso no
	// devuelven (datos, total) sino el agregado ya calculado. El detalle
	// paginado sigue sirviéndose por GetPedidosAdmin / GetComisiones.
	//
	//   - GetReporteResumen:        KPIs del periodo (ventas, pedidos, ticket
	//                              medio, comisiones devengadas).
	//   - GetTopProductos:          ranking por ingresos, excluye cancelados.
	//   - GetTopReferidos:          ranking por ventas, con su comisión real.
	//   - GetComisionesPendientes:  ledger pendiente de liquidar, sin liquidar.
	GetReporteResumen(filtro ReporteFiltro) (*models.ReporteResumen, error)
	GetTopProductos(limit int) ([]models.TopProducto, error)
	GetTopReferidos(limit int) ([]models.TopReferido, error)
	GetComisionesPendientes(limit int) ([]models.Comision, error)
}

// ReporteFiltro encapsula los query-params de GET /api/reportes/resumen.
//
// Desde y Hasta son opcionales (nil = sin recorte de fecha).
//
// Hasta es una cota EXCLUSIVA (`created_at < Hasta`), porque el controller ya
// convierte el `?hasta=2026-12-31` del usuario en `2027-01-01` antes de armar
// este struct. Quien construya un ReporteFiltro a mano debe respetar esa
// semántica o el último día del periodo se queda fuera.
type ReporteFiltro struct {
	Desde *time.Time
	Hasta *time.Time
}

// ComisionFiltro encapsula los query-params de GET /api/comisiones/mias y
// GET /api/admin/comisiones. ReferidoID nil = todos (admin).
type ComisionFiltro struct {
	ReferidoID *int
	Estado     string // "" = todos
	Page       int
	Limit      int
}

// ProductoFiltro encapsula los query-params de GET /api/admin/productos.
// Página y Limit se normalizan en el repository (defaults + cap=100).
type ProductoFiltro struct {
	Q      string // ILIKE sobre nombre y categoria; "" = sin búsqueda
	Activo *bool  // nil=todos, &true=v*, &false=inactivos
	Page   int    // 1-based
	Limit  int    // 1..100, default 20
}

// PedidoFiltro encapsula los query-params de GET /api/admin/pedidos.
// Estado vacío = sin filtro (devuelve todos los estados). Page y Limit
// se normalizan en el repository con los mismos defaults que ProductoFiltro.
type PedidoFiltro struct {
	Estado string // "" = todos; si no, debe pasar el CHECK del schema
	Page   int
	Limit  int
}

// CacheService es la abstracción de la caché. Todas las operaciones son
// nil-safe: si Redis está caído la API sigue sirviendo desde PostgreSQL.
type CacheService interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	// Increment suma 1 a un contador y devuelve el valor resultante. Se usa
	// para el versionado de claves de la red (invalidar sin enumerar claves).
	Increment(ctx context.Context, key string) (int64, error)
}
