package controllers_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"multicatalogo-backend/internal/dto"
	"multicatalogo-backend/models"
)

// Tests de T4: validación del checkout contra
// openapi.yaml → components.schemas.CheckoutRequest.
//
// El requisito (`required: [direccion, ciudad, telefono, metodo_pago,
// items]`) existía en el contrato desde antes de que la tabla `pedidos`
// tuviera esas columnas. Estos tests fijan que ahora sí se exige.

// Los cuerpos de prueba se escriben enteros y explícitos en vez de derivarse
// quitando campos: partir el JSON con Split rómpido el array `items`, que
// también contiene comas y comillas, y el fallo aparecería como un 400
// espurio que parece un bug del servidor.

func TestCheckout_validacionCamposObligatorios(t *testing.T) {
	token := firmarToken(t, 5, "comprador@x.com")

	repo := &fakeRepo{
		CreatePedidoConUsuarioFn: func(dto.CheckoutRequest, int) (*models.PedidoDetalle, error) {
			t.Error("un body inválido no debe crear un pedido")
			return nil, nil
		},
	}
	app := appConRepo(repo, &MockCacheService{})

	items := `"items":[{"id":1,"nombre":"Serum","precio":45,"cantidad":2}]`
	casos := []struct {
		nombre string
		body   string
	}{
		{"sin direccion", `{"ciudad":"Quito","telefono":"0987654321",` +
			`"metodo_pago":"tarjeta",` + items + `}`},
		{"sin ciudad", `{"direccion":"Av. 10 de Agosto 100","telefono":"0987654321",` +
			`"metodo_pago":"tarjeta",` + items + `}`},
		{"sin telefono", `{"direccion":"Av. 10 de Agosto 100","ciudad":"Quito",` +
			`"metodo_pago":"tarjeta",` + items + `}`},
		{"sin metodo_pago", `{"direccion":"Av. 10 de Agosto 100","ciudad":"Quito",` +
			`"telefono":"0987654321",` + items + `}`},
		{"carrito vacío", `{"direccion":"Av. 10 de Agosto 100","ciudad":"Quito",` +
			`"telefono":"0987654321","metodo_pago":"tarjeta","items":[]}`},
		{"sin items", `{"direccion":"Av. 10 de Agosto 100","ciudad":"Quito",` +
			`"telefono":"0987654321","metodo_pago":"tarjeta"}`},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			code, body := pedirJSON(t, app, http.MethodPost, token, "/api/checkout", c.body)
			if code != http.StatusBadRequest {
				t.Fatalf("esperaba 400, obtuve %d (%s)", code, body)
			}
			var ae apiErr
			_ = json.Unmarshal(body, &ae)
			if ae.Code != "VALIDATION_ERROR" {
				t.Errorf("código = %q, esperaba VALIDATION_ERROR", ae.Code)
			}
		})
	}
}

func TestCheckout_validacionDeLongitudes(t *testing.T) {
	token := firmarToken(t, 5, "comprador@x.com")
	repo := &fakeRepo{
		CreatePedidoConUsuarioFn: func(dto.CheckoutRequest, int) (*models.PedidoDetalle, error) {
			t.Error("un body inválido no debe crear un pedido")
			return nil, nil
		},
	}
	app := appConRepo(repo, &MockCacheService{})

	casos := []struct {
		nombre string
		body   string
	}{
		{"dirección demasiado corta", `{"direccion":"a","ciudad":"Quito",` +
			`"telefono":"0987654321","metodo_pago":"tarjeta",` +
			`"items":[{"id":1,"nombre":"S","precio":1,"cantidad":1}]}`},
		{"ciudad demasiado corta", `{"direccion":"Av. 10 de Agosto 100","ciudad":"Q",` +
			`"telefono":"0987654321","metodo_pago":"tarjeta",` +
			`"items":[{"id":1,"nombre":"S","precio":1,"cantidad":1}]}`},
		{"teléfono demasiado corto", `{"direccion":"Av. 10 de Agosto 100","ciudad":"Quito",` +
			`"telefono":"123","metodo_pago":"tarjeta",` +
			`"items":[{"id":1,"nombre":"S","precio":1,"cantidad":1}]}`},
		{"cantidad cero", `{"direccion":"Av. 10 de Agosto 100","ciudad":"Quito",` +
			`"telefono":"0987654321","metodo_pago":"tarjeta",` +
			`"items":[{"id":1,"nombre":"S","precio":1,"cantidad":0}]}`},
		{"id de producto cero", `{"direccion":"Av. 10 de Agosto 100","ciudad":"Quito",` +
			`"telefono":"0987654321","metodo_pago":"tarjeta",` +
			`"items":[{"id":0,"nombre":"S","precio":1,"cantidad":1}]}`},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			code, _ := pedirJSON(t, app, http.MethodPost, token, "/api/checkout", c.body)
			if code != http.StatusBadRequest {
				t.Errorf("esperaba 400, obtuve %d", code)
			}
		})
	}
}

// TestCheckout_detallesDeValidacionLegibles comprueba que el 400 dice QUÉ
// campo falta, no sólo que "algo está mal". Sin esto el usuario ve un 400 y no
// sabe qué corregir.
func TestCheckout_detallesDeValidacionLegibles(t *testing.T) {
	// middleware.RespondError omite `details` salvo con APP_ENV=dev, para no
	// filtrar internals en producción. Sin esto el detalle nunca viajaría y
	// el test no probaría nada.
	t.Setenv("APP_ENV", "dev")

	token := firmarToken(t, 5, "comprador@x.com")
	app := appConRepo(&fakeRepo{}, &MockCacheService{})

	body := `{"direccion":"Av. 10 de Agosto 100","telefono":"0987654321",` +
		`"metodo_pago":"tarjeta",` +
		`"items":[{"id":1,"nombre":"Serum","precio":45,"cantidad":2}]}`
	code, resp := pedirJSON(t, app, http.MethodPost, token, "/api/checkout", body)
	if code != http.StatusBadRequest {
		t.Fatalf("esperaba 400, obtuve %d", code)
	}
	// `details` sólo se serializa con APP_ENV=dev, así que se busca en el
	// mensaje o en el detalle: lo que importa es que nombre el campo.
	if !strings.Contains(strings.ToLower(string(resp)), "ciudad") {
		t.Errorf("el detalle no nombra el campo faltante: %s", resp)
	}
}

// TestCheckout_identidadVieneDelJWT es la prueba de seguridad del cambio:
// aunque el cliente mande un email distinto en el body, el pedido se guarda a
// nombre de quien está autenticado.
func TestCheckout_identidadVieneDelJWT(t *testing.T) {
	token := firmarToken(t, 42, "autenticado@x.com")
	var emailGuardado string
	repo := &fakeRepo{
		CreatePedidoConUsuarioFn: func(req dto.CheckoutRequest, _ int) (*models.PedidoDetalle, error) {
			emailGuardado = req.EmailUsuario
			return &models.PedidoDetalle{ID: 1, Total: 90}, nil
		},
	}
	app := appConRepo(repo, &MockCacheService{})

	body := `{"nombre":"Yo","email":"victima@otra.com","direccion":"Av. 10 de Agosto 100",` +
		`"ciudad":"Quito","telefono":"0987654321","metodo_pago":"tarjeta",` +
		`"items":[{"id":1,"nombre":"S","precio":45,"cantidad":2}]}`
	code, respBody := pedirJSON(t, app, http.MethodPost, token, "/api/checkout", body)
	if code != http.StatusCreated {
		t.Fatalf("esperaba 201, obtuve %d (%s)", code, respBody)
	}

	if emailGuardado != "autenticado@x.com" {
		t.Errorf("el pedido se guardó a nombre de %q; debe ser el del token (%q). "+
			"El email del body fue aceptado → escalada de privilegios",
			emailGuardado, "autenticado@x.com")
	}
}

// TestCheckout_anonymousNoCompra comprueba que la validación no se puede
// esquivar sin token.
func TestCheckout_anonymousNoCompra(t *testing.T) {
	valido := `{"direccion":"Av. 10 de Agosto 100","ciudad":"Quito",` +
		`"telefono":"0987654321","metodo_pago":"tarjeta",` +
		`"items":[{"id":1,"nombre":"Serum","precio":45,"cantidad":2}]}`
	app := appConRepo(&fakeRepo{}, &MockCacheService{})
	code, _ := pedirJSON(t, app, http.MethodPost, "", "/api/checkout", valido)
	if code != http.StatusUnauthorized {
		t.Errorf("esperaba 401, obtuve %d", code)
	}
}
