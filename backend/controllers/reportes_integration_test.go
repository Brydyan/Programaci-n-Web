package controllers_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"

	"multicatalogo-backend/config"
	"multicatalogo-backend/models"
	"multicatalogo-backend/repository"
)

// Tests de integración de /api/reportes/*.
//
// Pegan contra la DB real porque los cuatro endpoints son agregaciones: lo
// que importa es que las cifras cuadren entre sí (que un LEFT JOIN no deje
// fuera un referido, que el filtro de pendientes no seepa de más), y eso no se
// puede verificar con un mock. Sin DB los tests hacen skip — es el mismo
// convenio que el resto del paquete.

// pedirReporte dispara un GET autenticado y devuelve el código y el cuerpo.
func pedirReporte(t *testing.T, app *fiber.App, token, url string) (int, []byte) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, url, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("app.Test(%s): %v", url, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, body
}

// TestReportes_requiereAdmin verifica el control de acceso de los cuatro
// endpoints. Es la prueba que más importa: un endpoint de reportes sin
// RequireAdmin publica la facturación de la organización a cualquier
// cliente con un token válido.
func TestReportes_requiereAdmin(t *testing.T) {
	db, err := sql.Open("postgres", dsnTest())
	if err != nil {
		t.Skipf("DB no disponible: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Skipf("DB ping fail: %v", err)
	}
	defer db.Close()
	config.DB = db
	ensureSeed(t, db)

	app := appConRepo(repository.NewRepository(db), &MockCacheService{})
	tokenCliente := firmarTokenRol(t, 1, "cliente@upse.edu.ec", "cliente")

	rutas := []string{
		"/api/reportes/resumen",
		"/api/reportes/top-productos",
		"/api/reportes/top-referidos",
		"/api/reportes/comisiones-pendientes",
	}
	for _, ruta := range rutas {
		t.Run("cliente recibe 403 en "+ruta, func(t *testing.T) {
			code, body := pedirReporte(t, app, tokenCliente, ruta)
			if code != http.StatusForbidden {
				t.Errorf("%s: esperaba 403, obtuve %d (%s)", ruta, code, body)
			}
		})
		t.Run("anónimo recibe 401 en "+ruta, func(t *testing.T) {
			code, body := pedirReporte(t, app, "", ruta)
			if code != http.StatusUnauthorized {
				t.Errorf("%s: esperaba 401, obtuve %d (%s)", ruta, code, body)
			}
		})
	}
}

// TestReportes_resumenValidaFechas comprueba el 400 de un filtro mal formado.
// Un `?desde=15-01-2026` que se ignorara en silencio devolvería el resumen
// global y el admin creería que está viendo un trimestre.
func TestReportes_resumenValidaFechas(t *testing.T) {
	db, err := sql.Open("postgres", dsnTest())
	if err != nil {
		t.Skipf("DB no disponible: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Skipf("DB ping fail: %v", err)
	}
	defer db.Close()
	config.DB = db
	ensureSeed(t, db)

	app := appConRepo(repository.NewRepository(db), &MockCacheService{})
	token := firmarTokenRol(t, 1, "admin@upse.edu.ec", "admin")

	malos := []string{
		"?desde=15-01-2026",
		"?hasta=2026-13-01",
		"?desde=ayer",
		"?desde=2026-12-31&hasta=2026-01-01",
	}
	for _, q := range malos {
		t.Run("400 con "+q, func(t *testing.T) {
			code, body := pedirReporte(t, app, token, "/api/reportes/resumen"+q)
			if code != http.StatusBadRequest {
				t.Errorf("esperaba 400 para %q, obtuve %d (%s)", q, code, body)
			}
		})
	}
}

// TestReportes_topProductosRespetaLimit comprueba el 400 por `limit` fuera de
// rango y el corte real del resultado.
func TestReportes_topProductosRespetaLimit(t *testing.T) {
	db, err := sql.Open("postgres", dsnTest())
	if err != nil {
		t.Skipf("DB no disponible: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Skipf("DB ping fail: %v", err)
	}
	defer db.Close()
	config.DB = db
	ensureSeed(t, db)

	app := appConRepo(repository.NewRepository(db), &MockCacheService{})
	token := firmarTokenRol(t, 1, "admin@upse.edu.ec", "admin")

	for _, q := range []string{"?limit=0", "?limit=51", "?limit=abc"} {
		t.Run("400 con "+q, func(t *testing.T) {
			code, body := pedirReporte(t, app, token, "/api/reportes/top-productos"+q)
			if code != http.StatusBadRequest {
				t.Errorf("esperaba 400 para %q, obtuve %d (%s)", q, code, body)
			}
		})
	}

	t.Run("limit=2 devuelve como máximo 2 filas", func(t *testing.T) {
		code, body := pedirReporte(t, app, token, "/api/reportes/top-productos?limit=2")
		if code != http.StatusOK {
			t.Fatalf("esperaba 200, obtuve %d (%s)", code, body)
		}
		var top []models.TopProducto
		if err := json.Unmarshal(body, &top); err != nil {
			t.Fatalf("respuesta no es un array de TopProducto: %v (%s)", err, body)
		}
		if len(top) > 2 {
			t.Errorf("limit=2 devolvió %d filas", len(top))
		}
		// El orden del contrato es por ingresos descendentes.
		for i := 1; i < len(top); i++ {
			if top[i].Ingresos > top[i-1].Ingresos {
				t.Errorf("fila %d (%.2f) supera a la anterior (%.2f): no está ordenado",
					i, top[i].Ingresos, top[i-1].Ingresos)
			}
		}
	})
}

// TestReportes_contratoSnakeCase blinda los nombres de campo del JSON.
//
// El frontend ya tiene sus tipos (types/api.ts) y el contrato está en
// openapi.yaml: si un campo pasa a camelCase, TypeScript no lo detecta
// porque el fetch devuelve `any` en la práctica, y el dashboard muestra
// "undefined" en vez de fallar.
func TestReportes_contratoSnakeCase(t *testing.T) {
	db, err := sql.Open("postgres", dsnTest())
	if err != nil {
		t.Skipf("DB no disponible: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Skipf("DB ping fail: %v", err)
	}
	defer db.Close()
	config.DB = db
	ensureSeed(t, db)

	app := appConRepo(repository.NewRepository(db), &MockCacheService{})
	token := firmarTokenRol(t, 1, "admin@upse.edu.ec", "admin")

	code, body := pedirReporte(t, app, token, "/api/reportes/resumen")
	if code != http.StatusOK {
		t.Fatalf("esperaba 200, obtuve %d (%s)", code, body)
	}

	var crudo map[string]json.RawMessage
	if err := json.Unmarshal(body, &crudo); err != nil {
		t.Fatalf("respuesta no es un objeto: %v", err)
	}

	esperados := []string{
		"total_vendido", "num_pedidos", "ticket_promedio", "comisiones_devengadas",
	}
	for _, campo := range esperados {
		if _, ok := crudo[campo]; !ok {
			t.Errorf("falta el campo %q en %s", campo, body)
		}
	}
	forbidden := []string{"totalVentas", "numPedidos", "ticketPromedio", "comisionesDevengadas"}
	for _, campo := range forbidden {
		if _, ok := crudo[campo]; ok {
			t.Errorf("apareció %q en camelCase: el contrato es snake_case (%s)", campo, body)
		}
	}

	// El JSON debe deserializar en el struct de models sin pérdida.
	var resumen models.ReporteResumen
	if err := json.Unmarshal(body, &resumen); err != nil {
		t.Fatalf("no deserializa en models.ReporteResumen: %v", err)
	}
	if resumen.NumPedidos < 0 || resumen.TicketPromedio < 0 {
		t.Errorf("agregados negativos: %+v", resumen)
	}
}

// TestReportes_comisionesPendientesSoloPendientes verifica el filtro: la
// pantalla de trabajo del admin antes de liquidar no debe mostrar comisiones
// ya pagadas o revertidas.
func TestReportes_comisionesPendientesSoloPendientes(t *testing.T) {
	db, err := sql.Open("postgres", dsnTest())
	if err != nil {
		t.Skipf("DB no disponible: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Skipf("DB ping fail: %v", err)
	}
	defer db.Close()
	config.DB = db
	ensureSeed(t, db)

	app := appConRepo(repository.NewRepository(db), &MockCacheService{})
	token := firmarTokenRol(t, 1, "admin@upse.edu.ec", "admin")

	code, body := pedirReporte(t, app, token, "/api/reportes/comisiones-pendientes?limit=50")
	if code != http.StatusOK {
		t.Fatalf("esperaba 200, obtuve %d (%s)", code, body)
	}
	var lista []models.Comision
	if err := json.Unmarshal(body, &lista); err != nil {
		t.Fatalf("respuesta no es un array de Comision: %v (%s)", err, body)
	}
	for i, c := range lista {
		if c.Estado != "pendiente" {
			t.Errorf("fila %d tiene estado %q; el endpoint sólo debe devolver 'pendiente'",
				i, c.Estado)
		}
	}

	// Cuadre contra la DB: el número de filas servidas no puede exceder al
	// de las comisiones realmente pendientes.
	var totalPendientes int
	if err := db.QueryRow(
		`SELECT count(*) FROM comisiones WHERE estado = 'pendiente'`,
	).Scan(&totalPendientes); err != nil {
		t.Fatalf("contando pendientes: %v", err)
	}
	if len(lista) > totalPendientes {
		t.Errorf("serví %d filas pero sólo hay %d pendientes", len(lista), totalPendientes)
	}
}

// TestReportes_topReferidosConservaVentasDelContador documenta que `ventas`
// viene del contador referidos.ventas y no del ledger, y que un referido sin
// comisiones NO desaparece del ranking (LEFT JOIN, no INNER).
func TestReportes_topReferidosConservaVentasDelContador(t *testing.T) {
	db, err := sql.Open("postgres", dsnTest())
	if err != nil {
		t.Skipf("DB no disponible: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Skipf("DB ping fail: %v", err)
	}
	defer db.Close()
	config.DB = db
	ensureSeed(t, db)

	// Un referido sin usuario ni comisiones: sólo existe para probar el
	// LEFT JOIN.
	var referidoID int
	if err := db.QueryRow(
		`INSERT INTO referidos (codigo, ventas) VALUES ($1, $2) RETURNING id`,
		fmt.Sprintf("TEST%d", time.Now().UnixNano()%100000),
		999.99,
	).Scan(&referidoID); err != nil {
		t.Skipf("no se pudo crear el referido de prueba: %v", err)
	}
	defer func() {
		_, _ = db.Exec(`DELETE FROM referidos WHERE id = $1`, referidoID)
	}()

	app := appConRepo(repository.NewRepository(db), &MockCacheService{})
	token := firmarTokenRol(t, 1, "admin@upse.edu.ec", "admin")

	code, body := pedirReporte(t, app, token, "/api/reportes/top-referidos?limit=50")
	if code != http.StatusOK {
		t.Fatalf("esperaba 200, obtuve %d (%s)", code, body)
	}
	var top []models.TopReferido
	if err := json.Unmarshal(body, &top); err != nil {
		t.Fatalf("respuesta no es un array de TopReferido: %v (%s)", err, body)
	}

	var encontrado bool
	for _, r := range top {
		if r.ReferidoID != referidoID {
			continue
		}
		encontrado = true
		if r.Ventas != 999.99 {
			t.Errorf("ventas = %.2f, esperaba 999.99 (viene de referidos.ventas)", r.Ventas)
		}
		if r.Comisiones != 0 {
			t.Errorf("comisiones = %.2f, esperaba 0 (sin ledger)", r.Comisiones)
		}
		if r.Codigo == "" {
			t.Error("codigo vacío: el LEFT JOIN a usuarios no debe vaciar el código")
		}
	}
	if !encontrado {
		t.Error("un referido sin comisiones desapareció del ranking: se está usando INNER JOIN")
	}
}
