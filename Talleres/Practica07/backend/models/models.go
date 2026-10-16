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
var (
	ErrCredencialesInvalidas = errors.New("credenciales inválidas")
	ErrCuentaDesactivada     = errors.New("cuenta desactivada")
	ErrUsuarioNoEncontrado   = errors.New("usuario no encontrado")
	ErrEmailDuplicado        = errors.New("email ya registrado")
	ErrCodigoRefInvalido     = errors.New("código de referido inválido")
)

// LoginRequest define la estructura esperada para el cuerpo de la petición (JSON) al iniciar sesión.
type LoginRequest struct {
	// Email representa el correo del usuario; la etiqueta `json:"email"` indica cómo se mapea el JSON a esta variable.
	Email string `json:"email"`
	// Password representa la contraseña del usuario; se extrae del campo "password" del JSON entrante.
	Password string `json:"password"`
}

// Producto define la estructura de los datos de un artículo en nuestro catálogo.
type Producto struct {
	// ID es el identificador único numérico del producto.
	ID int `json:"id"`
	// Nombre es la descripción en texto del producto.
	Nombre string `json:"nombre"`
	// Precio es el costo del producto, almacenado como un número con decimales (float64).
	Precio float64 `json:"precio"`
	// Img es la URL o ruta que apunta a la fotografía o imagen del producto.
	Img string `json:"img"`
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