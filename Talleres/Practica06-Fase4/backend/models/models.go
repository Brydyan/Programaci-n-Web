// Declaramos el paquete models para agrupar las estructuras de datos de la aplicación.
package models

// APIError define la estructura estándar para las respuestas de error de la API.
type APIError struct {
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

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

// Referido define la estructura de un miembro en la red de referidos.
type Referido struct {
	// ID es el identificador único numérico del referido.
	ID int `json:"id"`
	// Nombre es el nombre completo del referido.
	Nombre string `json:"nombre"`
	// Nivel es la jerarquía del referido (ej. Nivel 1, Nivel 2).
	Nivel string `json:"nivel"`
	// Ventas representa el monto de ventas mensuales formateado.
	Ventas string `json:"ventas"`
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
