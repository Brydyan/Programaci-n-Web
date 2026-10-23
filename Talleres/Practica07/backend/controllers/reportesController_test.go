package controllers

import (
	"strconv"
	"testing"
	"time"
)

// Tests de las funciones puras del controller de reportes. No tocan DB ni
// Redis: el comportamiento que verifican es el parseo de query-params, que es
// donde se concentran los errores silenciosos (una fecha mal parseada devuelve
// ceros en vez de un 400, y el admin ve un dashboard vacío sin saber por qué).

func TestParseFecha(t *testing.T) {
	t.Parallel()

	t.Run("vacío devuelve nil sin error", func(t *testing.T) {
		t.Parallel()
		got, err := parseFecha("")
		if err != nil {
			t.Fatalf("no esperaba error, obtuve %v", err)
		}
		if got != nil {
			t.Errorf("esperaba nil, obtuve %v", got)
		}
	})

	t.Run("AAAA-MM-DD se parsea a UTC medianoche", func(t *testing.T) {
		t.Parallel()
		got, err := parseFecha("2026-01-15")
		if err != nil {
			t.Fatalf("no esperaba error, obtuve %v", err)
		}
		esperado := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
		if got == nil || !got.Equal(esperado) {
			t.Errorf("esperaba %v, obtuve %v", esperado, got)
		}
	})

	// El caso que más se cuela: "15-01-2026" es válido para el usuario y
	// silencio para time.Parse. Sin este test, el filtro devolvería todo.
	casosInvalidos := []string{
		"15-01-2026",
		"2026-13-01", // mes inexistente
		"2026-02-30", // día inexistente
		"2026-1-5",   // sin zero-padding
		"15/01/2026",
		"ayer",
	}
	for _, raw := range casosInvalidos {
		t.Run("rechaza "+raw, func(t *testing.T) {
			t.Parallel()
			if _, err := parseFecha(raw); err == nil {
				t.Errorf("esperaba error para %q, obtuve nil", raw)
			}
		})
	}
}

// TestParseFechaHasta verifica que el límite superior sea el día SIGUIENTE.
// Es la diferencia entre "hasta el 31" incluyéndolo o excluyéndolo, y el fallo
// es invisible: el resumen simplemente sale un día corto.
func TestParseFechaHasta(t *testing.T) {
	t.Parallel()

	t.Run("suma un día", func(t *testing.T) {
		t.Parallel()
		got, err := parseFechaHasta("2026-01-15")
		if err != nil {
			t.Fatalf("no esperaba error, obtuve %v", err)
		}
		esperado := time.Date(2026, 1, 16, 0, 0, 0, 0, time.UTC)
		if got == nil || !got.Equal(esperado) {
			t.Errorf("esperaba %v, obtuve %v", esperado, got)
		}
	})

	t.Run("fin de mes", func(t *testing.T) {
		t.Parallel()
		got, _ := parseFechaHasta("2026-01-31")
		esperado := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
		if got == nil || !got.Equal(esperado) {
			t.Errorf("31 de enero debe acotar en 1 de febrero, obtuve %v", got)
		}
	})

	t.Run("año bisiesto", func(t *testing.T) {
		t.Parallel()
		got, _ := parseFechaHasta("2028-02-28")
		esperado := time.Date(2028, 2, 29, 0, 0, 0, 0, time.UTC)
		if got == nil || !got.Equal(esperado) {
			t.Errorf("28 de febrero de 2028 debe acotar en 29, obtuve %v", got)
		}
	})

	t.Run("vacío sigue siendo nil", func(t *testing.T) {
		t.Parallel()
		got, err := parseFechaHasta("")
		if err != nil || got != nil {
			t.Errorf("esperaba (nil, nil), obtuve (%v, %v)", got, err)
		}
	})
}

func TestParseLimit(t *testing.T) {
	t.Parallel()

	t.Run("default 10 sin parámetro", func(t *testing.T) {
		t.Parallel()
		got, err := parseLimit("")
		if err != nil || got != 10 {
			t.Errorf("esperaba 10 sin error, obtuve (%d, %v)", got, err)
		}
	})

	t.Run("acepta el rango 1..50", func(t *testing.T) {
		t.Parallel()
		for _, want := range []int{1, 25, 50} {
			raw := strconv.Itoa(want)
			got, err := parseLimit(raw)
			if err != nil {
				t.Errorf("limit=%s debía ser válido: %v", raw, err)
				continue
			}
			if got != want {
				t.Errorf("limit=%s devolvió %d, esperaba %d", raw, got, want)
			}
		}
	})

	// El tope de 50 evita que ?limit=999999 convierta un endpoint de
	// dashboard en un volcado de tabla.
	fueraDeRango := []string{"0", "-1", "51", "1000"}
	for _, raw := range fueraDeRango {
		t.Run("rechaza "+raw, func(t *testing.T) {
			t.Parallel()
			if _, err := parseLimit(raw); err == nil {
				t.Errorf("esperaba error para limit=%s", raw)
			}
		})
	}

	t.Run("rechaza no-numérico", func(t *testing.T) {
		t.Parallel()
		for _, raw := range []string{"abc", "10.5", "1e3", " 5"} {
			if _, err := parseLimit(raw); err == nil {
				t.Errorf("esperaba error para limit=%q", raw)
			}
		}
	})
}

// TestFechaOpcional protege contra la colisión de claves de caché: si "sin
// filtro" generara la misma clave que el valor cero de la fecha, un reporte
// sin fechas serviría datos de 0001-01-01.
func TestFechaOpcional(t *testing.T) {
	t.Parallel()

	if got := fechaOpcional(nil); got != "-" {
		t.Errorf("nil debe serializarse como \"-\", obtuve %q", got)
	}
	valor := time.Date(2026, 3, 7, 0, 0, 0, 0, time.UTC)
	if got := fechaOpcional(&valor); got != "2026-03-07" {
		t.Errorf("esperaba 2026-03-07, obtuve %q", got)
	}
	if fechaOpcional(nil) == fechaOpcional(&valor) {
		t.Error("nil y una fecha real no pueden compartir clave de caché")
	}
}
