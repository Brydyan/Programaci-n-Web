package dto_test

import (
	"strings"
	"testing"

	"multicatalogo-backend/internal/dto"
)

// hasFieldErr devuelve true si err contiene un error de validación
// para el campo fieldName. Esto es robusto ante la salida de
// validator.ValidationErrors (que es un slice) y ante strings sueltos.
func hasFieldErr(err error, fieldName string) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	// validator v10 formatea como "Key: 'DTP.field' Error:..."
	return strings.Contains(s, fieldName) || strings.Contains(s, "'"+fieldName+"'")
}

func TestLoginRequest(t *testing.T) {
	cases := []struct {
		name      string
		req       dto.LoginRequest
		wantErr   bool
		errField  string
	}{
		{"valid login", dto.LoginRequest{Email: "admin@upse.edu.ec", Password: "supersecreto"}, false, ""},
		{"empty email fails required", dto.LoginRequest{Email: "", Password: "supersecreto"}, true, "Email"},
		{"invalid email fails email", dto.LoginRequest{Email: "no-es-email", Password: "supersecreto"}, true, "Email"},
		{"short password fails min=8", dto.LoginRequest{Email: "admin@upse.edu.ec", Password: "abc"}, true, "Password"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := dto.Validate.Struct(tc.req)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected validation error, got nil")
				}
				if tc.errField != "" && !hasFieldErr(err, tc.errField) {
					t.Fatalf("expected error on %q, got: %v", tc.errField, err)
				}
			} else if err != nil {
				t.Fatalf("expected nil error, got: %v", err)
			}
		})
	}
}

func TestRegisterRequest(t *testing.T) {
	cases := []struct {
		name     string
		req      dto.RegisterRequest
		wantErr  bool
		errField string
	}{
		{
			"valid register",
			dto.RegisterRequest{Nombre: "Ana Pérez", Email: "ana@upse.edu.ec", Password: "supersecreto"},
			false, "",
		},
		{
			"invalid ref fails alphanum/len",
			dto.RegisterRequest{
				Nombre: "Ana Pérez", Email: "ana@upse.edu.ec", Password: "supersecreto",
				Ref: "abc!1",
			},
			true, "Ref",
		},
		{
			"omit ref is allowed (omitempty)",
			dto.RegisterRequest{Nombre: "Ana Pérez", Email: "ana@upse.edu.ec", Password: "supersecreto"},
			false, "",
		},
		{
			"empty nombre fails required",
			dto.RegisterRequest{Email: "ana@upse.edu.ec", Password: "supersecreto"},
			true, "Nombre",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := dto.Validate.Struct(tc.req)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected validation error, got nil")
				}
				if tc.errField != "" && !hasFieldErr(err, tc.errField) {
					t.Fatalf("expected error on %q, got: %v", tc.errField, err)
				}
			} else if err != nil {
				t.Fatalf("expected nil error, got: %v", err)
			}
		})
	}
}
