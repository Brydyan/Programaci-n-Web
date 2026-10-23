// comisiones.go — Reparto de comisiones del MLM.
//
// Es la regla de negocio central del sistema ("el 10% del primer nivel, 5% del
// segundo, 2% del tercero"), así que vive en el paquete domain como función
// pura: entra un total y una cadena de nodos, sale el reparto. Sin I/O, sin
// redondeos implícitos del SQL, y por lo tanto trivial de testear —que es
// justamente lo que no pasaba antes, cuando las tasas vivían hardcodeadas
// dentro del repository y el reparto sólo se podía comprobar haciendo un
// checkout completo contra la base.
package domain

import "math"

// TasasComision es el reparto vigente, por posición en la cadena de upline.
// El slice es la fuente de verdad: añadir un nivel es añadir una tasa.
var TasasComision = []float64{0.10, 0.05, 0.02}

// MaxNivelesComision acota cuántos niveles se acreditan. Si el slice de
// tasas crece, este tope avisa de que hay que revisar el negocio.
const MaxNivelesComision = 3

// ComisionCalculada es el reparto de un nivel concreto, ya listo para
// insertarse como fila de la tabla `comisiones`.
type ComisionCalculada struct {
	Nivel      int     `json:"nivel"`
	ReferidoID int     `json:"referido_id"`
	Porcentaje float64 `json:"porcentaje"` // 10, 5, 2 — no 0.10
	Monto      float64 `json:"monto"`
}

// RepartirComisiones reparte `total` entre los nodos de `cadenaUpline`.
//
// `cadenaUpline` llega en orden: el índice 0 es el referente más cercano
// (nivel 1) y así sucesivamente. El reparto se corta en cuanto se acaba la
// cadena o se agotan las tasas — un usuario sin upline no genera comisión y
// eso es un resultado válido, no un error.
//
// El monto se redondea a 2 decimales (lo que la columna NUMERIC(10,2) haría al
// guardar) para que el total del ledger cuadre con lo que ve el usuario.
func RepartirComisiones(total float64, cadenaUpline []int, tasas []float64) []ComisionCalculada {
	if total <= 0 || len(cadenaUpline) == 0 || len(tasas) == 0 {
		return []ComisionCalculada{}
	}

	salida := make([]ComisionCalculada, 0, len(tasas))
	for i, referidoID := range cadenaUpline {
		if i >= len(tasas) {
			break
		}
		tasa := tasas[i]
		if tasa <= 0 {
			continue
		}
		salida = append(salida, ComisionCalculada{
			Nivel:      i + 1,
			ReferidoID: referidoID,
			Porcentaje: redondear2(tasa * 100),
			Monto:      redondear2(total * tasa),
		})
	}
	return salida
}

// TotalComisiones suma los montos de un reparto.
func TotalComisiones(comisiones []ComisionCalculada) float64 {
	total := 0.0
	for _, c := range comisiones {
		total += c.Monto
	}
	return redondear2(total)
}

// redondear2 lleva a dos decimales. math.Round corrige a mitad hacia arriba,
// que es lo que espera el redondeo bancario por defecto en Postgres.
func redondear2(v float64) float64 {
	return math.Round(v*100) / 100
}