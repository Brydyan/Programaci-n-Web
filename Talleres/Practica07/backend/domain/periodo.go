// periodo.go — Etiqueta de periodo de una liquidación.
//
// Es lógica de negocio mínima pero no trivial: el periodo agrupa pagos y se
// muestra en el historial. Si dos liquidaciones del mismo mes no comparten
// etiqueta, el histórico deja de ser legible.
//
// Vive en domain (puro) para poder testearlo sin base de datos ni reloj real.
package domain

import "time"

// FormatoPeriodo es la convención `YYYY-MM` que usa la columna
// liquidaciones.periodo.
const FormatoPeriodo = "2006-01"

// PeriodoLiquidacion devuelve la etiqueta del periodo al que pertenece una
// fecha. El cero de time.Time devuelve la cadena vacía en vez de "0001-01":
// una liquidación sin fecha real debe poder quedar sin periodo en vez de
// mostrar un año absurdo en el historial.
func PeriodoLiquidacion(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(FormatoPeriodo)
}

// PeriodoActual es la etiqueta del periodo en curso.
func PeriodoActual() string {
	return PeriodoLiquidacion(time.Now())
}