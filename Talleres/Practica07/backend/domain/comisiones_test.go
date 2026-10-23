package domain

import "testing"

func TestRepartirComisiones_cadenaCompleta(t *testing.T) {
	// Comprador colgado del nodo 2, cuyo upline es 2 ← 1.
	comisiones := RepartirComisiones(100, []int{2, 1}, TasasComision)

	if len(comisiones) != 2 {
		t.Fatalf("esperaba 2 comisiones (cadena de 2), obtuvo %d", len(comisiones))
	}
	if comisiones[0].Monto != 10 || comisiones[0].Porcentaje != 10 {
		t.Errorf("nivel 1: esperaba 10%% / 10.00, obtuvo %.2f%% / %.2f",
			comisiones[0].Porcentaje, comisiones[0].Monto)
	}
	if comisiones[1].Monto != 5 || comisiones[1].Porcentaje != 5 {
		t.Errorf("nivel 2: esperaba 5%% / 5.00, obtuvo %.2f%% / %.2f",
			comisiones[1].Porcentaje, comisiones[1].Monto)
	}
	if comisiones[0].Nivel != 1 || comisiones[1].Nivel != 2 {
		t.Errorf("los niveles deben ser 1 y 2, obtuvieron %d y %d",
			comisiones[0].Nivel, comisiones[1].Nivel)
	}
}

// La cadena manda sobre las tasas: un usuario nuevo sin upline no genera
// comisión y eso es un resultado válido, no un error.
func TestRepartirComisiones_cadenaCorta(t *testing.T) {
	comisiones := RepartirComisiones(80, []int{9}, TasasComision)

	if len(comisiones) != 1 {
		t.Fatalf("esperaba 1 comisión, obtuvo %d", len(comisiones))
	}
	if comisiones[0].Monto != 8 {
		t.Errorf("esperaba 8.00, obtuvo %.2f", comisiones[0].Monto)
	}
}

func TestRepartirComisiones_sinUpline(t *testing.T) {
	comisiones := RepartirComisiones(100, nil, TasasComision)

	if comisiones == nil {
		t.Fatal("debe devolver slice vacío, no nil (el JSON sería null)")
	}
	if len(comisiones) != 0 {
		t.Errorf("esperaba 0 comisiones, obtuvo %d", len(comisiones))
	}
}

// Una cadena más larga que las tasas no genera comisiones de más: el 4º nivel
// simplemente no existe en el negocio.
func TestRepartirComisiones_noExcedeLasTasas(t *testing.T) {
	comisiones := RepartirComisiones(1000, []int{5, 4, 3, 2, 1}, TasasComision)

	if len(comisiones) != len(TasasComision) {
		t.Fatalf("esperaba %d comisiones, obtuvo %d", len(TasasComision), len(comisiones))
	}
}

// El redondeo a 2 decimales es la diferencia entre un ledger que cuadra con lo
// que ve el usuario y uno que difiere un centavo por fila.
func TestRepartirComisiones_redondeaADosDecimales(t *testing.T) {
	comisiones := RepartirComisiones(33.33, []int{2, 1}, TasasComision)

	if comisiones[0].Monto != 3.33 {
		t.Errorf("nivel 1: esperaba 3.33, obtuvo %.4f", comisiones[0].Monto)
	}
	if comisiones[1].Monto != 1.67 {
		t.Errorf("nivel 2: esperaba 1.67, obtuvo %.4f", comisiones[1].Monto)
	}
}

func TestRepartirComisiones_totalCeroONegativo(t *testing.T) {
	for _, total := range []float64{0, -50} {
		comisiones := RepartirComisiones(total, []int{1, 2}, TasasComision)
		if len(comisiones) != 0 {
			t.Errorf("total=%v: esperaba 0 comisiones, obtuvo %d", total, len(comisiones))
		}
	}
}

// Una tasa en cero o negativa no debe generar una fila con monto 0.
func TestRepartirComisiones_ignoraTasasNoPositivas(t *testing.T) {
	comisiones := RepartirComisiones(100, []int{1, 2}, []float64{0.10, 0, -0.05})

	if len(comisiones) != 1 {
		t.Fatalf("esperaba 1 comisión válida, obtuvo %d", len(comisiones))
	}
	if comisiones[0].Monto != 10 {
		t.Errorf("esperaba 10.00, obtuvo %.2f", comisiones[0].Monto)
	}
}

func TestTotalComisiones(t *testing.T) {
	comisiones := RepartirComisiones(100, []int{2, 1, 0}, TasasComision)

	if len(comisiones) != 3 {
		t.Fatalf("esperaba 3 comisiones, obtuvo %d", len(comisiones))
	}
	// 10 + 5 + 2 = 17% del total
	if total := TotalComisiones(comisiones); total != 17 {
		t.Errorf("esperaba 17.00, obtuvo %.2f", total)
	}
}

// Guardarraíl de negocio: si alguien cambia TasasComision sin revisar
// MaxNivelesComision, la base de datos y el cálculo se desalinean.
func TestTasasComision_consistentesConMaxNiveles(t *testing.T) {
	if len(TasasComision) != MaxNivelesComision {
		t.Errorf("hay %d tasas pero MaxNivelesComision es %d: revisa el negocio",
			len(TasasComision), MaxNivelesComision)
	}
}
