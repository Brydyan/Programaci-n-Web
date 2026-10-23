package controllers_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"

	"multicatalogo-backend/domain"
	"multicatalogo-backend/interfaces"
	"multicatalogo-backend/models"
)

// Tests de controllers que NO necesitan PostgreSQL: usan fakeRepo (ver
// fakeRepo_test.go) y recordingCache (ver controllers_test.go).
//
// Qué se cubre aquí y por qué importa más que un 80% de cobertura decorativo:
// el código que traduce query-params, decide códigos HTTP y decide cuándo
// cachear. El SQL ya lo cubre la suite de integración cuando hay DB; esto
// corre siempre.

// ── Helpers ─────────────────────────────────────────────────────────────────

// paginaRespuesta es la forma de {data, page, limit, total} que devuelven los
// endpoints paginados. No hay un tipo en models porque se arma con fiber.Map.
type paginaRespuesta struct {
	Data  json.RawMessage `json:"data"`
	Page  int             `json:"page"`
	Limit int             `json:"limit"`
	Total int             `json:"total"`
}

func pedirJSON(
	t *testing.T, app *fiber.App, method, token, url, body string,
) (int, []byte) {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, url, nil)
	} else {
		req = httptest.NewRequest(method, url, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("app.Test(%s %s): %v", method, url, err)
	}
	defer resp.Body.Close()
	buf, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, buf
}

// ── Reportes ────────────────────────────────────────────────────────────────

func TestReportesController_devuelveDatosDelRepo(t *testing.T) {
	token := firmarTokenRol(t, 1, "admin@upse.edu.ec", "admin")

	repo := &fakeRepo{
		GetReporteResumenFn: func(f interfaces.ReporteFiltro) (*models.ReporteResumen, error) {
			return &models.ReporteResumen{
				TotalVendido: 1500.50, NumPedidos: 3,
				TicketPromedio: 500.16, ComisionesDevengadas: 150.05,
			}, nil
		},
		GetTopProductosFn: func(limit int) ([]models.TopProducto, error) {
			if limit != 10 {
				return nil, fmt.Errorf("esperaba limit=10 por defecto, llegó %d", limit)
			}
			return []models.TopProducto{
				{ProductoID: 1, Nombre: "Shampoo", Unidades: 4, Ingresos: 200},
			}, nil
		},
		GetTopReferidosFn: func(limit int) ([]models.TopReferido, error) {
			return []models.TopReferido{
				{ReferidoID: 2, Codigo: "AB12", Nombre: "Ana", Ventas: 900, Comisiones: 90},
			}, nil
		},
		GetComisionesPendientesFn: func(limit int) ([]models.Comision, error) {
			// Sin ?limit= en la URL el controller aplica el default 10.
			if limit != 10 {
				return nil, fmt.Errorf("esperaba limit=10 por defecto, llegó %d", limit)
			}
			return []models.Comision{
				{ID: 1, Nivel: 1, Porcentaje: 10, Monto: 50, Estado: "pendiente"},
			}, nil
		},
	}
	app := appConRepo(repo, &MockCacheService{})

	code, body := pedirJSON(t, app, http.MethodGet, token, "/api/reportes/resumen", "")
	if code != http.StatusOK {
		t.Fatalf("resumen: esperaba 200, obtuve %d (%s)", code, body)
	}
	var resumen models.ReporteResumen
	if err := json.Unmarshal(body, &resumen); err != nil {
		t.Fatalf("resumen no deserializa: %v (%s)", err, body)
	}
	if resumen.TotalVendido != 1500.50 || resumen.NumPedidos != 3 {
		t.Errorf("KPIs no coinciden: %+v", resumen)
	}

	code, body = pedirJSON(t, app, http.MethodGet, token, "/api/reportes/top-productos", "")
	if code != http.StatusOK {
		t.Fatalf("top-productos: esperaba 200, obtuve %d (%s)", code, body)
	}
	var prods []models.TopProducto
	if err := json.Unmarshal(body, &prods); err != nil {
		t.Fatalf("top-productos no deserializa: %v (%s)", err, body)
	}
	if len(prods) != 1 || prods[0].Nombre != "Shampoo" {
		t.Errorf("ranking inesperado: %+v", prods)
	}

	code, body = pedirJSON(t, app, http.MethodGet, token, "/api/reportes/top-referidos", "")
	if code != http.StatusOK {
		t.Fatalf("top-referidos: esperaba 200, obtuve %d (%s)", code, body)
	}
	var ref []models.TopReferido
	if err := json.Unmarshal(body, &ref); err != nil {
		t.Fatalf("top-referidos no deserializa: %v (%s)", err, body)
	}
	if len(ref) != 1 || ref[0].Codigo != "AB12" {
		t.Errorf("ranking inesperado: %+v", ref)
	}

	code, _ = pedirJSON(t, app, http.MethodGet, token, "/api/reportes/comisiones-pendientes", "")
	if code != http.StatusOK {
		t.Errorf("comisiones-pendientes: esperaba 200, obtuve %d", code)
	}
}

// TestReportesController_pasaElFiltroDePeriodo verifica que `desde`/`hasta`
// llegan al repositorio con la forma correcta. Un periodo descartado en
// silencio devuelveía "todo el histórico" con apariencia de estar filtrado.
func TestReportesController_pasaElFiltroDePeriodo(t *testing.T) {
	token := firmarTokenRol(t, 1, "admin@upse.edu.ec", "admin")

	var recibido interfaces.ReporteFiltro
	repo := &fakeRepo{
		GetReporteResumenFn: func(f interfaces.ReporteFiltro) (*models.ReporteResumen, error) {
			recibido = f
			return &models.ReporteResumen{}, nil
		},
	}
	app := appConRepo(repo, &MockCacheService{})

	code, body := pedirJSON(t, app, http.MethodGet, token,
		"/api/reportes/resumen?desde=2026-01-01&hasta=2026-03-31", "")
	if code != http.StatusOK {
		t.Fatalf("esperaba 200, obtuve %d (%s)", code, body)
	}

	if recibido.Desde == nil || recibido.Desde.Format("2006-01-02") != "2026-01-01" {
		t.Errorf("Desde = %v, esperaba 2026-01-01", recibido.Desde)
	}
	// hasta es inclusivo para el usuario → la cota llega como día siguiente.
	if recibido.Hasta == nil || recibido.Hasta.Format("2006-01-02") != "2026-04-01" {
		t.Errorf("Hasta = %v, esperaba 2026-04-01 (día siguiente al inclusivo)", recibido.Hasta)
	}
}

func TestReportesController_repoFalla_devuelve500(t *testing.T) {
	token := firmarTokenRol(t, 1, "admin@upse.edu.ec", "admin")
	repo := &fakeRepo{
		GetReporteResumenFn: func(interfaces.ReporteFiltro) (*models.ReporteResumen, error) {
			return nil, errors.New("boom")
		},
	}
	app := appConRepo(repo, &MockCacheService{})

	code, body := pedirJSON(t, app, http.MethodGet, token, "/api/reportes/resumen", "")
	if code != http.StatusInternalServerError {
		t.Fatalf("esperaba 500, obtuve %d (%s)", code, body)
	}
	var ae apiErr
	_ = json.Unmarshal(body, &ae)
	if ae.Code != "INTERNAL" {
		t.Errorf("código de error = %q, esperaba INTERNAL", ae.Code)
	}
	// El mensaje al cliente nunca debe ser el error crudo de infraestructura.
	if strings.Contains(ae.Message, "boom") {
		t.Errorf("se filtró el detalle interno al cliente: %q", ae.Message)
	}
}

// TestReportesController_cacheHitNoVuelveALaBase es el test que justifica el
// caché: la segunda llamada idéntica debe servirse desde Redis sin tocar el
// repositorio.
func TestReportesController_cacheHitNoVuelveALaBase(t *testing.T) {
	token := firmarTokenRol(t, 1, "admin@upse.edu.ec", "admin")
	cache := newRecordingCache()

	var llamadas int
	repo := &fakeRepo{
		GetTopProductosFn: func(limit int) ([]models.TopProducto, error) {
			llamadas++
			return []models.TopProducto{
				{ProductoID: 1, Nombre: "Shampoo", Unidades: 2, Ingresos: 30},
			}, nil
		},
	}
	app := appConRepo(repo, cache)

	code, _ := pedirJSON(t, app, http.MethodGet, token, "/api/reportes/top-productos", "")
	if code != http.StatusOK {
		t.Fatalf("primera llamada: esperaba 200, obtuve %d", code)
	}
	code, body := pedirJSON(t, app, http.MethodGet, token, "/api/reportes/top-productos", "")
	if code != http.StatusOK {
		t.Fatalf("segunda llamada: esperaba 200, obtuve %d", code)
	}

	if llamadas != 1 {
		t.Errorf("el repositorio se consultó %d veces; la segunda debía servirse del caché", llamadas)
	}
	if !strings.Contains(string(body), "Shampoo") {
		t.Errorf("la respuesta cacheada perdió el dato: %s", body)
	}
}

// TestReportesController_distintoLimitNoComparteClave protege contra el bug
// clásico: cachear `?limit=10` y servirlo para `?limit=2`.
func TestReportesController_distintoLimitNoComparteClave(t *testing.T) {
	token := firmarTokenRol(t, 1, "admin@upse.edu.ec", "admin")
	cache := newRecordingCache()

	repo := &fakeRepo{
		GetTopProductosFn: func(limit int) ([]models.TopProducto, error) {
			filas := make([]models.TopProducto, limit)
			for i := range filas {
				filas[i] = models.TopProducto{ProductoID: i + 1, Nombre: fmt.Sprintf("P%d", i)}
			}
			return filas, nil
		},
	}
	app := appConRepo(repo, cache)

	pedirJSON(t, app, http.MethodGet, token, "/api/reportes/top-productos?limit=2", "")

	_, body := pedirJSON(t, app, http.MethodGet, token, "/api/reportes/top-productos?limit=5", "")
	var filas []models.TopProducto
	if err := json.Unmarshal(body, &filas); err != nil {
		t.Fatalf("no deserializa: %v (%s)", err, body)
	}
	if len(filas) != 5 {
		t.Errorf("limit=5 devolvió %d filas: la clave de caché no incluye el limit", len(filas))
	}
}

// TestReportesController_redisCaidoSirveDePostgres comprueba la degradación
// silenciosa: si Redis falla, la API responde igual desde la base.
func TestReportesController_redisCaidoSirveDePostgres(t *testing.T) {
	token := firmarTokenRol(t, 1, "admin@upse.edu.ec", "admin")
	repo := &fakeRepo{
		GetTopReferidosFn: func(int) ([]models.TopReferido, error) {
			return []models.TopReferido{
				{ReferidoID: 1, Codigo: "ZZ", Nombre: "Z", Ventas: 10, Comisiones: 1},
			}, nil
		},
	}
	// MockCacheService devuelve error en Get: es el caso "Redis caído".
	app := appConRepo(repo, &MockCacheService{})

	code, body := pedirJSON(t, app, http.MethodGet, token, "/api/reportes/top-referidos", "")
	if code != http.StatusOK {
		t.Fatalf("con Redis caído esperaba 200 desde Postgres, obtuve %d (%s)", code, body)
	}
}

// ── Comisiones ──────────────────────────────────────────────────────────────

func TestComisionesController_filtroYErrores(t *testing.T) {
	t.Run("estado inválido da 400 sin tocar la base", func(t *testing.T) {
		token := firmarTokenRol(t, 1, "admin@upse.edu.ec", "admin")
		llamadas := 0
		repo := &fakeRepo{
			GetComisionesFn: func(interfaces.ComisionFiltro) ([]models.Comision, int, error) {
				llamadas++
				return nil, 0, nil
			},
		}
		app := appConRepo(repo, &MockCacheService{})

		code, body := pedirJSON(t, app, http.MethodGet, token,
			"/api/admin/comisiones?estado=inventado", "")
		if code != http.StatusBadRequest {
			t.Fatalf("esperaba 400, obtuve %d (%s)", code, body)
		}
		if llamadas != 0 {
			t.Error("un filtro inválido no debe llegar al repositorio")
		}
	})

	t.Run("referred_id no numérico da 400", func(t *testing.T) {
		token := firmarTokenRol(t, 1, "admin@upse.edu.ec", "admin")
		app := appConRepo(&fakeRepo{}, &MockCacheService{})
		code, _ := pedirJSON(t, app, http.MethodGet, token,
			"/api/admin/comisiones?referido_id=abc", "")
		if code != http.StatusBadRequest {
			t.Errorf("esperaba 400, obtuve %d", code)
		}
	})

	t.Run("filtro válido llega al repo", func(t *testing.T) {
		token := firmarTokenRol(t, 1, "admin@upse.edu.ec", "admin")
		var filtro interfaces.ComisionFiltro
		repo := &fakeRepo{
			GetComisionesFn: func(f interfaces.ComisionFiltro) ([]models.Comision, int, error) {
				filtro = f
				return []models.Comision{}, 0, nil
			},
			GetTotalesComisionesFn: func(*int) ([]models.ComisionTotal, error) {
				return []models.ComisionTotal{}, nil
			},
		}
		app := appConRepo(repo, &MockCacheService{})

		code, _ := pedirJSON(t, app, http.MethodGet, token,
			"/api/admin/comisiones?estado=pagada&referido_id=7", "")
		if code != http.StatusOK {
			t.Fatalf("esperaba 200, obtuve %d", code)
		}
		if filtro.Estado != "pagada" {
			t.Errorf("Estado = %q, esperaba pagada", filtro.Estado)
		}
		if filtro.ReferidoID == nil || *filtro.ReferidoID != 7 {
			t.Errorf("ReferidoID = %v, esperaba 7", filtro.ReferidoID)
		}
	})

	t.Run("sin nodo de red devuelve vacío, no error", func(t *testing.T) {
		token := firmarToken(t, 5, "nuevo@upse.edu.ec")
		repo := &fakeRepo{
			GetReferidoIDByUsuarioFn: func(int) (*int, error) { return nil, nil },
			GetComisionesFn: func(interfaces.ComisionFiltro) ([]models.Comision, int, error) {
				t.Error("no debe consultar comisiones si el usuario no tiene nodo")
				return nil, 0, nil
			},
		}
		app := appConRepo(repo, &MockCacheService{})

		code, body := pedirJSON(t, app, http.MethodGet, token, "/api/comisiones/mias", "")
		if code != http.StatusOK {
			t.Fatalf("un usuario sin nodo no es un error: esperaba 200, obtuve %d (%s)", code, body)
		}
		var res models.ComisionResumen
		if err := json.Unmarshal(body, &res); err != nil {
			t.Fatalf("no deserializa: %v", err)
		}
		if len(res.Data) != 0 {
			t.Errorf("esperaba lista vacía, obtuve %d filas", len(res.Data))
		}
	})

	t.Run("fallo al buscar el nodo da 500", func(t *testing.T) {
		token := firmarToken(t, 5, "nuevo@upse.edu.ec")
		repo := &fakeRepo{
			GetReferidoIDByUsuarioFn: func(int) (*int, error) { return nil, errors.New("db caída") },
		}
		app := appConRepo(repo, &MockCacheService{})
		code, _ := pedirJSON(t, app, http.MethodGet, token, "/api/comisiones/mias", "")
		if code != http.StatusInternalServerError {
			t.Errorf("esperaba 500, obtuve %d", code)
		}
	})
}

// ── Pedidos: mapeo de errores de la FSM ─────────────────────────────────────

func TestPedidosController_mapeaErroresDeTransicion(t *testing.T) {
	token := firmarTokenRol(t, 1, "admin@upse.edu.ec", "admin")

	casos := []struct {
		nombre   string
		errRepo  error
		esperado int
	}{
		{"id inexistente", models.ErrPedidoNoEncontrado, http.StatusNotFound},
		{"transición ilegal", domain.ErrTransicionInvalida, http.StatusConflict},
		{"fallo de base", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			repo := &fakeRepo{
				CambiarEstadoPedidoFn: func(int, domain.EstadoPedido) error { return c.errRepo },
			}
			app := appConRepo(repo, &MockCacheService{})

			code, body := pedirJSON(t, app, http.MethodPatch, token,
				"/api/admin/pedidos/1/estado", `{"estado":"entregado"}`)
			if code != c.esperado {
				t.Errorf("esperaba %d, obtuve %d (%s)", c.esperado, code, body)
			}
		})
	}

	t.Run("estado fuera del CHECK da 400", func(t *testing.T) {
		repo := &fakeRepo{
			CambiarEstadoPedidoFn: func(int, domain.EstadoPedido) error {
				t.Error("un estado inválido no debe llegar al repositorio")
				return nil
			},
		}
		app := appConRepo(repo, &MockCacheService{})

		code, _ := pedirJSON(t, app, http.MethodPatch, token,
			"/api/admin/pedidos/1/estado", `{"estado":"inventado"}`)
		if code != http.StatusBadRequest {
			t.Errorf("esperaba 400, obtuve %d", code)
		}
	})

	t.Run("id no numérico da 400", func(t *testing.T) {
		app := appConRepo(&fakeRepo{}, &MockCacheService{})
		code, _ := pedirJSON(t, app, http.MethodPatch, token,
			"/api/admin/pedidos/abc/estado", `{"estado":"entregado"}`)
		if code != http.StatusBadRequest {
			t.Errorf("esperaba 400, obtuve %d", code)
		}
	})
}

// ── Liquidaciones ───────────────────────────────────────────────────────────

func TestLiquidacionesController_crearMapeaErrores(t *testing.T) {
	token := firmarTokenRol(t, 1, "admin@upse.edu.ec", "admin")

	t.Run("sin deuda da 409, no 201 vacío", func(t *testing.T) {
		repo := &fakeRepo{
			LiquidarFn: func(int) (*models.Liquidacion, error) {
				return nil, models.ErrNadaQueLiquidar
			},
		}
		app := appConRepo(repo, &MockCacheService{})
		code, body := pedirJSON(t, app, http.MethodPost, token,
			"/api/admin/liquidaciones", `{"referido_id":2}`)
		if code != http.StatusConflict {
			t.Errorf("esperaba 409, obtuve %d (%s)", code, body)
		}
	})

	t.Run("referido inexistente da 404", func(t *testing.T) {
		repo := &fakeRepo{
			LiquidarFn: func(int) (*models.Liquidacion, error) {
				return nil, models.ErrReferidoInvalido
			},
		}
		app := appConRepo(repo, &MockCacheService{})
		code, _ := pedirJSON(t, app, http.MethodPost, token,
			"/api/admin/liquidaciones", `{"referido_id":999}`)
		if code != http.StatusNotFound {
			t.Errorf("esperaba 404, obtuve %d", code)
		}
	})

	t.Run("referido_id no positivo da 400 sin llamar al repo", func(t *testing.T) {
		repo := &fakeRepo{
			LiquidarFn: func(int) (*models.Liquidacion, error) {
				t.Error("no debe liquidar con referido_id inválido")
				return nil, nil
			},
		}
		app := appConRepo(repo, &MockCacheService{})
		code, _ := pedirJSON(t, app, http.MethodPost, token,
			"/api/admin/liquidaciones", `{"referido_id":0}`)
		if code != http.StatusBadRequest {
			t.Errorf("esperaba 400, obtuve %d", code)
		}
	})

	t.Run("éxito da 201 con la liquidación", func(t *testing.T) {
		repo := &fakeRepo{
			LiquidarFn: func(referidoID int) (*models.Liquidacion, error) {
				if referidoID != 2 {
					t.Errorf("referido_id = %d, esperaba 2", referidoID)
				}
				return &models.Liquidacion{
					ID: 1, ReferidoID: referidoID, Total: 150, Estado: "pagada",
				}, nil
			},
		}
		app := appConRepo(repo, &MockCacheService{})
		code, body := pedirJSON(t, app, http.MethodPost, token,
			"/api/admin/liquidaciones", `{"referido_id":2}`)
		if code != http.StatusCreated {
			t.Fatalf("esperaba 201, obtuve %d (%s)", code, body)
		}
		var liq models.Liquidacion
		if err := json.Unmarshal(body, &liq); err != nil {
			t.Fatalf("no deserializa: %v", err)
		}
		if liq.Total != 150 || liq.Estado != "pagada" {
			t.Errorf("liquidación inesperada: %+v", liq)
		}
	})
}

func TestLiquidacionesController_listarNormalizaPaginacion(t *testing.T) {
	token := firmarTokenRol(t, 1, "admin@upse.edu.ec", "admin")
	var page, limit int
	repo := &fakeRepo{
		GetLiquidacionesFn: func(p, l int) ([]models.Liquidacion, int, error) {
			page, limit = p, l
			return []models.Liquidacion{}, 0, nil
		},
	}
	app := appConRepo(repo, &MockCacheService{})

	// Sin query-params: el controller normaliza a page=1 limit=20.
	code, body := pedirJSON(t, app, http.MethodGet, token, "/api/admin/liquidaciones", "")
	if code != http.StatusOK {
		t.Fatalf("esperaba 200, obtuve %d (%s)", code, body)
	}

	var pag paginaRespuesta
	_ = json.Unmarshal(body, &pag)
	if pag.Page != 1 || pag.Limit != 20 {
		t.Errorf("esperaba page=1 limit=20, obtuve page=%d limit=%d", pag.Page, pag.Limit)
	}

	// El repo recibe 0 (sin normalizar) y el controller normaliza al responder.
	if page != 0 || limit != 0 {
		t.Errorf("el repo recibió page=%d limit=%d; la normalización es del controller", page, limit)
	}
}

func TestLiquidacionesController_listarMiasSinNodo(t *testing.T) {
	token := firmarToken(t, 9, "sinred@upse.edu.ec")
	repo := &fakeRepo{
		GetReferidoIDByUsuarioFn: func(int) (*int, error) { return nil, nil },
		GetLiquidacionesDeFn: func(int, int, int) ([]models.Liquidacion, int, error) {
			t.Error("no debe consultar liquidaciones de un referido inexistente")
			return nil, 0, nil
		},
	}
	app := appConRepo(repo, &MockCacheService{})

	code, body := pedirJSON(t, app, http.MethodGet, token, "/api/liquidaciones/mias", "")
	if code != http.StatusOK {
		t.Fatalf("esperaba 200, obtuve %d (%s)", code, body)
	}
	var pag paginaRespuesta
	_ = json.Unmarshal(body, &pag)
	var liqs []models.Liquidacion
	_ = json.Unmarshal(pag.Data, &liqs)
	if len(liqs) != 0 {
		t.Errorf("esperaba lista vacía, obtuve %d", len(liqs))
	}
}

func TestLiquidacionesController_deuda(t *testing.T) {
	token := firmarTokenRol(t, 1, "admin@upse.edu.ec", "admin")
	repo := &fakeRepo{
		GetDeudaFn: func() ([]models.DeudaDistribuidor, error) {
			return []models.DeudaDistribuidor{
				{ReferidoID: 2, Codigo: "AB", Nombre: "Ana", Total: 90, Comisiones: 2},
			}, nil
		},
	}
	app := appConRepo(repo, &MockCacheService{})

	code, body := pedirJSON(t, app, http.MethodGet, token, "/api/admin/liquidaciones/deuda", "")
	if code != http.StatusOK {
		t.Fatalf("esperaba 200, obtuve %d (%s)", code, body)
	}
	var deudas []models.DeudaDistribuidor
	if err := json.Unmarshal(body, &deudas); err != nil {
		t.Fatalf("no deserializa: %v", err)
	}
	if len(deudas) != 1 || deudas[0].Total != 90 {
		t.Errorf("deuda inesperada: %+v", deudas)
	}
}
