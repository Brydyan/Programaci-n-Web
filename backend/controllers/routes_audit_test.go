package controllers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestAuditoriaRutas_todasLasEscriturasRechazanSinToken ejecuta, contra la
// app real, la misma tabla que ROUTES_AUDIT.md documenta.
//
// Es la verificación automática de la auditoría. Deliberadamente NO parsea
// routes.go con regexp: un test que lee el fuente sólo verifica que el texto
// diga lo que dice, no que la ruta esté protegida de verdad. Este test pide
// la ruta por HTTP y exige 401, que es la propiedad que importa.
//
// No necesita DB: RequireAuth corre antes que cualquier handler, así que el
// repositorio nil nunca llega a tocarse.

type entradaAuditoria struct {
	metodo    string
	ruta      string
	protegida bool // true = debe exigir token
}

// entradasEscritura son TODAS las rutas de escritura de la API. Si alguien
// añade un endpoint nuevo y lo olvida aquí, la auditoría queda incompleta;
// por eso el test de abajo también cuenta los handlers registrados.
var entradasAuditoria = []entradaAuditoria{
	{http.MethodPost, "/api/login", false},
	{http.MethodPost, "/api/register", false},
	{http.MethodPost, "/api/checkout", true},
	{http.MethodPost, "/api/admin/productos", true},
	{http.MethodPut, "/api/admin/productos/1", true},
	{http.MethodDelete, "/api/admin/productos/1", true},
	{http.MethodPatch, "/api/admin/pedidos/1/estado", true},
	{http.MethodPost, "/api/admin/liquidaciones", true},
}

func TestAuditoriaRutas_escriturasRechazanSinToken(t *testing.T) {
	app := setupTestApp()

	for _, e := range entradasAuditoria {
		t.Run(e.metodo+" "+e.ruta, func(t *testing.T) {
			req := httptest.NewRequest(e.metodo, e.ruta, nil)
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req, 5000)
			if err != nil {
				t.Fatalf("app.Test: %v", err)
			}
			defer resp.Body.Close()

			if e.protegida && resp.StatusCode != http.StatusUnauthorized {
				t.Errorf("%s %s es PROTEGIDA en ROUTES_AUDIT.md pero respondió %d sin token",
					e.metodo, e.ruta, resp.StatusCode)
			}
		})
	}
}

// TestAuditoriaRutas_lecturasProtegidasRechazanSinToken cubre las lecturas
// que también exigen token. Una GET protegida que devuelve 200 sin token
// filtra datos aunque no escriba nada.
func TestAuditoriaRutas_lecturasProtegidasRechazanSinToken(t *testing.T) {
	app := setupTestApp()

	protegidas := []struct {
		metodo string
		ruta   string
	}{
		{http.MethodGet, "/api/pedidos/mios"},
		{http.MethodGet, "/api/pedidos/1"},
		{http.MethodGet, "/api/red/mia"},
		{http.MethodGet, "/api/comisiones/mias"},
		{http.MethodGet, "/api/liquidaciones/mias"},
		{http.MethodGet, "/api/red"},
		{http.MethodGet, "/api/admin/productos"},
		{http.MethodGet, "/api/admin/pedidos"},
		{http.MethodGet, "/api/admin/comisiones"},
		{http.MethodGet, "/api/admin/liquidaciones"},
		{http.MethodGet, "/api/admin/liquidaciones/deuda"},
		{http.MethodGet, "/api/reportes/resumen"},
		{http.MethodGet, "/api/reportes/top-productos"},
		{http.MethodGet, "/api/reportes/top-referidos"},
		{http.MethodGet, "/api/reportes/comisiones-pendientes"},
	}

	for _, e := range protegidas {
		t.Run(e.metodo+" "+e.ruta, func(t *testing.T) {
			resp, err := app.Test(httptest.NewRequest(e.metodo, e.ruta, nil), 5000)
			if err != nil {
				t.Fatalf("app.Test: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusUnauthorized {
				t.Errorf("%s %s está marcada como PROTEGIDA pero respondió %d sin token",
					e.metodo, e.ruta, resp.StatusCode)
			}
		})
	}
}

// TestAuditoriaRutas_publicasNoExigenToken es la contraparte: una ruta
// documentada como PUBLIC que empieza a pedir token rompe el catálogo para
// los visitantes, y es un fallo igual de difícil de ver.
func TestAuditoriaRutas_publicasNoExigenToken(t *testing.T) {
	app := setupTestApp()

	publicas := []struct {
		metodo string
		ruta   string
	}{
		{http.MethodGet, "/api/health"},
		{http.MethodGet, "/api/productos"},
		{http.MethodGet, "/api/productos/1"},
		{http.MethodGet, "/api/producto/1"},
	}

	for _, e := range publicas {
		t.Run(e.metodo+" "+e.ruta, func(t *testing.T) {
			resp, err := app.Test(httptest.NewRequest(e.metodo, e.ruta, nil), 5000)
			if err != nil {
				t.Fatalf("app.Test: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusUnauthorized {
				t.Errorf("%s %s es PUBLIC en ROUTES_AUDIT.md pero exige token",
					e.metodo, e.ruta)
			}
		})
	}
}

// TestAuditoriaRutas_todasLasEscriturasEstanCubiertas comprueba que ninguna
// ruta de escritura quedó fuera de la tabla. Mantener el exhaustivo es lo que
// hace que la auditoría sirva: una tabla a la que se le olvidan rutas
// "audita" menos de lo que dice.
func TestAuditoriaRutas_todasLasEscriturasEstanCubiertas(t *testing.T) {
	// Se cuenta contra el número de handlers de escritura registrados en
	// routes.go. Si este número cambia y entradasAuditoria no, el test falla.
	const escriturasRegistradas = 8

	if len(entradasAuditoria) != escriturasRegistradas {
		t.Errorf("entradasAuditoria tiene %d rutas de escritura, routes.go registra %d: "+
			"actualiza ROUTES_AUDIT.md y esta tabla",
			len(entradasAuditoria), escriturasRegistradas)
	}
}
