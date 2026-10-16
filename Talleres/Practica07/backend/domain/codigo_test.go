package domain

import (
	"testing"
)

func TestGenerarCodigoRef_LongitudYAlfabeto(t *testing.T) {
	seen := make(map[string]struct{}, 1000)
	for i := 0; i < 1000; i++ {
		codigo, err := GenerarCodigoRef()
		if err != nil {
			t.Fatalf("iteración %d: error inesperado: %v", i, err)
		}
		if len(codigo) != 6 {
			t.Fatalf("iteración %d: longitud esperada 6, got %d (%q)", i, len(codigo), codigo)
		}
		for j := 0; j < len(codigo); j++ {
			c := codigo[j]
			enAlfabeto := false
			for k := 0; k < len(codigoRefAlphabet); k++ {
				if codigoRefAlphabet[k] == c {
					enAlfabeto = true
					break
				}
			}
			if !enAlfabeto {
				t.Fatalf("iteración %d: carácter %q fuera del alfabeto permitido (%q)",
					i, c, codigo)
			}
		}
		seen[codigo] = struct{}{}
	}
	// Probabilidad de colisión en 1000 generaciones sobre un alfabeto de 32
	// chars con longitud 6 → espacio total 32^6 ≈ 1.07e9. Birthday paradox:
	// P(colisión) ≈ 1 - exp(-1000²/(2*32^6)) ≈ 4.6e-7. Asumimos 0 colisiones.
	if len(seen) != 1000 {
		t.Fatalf("se observaron colisiones en 1000 generaciones: %d códigos únicos", len(seen))
	}
}

func TestCodigoEsValido(t *testing.T) {
	cases := []struct {
		in   string
		want bool
		desc string
	}{
		// Válidos — acepta códigos seed preexistentes (LUIS contiene I)
		{"LUIS", true, "válido (seed legacy con I)"},
		{"ADMIN", true, "válido (seed legacy 5 chars)"},
		{"TU", true, "válido (seed legacy 2 chars)"},
		{"ABC234", true, "válido (alfabeto generado)"},
		{"ABCDEF", true, "válido (sólo letras)"},
		{"234567", true, "válido (sólo dígitos)"},
		{"ABC101", true, "1 permitido en input (sólo generado es estricto)"},
		{"ABCI01", true, "I y 0 permitidos en input"},
		{"ABCDEF0", true, "0 permitido en input"},
		// Inválidos
		{"", false, "vacío"},
		{"A", false, "demasiado corto"},
		{"ABCDEFGHI", false, "demasiado largo (9 chars)"},
		{"abcdef", false, "minúsculas"},
		{"AB-CDE", false, "contiene guion"},
		{"ABC DEF", false, "contiene espacio"},
		{"ÁÉÍÓÚ", false, "acentos"},
	}
	for _, tc := range cases {
		got := CodigoEsValido(tc.in)
		if got != tc.want {
			t.Errorf("CodigoEsValido(%q) = %v; want %v  (%s)", tc.in, got, tc.want, tc.desc)
		}
	}
}