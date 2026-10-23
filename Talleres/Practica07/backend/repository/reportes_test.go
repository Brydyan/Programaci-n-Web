package repository

import "testing"

// TestNormalizarLimitReporte blinda el tope del `?limit=` de los reportes.
//
// El fallo que previene: sin tope, `?limit=999999` en un endpoint de
// dashboard se convierte en un volcado de tabla que tumba Postgres por un
// GROUP BY con JOIN sobre pedido_items. No da error, sólo deja la API sin
// responder — el peor tipo de bug.
func TestNormalizarLimitReporte(t *testing.T) {
	t.Parallel()

	t.Run("acota al rango 1..50 con default 10", func(t *testing.T) {
		t.Parallel()
		casos := []struct {
			in       int
			esperado int
			motivo   string
		}{
			{0, 10, "sin parámetro → default del contrato"},
			{-5, 10, "negativo → default, no 0 (LIMIT 0 no devuelve nada)"},
			{1, 1, "límite inferior"},
			{10, 10, "valor por defecto explícito"},
			{50, 50, "límite superior"},
			{51, 50, "uno por encima se recorta"},
			{100000, 50, "abuso recortado"},
		}
		for _, c := range casos {
			got := normalizarLimitReporte(c.in)
			if got != c.esperado {
				t.Errorf("limit=%d: esperaba %d, obtuve %d (%s)",
					c.in, c.esperado, got, c.motivo)
			}
		}
	})

	// El recorte nunca puede devolver 0: en SQL `LIMIT 0` devuelve el
	// conjunto vacío y el admin vería un dashboard sin filas sin explicación.
	t.Run("nunca devuelve 0", func(t *testing.T) {
		t.Parallel()
		for _, in := range []int{-100, -1, 0, 1, 50, 5000} {
			if got := normalizarLimitReporte(in); got < 1 {
				t.Errorf("limit=%d produjo %d, que vaciaría la respuesta", in, got)
			}
		}
	})
}
