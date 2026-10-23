// Package domain contiene lógica de negocio pura (sin I/O ni dependencias
// de frameworks). Su propósito es encapsular reglas y utilidades que se
// comparten entre capas (controllers, repository, etc.) y que pueden
// probarse en aislamiento.
//
// codigo.go implementa la generación y validación de códigos alfanuméricos
// cortos usados como códigos de referido.
package domain

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

const (
	// codigoRefLength es la longitud fija de los códigos NUEVOS que genera
	// GenerarCodigoRef. Los códigos seed preexistentes (TU, LUIS, ADMIN)
	// tienen longitudes distintas — la validación es laxa con la longitud.
	codigoRefLength = 6

	// codigoRefAlphabet es el alfabeto usado por GenerarCodigoRef. Excluye
	// caracteres que se confunden visualmente entre sí (I/O/0/1) para que
	// un código se pueda transcribir sin ambigüedad.
	codigoRefAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
)

// GenerarCodigoRef genera un código alfanumérico aleatorio de longitud
// codigoRefLength usando crypto/rand (resistente a predicción, a diferencia
// de math/rand). Cada posición se sortea con un big.Int en el rango
// [0, len(alfabeto)) mediante rand.Int, que internamente usa el lector
// criptográfico del sistema operativo.
//
// Devuelve un error sólo si el lector criptográfico falla (caso extremo).
func GenerarCodigoRef() (string, error) {
	out := make([]byte, codigoRefLength)
	max := big.NewInt(int64(len(codigoRefAlphabet)))
	for i := 0; i < codigoRefLength; i++ {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", fmt.Errorf("error generando código de referido: %w", err)
		}
		out[i] = codigoRefAlphabet[n.Int64()]
	}
	return string(out), nil
}

// CodigoEsValido valida que el código sea alfanumérico uppercase de 2 a 8
// caracteres. Es laxo con el alfabeto (permite I/O/0/1) para aceptar códigos
// seed preexistentes como "TU", "ADMIN", "LUIS". Los códigos NUEVOS los
// genera GenerarCodigoRef desde el alfabeto estricto para máxima claridad
// al transcribirlos.
//
// El controller ya pasa el código por strings.ToUpper antes de llegar aquí.
func CodigoEsValido(codigo string) bool {
	if len(codigo) < 2 || len(codigo) > 8 {
		return false
	}
	for i := 0; i < len(codigo); i++ {
		c := codigo[i]
		ok := (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
		if !ok {
			return false
		}
	}
	return true
}
