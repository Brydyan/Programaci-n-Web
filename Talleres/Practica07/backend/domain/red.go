// red.go — Construcción del árbol multinivel a partir de una lista plana.
//
// La consulta SQL (recursive CTE) devuelve los nodos **planos**: uno por fila,
// con su parent_id y la profundidad relativa calculada en SQL sólo para
// ordenar. Ensamblar la jerarquía es lógica de negocio pura —sin I/O ni
// dependencias— así que vive aquí y se prueba en aislamiento, igual que la FSM
// de pedido_estado.go.
//
// Reglas que implementa ConstruirArbol:
//
//   - Un nodo cuyo parent_id no está en el conjunto se trata como raíz.
//     Necesario porque /api/red/mia devuelve un subárbol: el padre del nodo
//     del usuario queda fuera y no debe aparecer como "colgado".
//   - El nivel (nivel) se recalcula como profundidad real desde la raíz: 0
//     para la raíz, 1 para sus hijos. Así /api/red/mia entrega nivel=0 al
//     propio usuario y /api/red entrega la jerarquía global completa.
//   - Se cortan ciclos. El schema no impide que parent_id forme un lazo y una
//     recursión sin guarda equivaldría a un cuelgue del proceso.
package domain

// maxProfundidadRed acota la recursión de ConstruirArbol. El negocio reparte
// comisiones a 3 niveles, pero el árbol puede ser más profundo; 20 deja
// margen sin permitir que un lazo se convierta en un cuelgue.
const maxProfundidadRed = 20

// NodoPlano es una fila de la tabla `referidos` enrichcida con el nombre del
// usuario vinculado. La produce el repository; la consume ConstruirArbol.
type NodoPlano struct {
	ID        int
	Codigo    string
	ParentID  *int
	UsuarioID *int
	Nombre    string
	Ventas    float64
}

// NodoRed es un nodo del árbol ya ensamblado. Es el contrato de salida de
// GET /api/red y GET /api/red/mia (ver openapi.yaml → components.schemas.NodoRed).
type NodoRed struct {
	ID     int       `json:"id"`
	Codigo string    `json:"codigo"`
	Nombre string    `json:"nombre"`
	Nivel  int       `json:"nivel"`
	Ventas float64   `json:"ventas"`
	Hijos  []NodoRed `json:"hijos,omitempty"`
}

// ConstruirArbol ensambla la jerarquía a partir de los nodos planos.
//
// Devuelve las raíces en el orden de entrada. Un conjunto vacío devuelve un
// slice vacío (nunca nil) para que el JSON sea `[]` y no `null`.
func ConstruirArbol(nodos []NodoPlano) []NodoRed {
	if len(nodos) == 0 {
		return []NodoRed{}
	}

	// Dos pasadas: primero se sabe qué ids existen, después se decide quién es
	// hijo de quién. Con una sola pasada un nodo cuyo padre aparece después
	// quedaría colgando y se trataría como raíz.
	presente := make(map[int]struct{}, len(nodos))
	for _, n := range nodos {
		presente[n.ID] = struct{}{}
	}

	hijosDe := make(map[int][]NodoPlano)
	var raices []NodoPlano
	for _, n := range nodos {
		if n.ParentID != nil {
			if _, ok := presente[*n.ParentID]; ok {
				hijosDe[*n.ParentID] = append(hijosDe[*n.ParentID], n)
				continue
			}
		}
		raices = append(raices, n)
	}

	// Sin raíces el conjunto es un ciclo puro (A→B→A). Se corta usando el
	// primer nodo como raíz: mejor un árbol truncado que una lista vacía que
	// el usuario no sabe interpretar. El nivel acotado en armarNodos impide
	// que ese lazo se convierta en una recursión infinita.
	if len(raices) == 0 {
		raices = nodos[:1]
	}

	return armarNodos(raices, hijosDe, 0)
}

// armarNodos convierte un nivel a NodoRed y baja al siguiente usando el índice
// de hijos ya construido. El índice se pasa completo en cada llamada (en vez
// de recalcularlo sobre el subconjunto) para que los nietos no se pierdan.
func armarNodos(actuales []NodoPlano, hijosDe map[int][]NodoPlano, nivel int) []NodoRed {
	if len(actuales) == 0 || nivel >= maxProfundidadRed {
		return nil
	}
	salida := make([]NodoRed, 0, len(actuales))
	for _, n := range actuales {
		salida = append(salida, NodoRed{
			ID:     n.ID,
			Codigo: n.Codigo,
			Nombre: n.Nombre,
			Nivel:  nivel,
			Ventas: n.Ventas,
			Hijos:  armarNodos(hijosDe[n.ID], hijosDe, nivel+1),
		})
	}
	return salida
}

// ContarNodos recorre el árbol y devuelve cuántos nodos contiene en total.
// Lo usan los tests y sirve para validar que una respuesta no perdió ramas.
func ContarNodos(nodos []NodoRed) int {
	total := 0
	for _, n := range nodos {
		total += 1 + ContarNodos(n.Hijos)
	}
	return total
}

// TotalVentas suma las ventas de todo el subárbol.
func TotalVentas(nodos []NodoRed) float64 {
	total := 0.0
	for _, n := range nodos {
		total += n.Ventas + TotalVentas(n.Hijos)
	}
	return total
}
