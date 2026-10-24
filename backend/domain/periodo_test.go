package domain

import (
	"testing"
	"time"
)

func TestPeriodoLiquidacion_etiqueta(t *testing.T) {
	casos := []struct {
		fecha    time.Time
		esperado string
	}{
		{time.Date(2026, 10, 22, 13, 59, 0, 0, time.UTC), "2026-10"},
		{time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), "2026-01"},
		{time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC), "2026-12"},
	}
	for _, c := range casos {
		if got := PeriodoLiquidacion(c.fecha); got != c.esperado {
			t.Errorf("PeriodoLiquidacion(%s) = %q, esperaba %q",
				c.fecha.Format(time.RFC3339), got, c.esperado)
		}
	}
}

// Dos pagos del mismo mes deben compartir etiqueta: de eso depende que el
// histórico se pueda leer.
func TestPeriodoLiquidacion_mismoMesMismaEtiqueta(t *testing.T) {
	inicio := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	fin := time.Date(2026, 10, 31, 23, 59, 0, 0, time.UTC)

	if PeriodoLiquidacion(inicio) != PeriodoLiquidacion(fin) {
		t.Error("dos fechas del mismo mes dio periodos distintos")
	}
}

// El cero de time.Time no debe producir "0001-01" en el historial.
func TestPeriodoLiquidacion_fechaCeroDevuelveVacio(t *testing.T) {
	if got := PeriodoLiquidacion(time.Time{}); got != "" {
		t.Errorf("esperaba cadena vacía, obtuvo %q", got)
	}
}

func TestPeriodoLiquidacion_ignoraZonaHoraria(t *testing.T) {
	// 2026-10-31 23:30 en UTC+2 ya es noviembre en UTC; el periodo se
	// calcula en UTC para que el histórico no cambie según quién mire.
	utc := time.Date(2026, 10, 31, 23, 30, 0, 0, time.UTC)
	if got := PeriodoLiquidacion(utc); got != "2026-10" {
		t.Errorf("esperaba 2026-10, obtuvo %q", got)
	}
}

func TestPeriodoActual(t *testing.T) {
	if len(PeriodoActual()) != len("2006-01") {
		t.Errorf("el periodo actual debería tener 7 caracteres, tiene %q", PeriodoActual())
	}
}
