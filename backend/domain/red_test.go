package domain

import "testing"

// ptrInt evita repetir &n en los literales de test.
func ptrInt(v int) *int { return &v }

func nodo(id int, parent *int, nombre string, ventas float64) NodoPlano {
	return NodoPlano{
		ID:       id,
		Codigo:   "CODE" + string(rune('A'+id)),
		ParentID: parent,
		Nombre:   nombre,
		Ventas:   ventas,
	}
}

// El caso base del negocio: raíz con dos hijos y un nieto.
func TestConstruirArbol_tresNiveles(t *testing.T) {
	planos := []NodoPlano{
		nodo(1, nil, "Raíz", 100),
		nodo(2, ptrInt(1), "Hijo A", 40),
		nodo(3, ptrInt(1), "Hijo B", 30),
		nodo(4, ptrInt(2), "Nieto A1", 10),
	}

	raices := ConstruirArbol(planos)

	if len(raices) != 1 {
		t.Fatalf("esperaba 1 raíz, obtuve %d", len(raices))
	}
	if raices[0].Nivel != 0 {
		t.Errorf("la raíz debe tener nivel 0, obtuvo %d", raices[0].Nivel)
	}
	if len(raices[0].Hijos) != 2 {
		t.Fatalf("esperaba 2 hijos, obtuve %d", len(raices[0].Hijos))
	}
	if raices[0].Hijos[0].Nivel != 1 {
		t.Errorf("los hijos deben tener nivel 1, obtuvo %d", raices[0].Hijos[0].Nivel)
	}
	nieto := raices[0].Hijos[0].Hijos
	if len(nieto) != 1 || nieto[0].Nivel != 2 {
		t.Errorf("esperaba 1 nieto de nivel 2, obtuve %+v", nieto)
	}
	if total := ContarNodos(raices); total != 4 {
		t.Errorf("el árbol debe conservar los 4 nodos, contiene %d", total)
	}
}

// /api/red/mia devuelve un subárbol: el padre del usuario del token no viene
// en el resultado y el nodo propio debe quedar como raíz (nivel 0).
func TestConstruirArbol_padreFueraDelConjunto(t *testing.T) {
	planos := []NodoPlano{
		nodo(7, nil, "Yo", 5), // el padre real (id 3) no está en el slice
		nodo(8, ptrInt(7), "Mi referido", 2),
	}

	raices := ConstruirArbol(planos)

	if len(raices) != 1 {
		t.Fatalf("el nodo propio debe quedar como raíz, obtuve %d raíces", len(raices))
	}
	if raices[0].ID != 7 || raices[0].Nivel != 0 {
		t.Errorf("esperaba id=7 nivel=0, obtuve id=%d nivel=%d", raices[0].ID, raices[0].Nivel)
	}
	if len(raices[0].Hijos) != 1 {
		t.Errorf("esperaba 1 hijo, obtuve %d", len(raices[0].Hijos))
	}
}

// Guarda anti-ciclo: A→B→A no puede colgarse el proceso. La recursion está
// acotada por maxProfundidadRed, así que el test debe terminar.
func TestConstruirArbol_cicloNoCuelga(t *testing.T) {
	planos := []NodoPlano{
		nodo(1, ptrInt(2), "A", 1),
		nodo(2, ptrInt(1), "B", 1),
	}

	raices := ConstruirArbol(planos)

	if len(raices) == 0 {
		t.Fatal("un ciclo puro debe devolver al menos una raíz truncada, no una lista vacía")
	}
	// Lo relevante es que la llamada retorna; el contenido truncado es aceptable.
	if total := ContarNodos(raices); total == 0 {
		t.Error("el árbol truncado debe conservar al menos un nodo")
	}
}

// Un slice vacío debe serializar como [] y no como null.
func TestConstruirArbol_vacioDevuelveSliceNoNil(t *testing.T) {
	raices := ConstruirArbol(nil)
	if raices == nil {
		t.Fatal("ConstruirArbol(nil) debe devolver un slice vacío, no nil")
	}
	if len(raices) != 0 {
		t.Errorf("esperaba 0 raíces, obtuve %d", len(raices))
	}
}

// Orden estable: las raíces conservan el orden de entrada.
func TestConstruirArbol_conservaOrdenDeRaices(t *testing.T) {
	planos := []NodoPlano{
		nodo(1, nil, "Raíz 1", 0),
		nodo(2, nil, "Raíz 2", 0),
		nodo(3, nil, "Raíz 3", 0),
	}

	raices := ConstruirArbol(planos)

	if len(raices) != 3 {
		t.Fatalf("esperaba 3 raíces, obtuve %d", len(raices))
	}
	for i, r := range raices {
		if r.ID != i+1 {
			t.Errorf("raíz %d: esperaba id=%d, obtuve %d", i, i+1, r.ID)
		}
	}
}

func TestTotalVentas_sumaTodoElSubarbol(t *testing.T) {
	raices := ConstruirArbol([]NodoPlano{
		nodo(1, nil, "Raíz", 100),
		nodo(2, ptrInt(1), "Hijo", 40),
		nodo(3, ptrInt(2), "Nieto", 10),
	})

	if got := TotalVentas(raices); got != 150 {
		t.Errorf("esperaba 150, obtuve %v", got)
	}
}
