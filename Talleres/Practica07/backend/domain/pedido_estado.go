// pedido_estado.go — Máquina de estados finita (FSM) del ciclo de vida de
// un pedido. Vive en el paquete domain (sin I/O ni dependencias de
// frameworks) para que pueda probarse en aislamiento y sea trivialmente
// reutilizable desde controllers o repository.
//
// Estados:
//   - pendiente  → estado inicial al crear el pedido (default del schema).
//   - preparando → el operador/admin lo está armando.
//   - enviado    → ya en camino al cliente.
//   - entregado  → recibido por el cliente (estado final).
//   - cancelado  → cancelado por el usuario o por admin (estado final).
//
// Reglas de cancelación:
//   - Sólo se permite cancelar desde pendiente o preparando. Una vez
//     enviado (o entregado) la cancelación ya no es razonable: el
//     paquete está fuera del almacén. Esto se refleja en
//     EstadosQuePermitenCancelar().
//
// Estados terminales:
//   - entregado y cancelado no aceptan transiciones de salida. Se
//     devuelven en EstadosFinales() para que el controller pueda
//     documentar el contrato o el repository rechace intentos con un
//     error claro (ErrTransicionInvalida).
package domain

import "errors"

// EstadoPedido es el tipo fuerte del estado del pedido. Usar string
// directamente permite errores tipográficos; este tipo se valida con
// el oneof de dto al parsear requests PATCH.
type EstadoPedido string

const (
	// EstadoPendiente es el estado por defecto al INSERT (alineado con el
	// default del schema: pedidos.estado DEFAULT 'pendiente').
	EstadoPendiente EstadoPedido = "pendiente"
	// EstadoPreparando indica que el pedido está siendo armado.
	EstadoPreparando EstadoPedido = "preparando"
	// EstadoEnviado indica que el pedido ya está en camino.
	EstadoEnviado EstadoPedido = "enviado"
	// EstadoEntregado es estado terminal: el cliente lo recibió.
	EstadoEntregado EstadoPedido = "entregado"
	// EstadoCancelado es estado terminal: el pedido fue cancelado.
	EstadoCancelado EstadoPedido = "cancelado"
)

// ErrTransicionInvalida es el sentinel que devuelve el repository cuando
// el cambio de estado pedido por el admin no cumple con la FSM.
// El controller lo mapea a 409 INVALID_TRANSITION.
var ErrTransicionInvalida = errors.New("transición de estado inválida")

// transicionesPermitidas codifica la matriz de transiciones válidas.
// Modelada como map[from][]to para que el lookup sea O(1) y para que
// añadir un estado nuevo (p.ej. "devuelto") sólo requiera tocar este map.
//
// Mantener este map privado fuerza a que toda consulta pase por
// EsTransicionValida — el controller y el repository nunca leen
// directamente la matriz.
//
// Nota: la cancelación está permitida desde pendiente y preparando
// (decisión documentada arriba). NO se permite desde enviado ni
// entregado: una vez despachado, la cancelación es operativa del
// transportista, no del sistema.
var transicionesPermitidas = map[EstadoPedido][]EstadoPedido{
	EstadoPendiente:  {EstadoPreparando, EstadoCancelado},
	EstadoPreparando: {EstadoEnviado, EstadoCancelado},
	EstadoEnviado:    {EstadoEntregado},
	EstadoEntregado:  {}, // terminal
	EstadoCancelado:  {}, // terminal
}

// EsTransicionValida devuelve true si pasar de from a to es legal según
// la FSM. Devuelve false si from == to (no-op) o si la combinación no
// está en la matriz.
//
// Casos cubiertos:
//   - pendiente  → preparando | cancelado → true
//   - preparando → enviado    | cancelado → true
//   - enviado    → entregado                → true
//   - entregado  → *                        → false (terminal)
//   - cancelado  → *                        → false (terminal)
//   - * → * mismo estado                     → false (no-op intencional)
func EsTransicionValida(from, to EstadoPedido) bool {
	if from == to {
		return false
	}
	permitidas, ok := transicionesPermitidas[from]
	if !ok {
		return false
	}
	for _, p := range permitidas {
		if p == to {
			return true
		}
	}
	return false
}

// EstadosFinales devuelve los estados terminales de la FSM. Útil para
// que el controller documente el contrato o el FE renderice la UI de
// forma distinta (sin botones de acción).
func EstadosFinales() []EstadoPedido {
	return []EstadoPedido{EstadoEntregado, EstadoCancelado}
}

// EstadosQuePermitenCancelar devuelve los estados desde los que aún se
// puede transicionar a EstadoCancelado. Mantener este set sincronizado
// con transicionesPermitidas es responsabilidad del programador (Go no
// tiene tipos dependientes para hacer el cross-check estático); un test
// en pedido_estado_test.go verifica la coherencia.
func EstadosQuePermitenCancelar() []EstadoPedido {
	return []EstadoPedido{EstadoPendiente, EstadoPreparando}
}
