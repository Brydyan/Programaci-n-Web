// Declaramos el paquete models para agrupar las estructuras de datos de la aplicación.
package models

import "errors"

// APIError define la estructura estándar para las respuestas de error de la API.
type APIError struct {
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// Errores sentinela del dominio. Los controllers los mapean a códigos HTTP
// según la tabla:
//
//	ErrCredencialesInvalidas → 401 UNAUTHORIZED
//	ErrCuentaDesactivada     → 403 FORBIDDEN
//	ErrUsuarioNoEncontrado   → 404 NOT_FOUND (rara vez se ve; el Login lo
//	                            colapsa con ErrCredencialesInvalidas para
//	                            no permitir enumeración de correos)
//	ErrEmailDuplicado        → 409 CONFLICT
//	ErrCodigoRefInvalido     → 400 VALIDATION_ERROR
//	ErrProductoNoEncontrado  → 404 NOT_FOUND
//	ErrProductoDuplicado     → 409 CONFLICT (nombre duplicado; opcional F2)
//	ErrPedidoNoEncontrado    → 404 NOT_FOUND
//	ErrNoEsDuenyo            → 404 NOT_FOUND (mapeado desde el controller;
//	                            no exponemos existencia del pedido a otros
//	                            clientes — preferimos 404 antes que 403)
var (
	ErrCredencialesInvalidas = errors.New("credenciales inválidas")
	ErrCuentaDesactivada     = errors.New("cuenta desactivada")
	ErrUsuarioNoEncontrado   = errors.New("usuario no encontrado")
	ErrEmailDuplicado        = errors.New("email ya registrado")
	ErrCodigoRefInvalido     = errors.New("código de referido inválido")
	ErrProductoNoEncontrado   = errors.New("producto no encontrado")
	ErrProductoDuplicado     = errors.New("producto ya existe")
	ErrPedidoNoEncontrado    = errors.New("pedido no encontrado")
	ErrNoEsDuenyo            = errors.New("no autorizado: no es dueño del pedido")
)

// LoginRequest define la estructura esperada para el cuerpo de la petición (JSON) al iniciar sesión.
type LoginRequest struct {
	// Email representa el correo del usuario; la etiqueta `json:"email"` indica cómo se mapea el JSON a esta variable.
	Email string `json:"email"`
	// Password representa la contraseña del usuario; se extrae del campo "password" del JSON entrante.
	Password string `json:"password"`
}

// Producto define la estructura de los datos de un artículo en nuestro catálogo.
// Migrado a DB real en F2: ahora refleja las columnas de la tabla `productos`
// (incluye categoria/stock/activo). Las versiones previas del FE que sólo
// consumían id/nombre/precio/img siguen funcionando porque los nuevos campos
// son zero-valued si la consulta no los devuelve.
type Producto struct {
	// ID es el identificador único numérico del producto.
	ID int `json:"id"`
	// Nombre es la descripción en texto del producto.
	Nombre string `json:"nombre"`
	// Precio es el costo del producto, almacenado como un número con decimales (float64).
	Precio float64 `json:"precio"`
	// Categoria agrupa el producto (rostro, tratamiento, etc.).
	Categoria string `json:"categoria"`
	// Stock disponible — los admin lo gestionan, la vista pública lo lee.
	Stock int `json:"stock"`
	// Img es la URL o ruta que apunta a la fotografía o imagen del producto.
	Img string `json:"img"`
	// Activo controla la presencia en el catálogo público. Los admin pueden
	// desactivarlo para mantener el histórico de pedidos sin exponerlo.
	Activo bool `json:"activo"`
}

// Referido define la estructura de un nodo del árbol multinivel.
// Antes (mock) incluía Nombre/Nivel/Ventas formateadas para el front;
// ahora expone los campos crudos de la tabla `referidos`. Los handlers
// que necesiten datos formateados para el front deben componerlos a
// partir de aquí (este sprint NO migra el endpoint /api/red, que sigue
// sirviendo el mock desde repository hasta F2).
type Referido struct {
	ID        int    `json:"id"`
	Codigo    string `json:"codigo"`
	ParentID  *int   `json:"parent_id,omitempty"`
	UsuarioID *int   `json:"usuario_id,omitempty"`
}

// Usuario es la fila de la tabla `usuarios` proyectada al exterior.
// Contraseña nunca se expone (sólo su hash vive en DB).
type Usuario struct {
	ID         int    `json:"id"`
	Email      string `json:"email"`
	Rol        string `json:"rol"` // "cliente" | "admin"
	Nombre     string `json:"nombre"`
	Activo     bool   `json:"activo"`
	ReferidoID *int   `json:"referido_id,omitempty"`
}

// RegisterRequest es el cuerpo de POST /api/register.
// Ref es opcional; cuando viene debe ser un código válido del alfabeto
// de referidos (lo verifica el controller con domain.CodigoEsValido).
type RegisterRequest struct {
	Nombre   string `json:"nombre"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Ref      string `json:"ref,omitempty"`
}

// RegisterResult es la respuesta de POST /api/register (201 Created).
// NO incluye un JWT: el contrato pide que el cliente haga login tras
// registrarse para autenticarse. F4 podría cambiar esto a "auto-login".
type RegisterResult struct {
	UsuarioID int    `json:"id"`
	Email     string `json:"email"`
	CodigoRef string `json:"codigo_ref"`
}

type ItemCarrito struct {
	ID       int     `json:"id"`
	Nombre   string  `json:"nombre"`
	Precio   float64 `json:"precio"`
	Cantidad int     `json:"cantidad"`
}

type CheckoutRequest struct {
	Nombre     string        `json:"nombre"`
	Email      string        `json:"email"`
	Direccion  string        `json:"direccion"`
	MetodoPago string        `json:"metodo_pago"`
	Items      []ItemCarrito `json:"items"`
}

type Pedido struct {
	ID     int     `json:"id"`
	Total  float64 `json:"total"`
	Estado string  `json:"estado"`
}

// PedidoDetalle es la vista completa de un pedido: cabecera + items.
// Es lo que devuelve GET /api/pedidos/:id y lo que regresa el checkout
// (POST /api/checkout) tras el COMMIT para que el FE tenga todo lo que
// necesita confirmar sin un GET extra.
//
// Campos:
//   - UsuarioID: puntero porque la columna pedidos.usuario_id permite
//     NULL (pedidos legacy de la migración 005 sin usuario asociado).
//     En código nuevo SIEMPRE se llena — el checkout lo exige.
//   - EmailUsuario: legado, conservado para no romper el FE existente y
//     porque el seed 005 mantiene filas con email_usuario NOT NULL.
//   - Items: slice de PedidoItem; nil si el pedido no tiene items (no
//     debería pasar, pero el repo lo deja defensive).
type PedidoDetalle struct {
	ID           int          `json:"id"`
	UsuarioID    *int         `json:"usuario_id,omitempty"`
	EmailUsuario string       `json:"email_usuario"`
	Total        float64      `json:"total"`
	MetodoPago   string       `json:"metodo_pago"`
	Estado       string       `json:"estado"`
	CreatedAt    string       `json:"created_at"`
	Items        []PedidoItem `json:"items,omitempty"`
}

// PedidoItem refleja la fila de pedido_items. producto_id se conserva
// aunque el producto esté soft-deleted — la FK tiene ON DELETE RESTRICT
// desde pedido_items pero el producto puede tener activo=false.
type PedidoItem struct {
	ID         int     `json:"id"`
	ProductoID int     `json:"producto_id"`
	Nombre     string  `json:"nombre"`
	Precio     float64 `json:"precio"`
	Cantidad   int     `json:"cantidad"`
}

// PedidoResumen es la fila "ligera" usada por los listados paginados
// (GET /api/pedidos/mios y GET /api/admin/pedidos). No incluye items —
// el FE los pide aparte con GET /api/pedidos/:id cuando el cliente abre
// el detalle.
type PedidoResumen struct {
	ID        int     `json:"id"`
	UsuarioID *int    `json:"usuario_id,omitempty"`
	Total     float64 `json:"total"`
	Estado    string  `json:"estado"`
	CreatedAt string  `json:"created_at"`
}