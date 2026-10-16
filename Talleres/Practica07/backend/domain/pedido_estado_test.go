package domain

import "testing"

// TestEsTransicionValida recorre la matriz de transiciones permitidas y
// rechaza explícitamente los pares inválidos más importantes. La idea es
// que cualquier cambio a la FSM se note aquí: si añades un nuevo estado
// sin actualizar los casos válidos, este test se rompe.
func TestEsTransicionValida(t *testing.T) {
	cases := []struct {
		from EstadoPedido
		to   EstadoPedido
		want bool
		desc string
	}{
		// Válidas — pendiente puede avanzar o cancelar
		{EstadoPendiente, EstadoPreparando, true, "pendiente → preparando"},
		{EstadoPendiente, EstadoCancelado, true, "pendiente → cancelado"},

		// Válidas — preparando puede enviar o cancelar
		{EstadoPreparando, EstadoEnviado, true, "preparando → enviado"},
		{EstadoPreparando, EstadoCancelado, true, "preparando → cancelado"},

		// Válidas — enviado sólo puede pasar a entregado (NO cancelar)
		{EstadoEnviado, EstadoEntregado, true, "enviado → entregado"},

		// Inválidas — desde enviado NO se puede cancelar
		{EstadoEnviado, EstadoCancelado, false, "enviado → cancelado (no se permite tras envío)"},

		// Inválidas — desde entregado (terminal) no se sale
		{EstadoEntregado, EstadoPendiente, false, "entregado → pendiente (terminal)"},
		{EstadoEntregado, EstadoPreparando, false, "entregado → preparando (terminal)"},
		{EstadoEntregado, EstadoEnviado, false, "entregado → enviado (terminal)"},
		{EstadoEntregado, EstadoCancelado, false, "entregado → cancelado (terminal)"},

		// Inválidas — desde cancelado (terminal) no se sale
		{EstadoCancelado, EstadoPendiente, false, "cancelado → pendiente (terminal)"},
		{EstadoCancelado, EstadoPreparando, false, "cancelado → preparando (terminal)"},
		{EstadoCancelado, EstadoEnviado, false, "cancelado → enviado (terminal)"},
		{EstadoCancelado, EstadoEntregado, false, "cancelado → entregado (terminal)"},

		// Inválidas — no se permite saltar pasos
		{EstadoPendiente, EstadoEnviado, false, "pendiente → enviado (salta preparando)"},
		{EstadoPendiente, EstadoEntregado, false, "pendiente → entregado (salta pasos)"},
		{EstadoPreparando, EstadoEntregado, false, "preparando → entregado (salta enviado)"},
		{EstadoPreparando, EstadoPendiente, false, "preparando → pendiente (no reversión)"},
		{EstadoEnviado, EstadoPreparando, false, "enviado → preparando (no reversión)"},
		{EstadoEnviado, EstadoPendiente, false, "enviado → pendiente (no reversión)"},

		// Inválidas — no-op mismo estado
		{EstadoPendiente, EstadoPendiente, false, "no-op pendiente"},
		{EstadoPreparando, EstadoPreparando, false, "no-op preparando"},
		{EstadoEnviado, EstadoEnviado, false, "no-op enviado"},
		{EstadoEntregado, EstadoEntregado, false, "no-op entregado"},
		{EstadoCancelado, EstadoCancelado, false, "no-op cancelado"},

		// Inválidas — estado desconocido
		{EstadoPedido("desconocido"), EstadoPendiente, false, "from desconocido"},
		{EstadoPendiente, EstadoPedido("desconocido"), false, "to desconocido"},
	}

	for _, tc := range cases {
		t.Run(string(tc.from)+"→"+string(tc.to), func(t *testing.T) {
			got := EsTransicionValida(tc.from, tc.to)
			if got != tc.want {
				t.Errorf("EsTransicionValida(%q, %q) = %v; want %v  (%s)",
					tc.from, tc.to, got, tc.want, tc.desc)
			}
		})
	}
}

// TestEstadosFinales verifica que el set devuelto es exactamente el
// esperado y que esos estados son terminales en la FSM (no salen a
// ningún otro). Es un test "documental" para que un cambio accidental
// en la matriz de transiciones se detecte.
func TestEstadosFinales(t *testing.T) {
	esperados := []EstadoPedido{EstadoEntregado, EstadoCancelado}
	got := EstadosFinales()
	if len(got) != len(esperados) {
		t.Fatalf("esperaba %d finales, got %d", len(esperados), len(got))
	}
	// Comparación order-insensitive.
	have := make(map[EstadoPedido]bool, len(got))
	for _, e := range got {
		have[e] = true
	}
	for _, e := range esperados {
		if !have[e] {
			t.Errorf("falta estado final %q", e)
		}
	}

	// Coherencia con la matriz: todo estado final debe tener lista vacía
	// (o ausente) de transiciones permitidas, y NO debe haber ninguna
	// transición desde él a otro estado.
	for _, f := range esperados {
		if permitidas, ok := transicionesPermitidas[f]; ok && len(permitidas) > 0 {
			t.Errorf("estado final %q tiene transiciones salientes: %v", f, permitidas)
		}
		for otro := range transicionesPermitidas {
			if otro == f {
				continue
			}
			if EsTransicionValida(f, EstadoPedido(otro)) {
				t.Errorf("transición ilegal detectada desde final %q → %q", f, otro)
			}
		}
	}
}

// TestEstadosQuePermitenCancelar verifica que el set devuelto es
// coherente con la FSM: cada estado de la lista debe efectivamente
// permitir transición a cancelado, y el tamaño del set debe ser
// exactamente el esperado.
func TestEstadosQuePermitenCancelar(t *testing.T) {
	got := EstadosQuePermitenCancelar()
	if len(got) != 2 {
		t.Fatalf("esperaba 2 estados cancelables, got %d: %v", len(got), got)
	}

	for _, s := range got {
		if !EsTransicionValida(s, EstadoCancelado) {
			t.Errorf("estado %q listado como cancelable, pero la FSM no lo permite", s)
		}
	}

	// Y el reverso: cualquier estado que SÍ permite cancelar debe
	// aparecer en el set.
	for from := range transicionesPermitidas {
		if EsTransicionValida(from, EstadoCancelado) {
			found := false
			for _, s := range got {
				if s == from {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("estado %q permite cancelar pero no está en EstadosQuePermitenCancelar", from)
			}
		}
	}
}