package controllers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"multicatalogo-backend/controllers"
	"multicatalogo-backend/domain"
	"multicatalogo-backend/interfaces"
	"multicatalogo-backend/internal/dto"
	"multicatalogo-backend/models"
)

// Segunda tanda de tests sin Postgres: auth, catálogo admin, pedidos y la
// caché versionada de la red. Ver controllers_unit_test.go para el patrón.

// ── Auth ────────────────────────────────────────────────────────────────────

func TestAuthController_login(t *testing.T) {
	t.Run("credenciales válidas devuelven token con el rol", func(t *testing.T) {
		repo := &fakeRepo{
			LoginFn: func(email, password string) (*models.Usuario, error) {
				if email != "admin@upse.edu.ec" || password != "12345678" {
					t.Errorf("credenciales reenviadas: %s / %s", email, password)
				}
				return &models.Usuario{ID: 1, Email: email, Rol: "admin"}, nil
			},
		}
		app := appConRepo(repo, &MockCacheService{})

		code, body := pedirJSON(t, app, http.MethodPost, "",
			"/api/login", `{"email":"admin@upse.edu.ec","password":"12345678"}`)
		if code != http.StatusOK {
			t.Fatalf("esperaba 200, obtuve %d (%s)", code, body)
		}
		var resp controllers.LoginResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			t.Fatalf("no deserializa: %v (%s)", err, body)
		}
		if resp.Token == "" {
			t.Error("token vacío")
		}
		if resp.Rol != "admin" {
			t.Errorf("rol = %q, esperaba admin", resp.Rol)
		}
	})

	t.Run("password corto da 400 sin tocar la base", func(t *testing.T) {
		repo := &fakeRepo{
			LoginFn: func(string, string) (*models.Usuario, error) {
				t.Error("un body inválido no debe llegar al repositorio")
				return nil, nil
			},
		}
		app := appConRepo(repo, &MockCacheService{})

		code, _ := pedirJSON(t, app, http.MethodPost, "",
			"/api/login", `{"email":"a@b.com","password":"corto"}`)
		if code != http.StatusBadRequest {
			t.Errorf("esperaba 400, obtuve %d", code)
		}
	})

	t.Run("email mal formado da 400", func(t *testing.T) {
		app := appConRepo(&fakeRepo{}, &MockCacheService{})
		code, _ := pedirJSON(t, app, http.MethodPost, "",
			"/api/login", `{"email":"no-es-email","password":"12345678"}`)
		if code != http.StatusBadRequest {
			t.Errorf("esperaba 400, obtuve %d", code)
		}
	})

	// 401 vs 403 no es cosmético: 403 = "la cuenta existe pero está apagada",
	// 401 = "no sé quién eres". Confundirlos filtra qué emails están dados de
	// alta en el sistema.
	t.Run("credenciales malas da 401, cuenta desactivada da 403", func(t *testing.T) {
		casos := []struct {
			nombre   string
			errRepo  error
			esperado int
		}{
			{"inválidas", models.ErrCredencialesInvalidas, http.StatusUnauthorized},
			{"desactivada", models.ErrCuentaDesactivada, http.StatusForbidden},
			{"fallo interno", errors.New("boom"), http.StatusInternalServerError},
		}
		for _, c := range casos {
			t.Run(c.nombre, func(t *testing.T) {
				repo := &fakeRepo{
					LoginFn: func(string, string) (*models.Usuario, error) { return nil, c.errRepo },
				}
				app := appConRepo(repo, &MockCacheService{})
				code, _ := pedirJSON(t, app, http.MethodPost, "",
					"/api/login", `{"email":"a@b.com","password":"12345678"}`)
				if code != c.esperado {
					t.Errorf("esperaba %d, obtuve %d", c.esperado, code)
				}
			})
		}
	})
}

func TestAuthController_register(t *testing.T) {
	t.Run("alta correcta devuelve 201 y sube la versión de la red", func(t *testing.T) {
		cache := newVersionedCache()
		repo := &fakeRepo{
			RegistrarFn: func(req models.RegisterRequest) (*models.RegisterResult, error) {
				if req.Ref != "AB12" {
					t.Errorf("ref = %q; debe normalizarse a mayúsculas", req.Ref)
				}
				return &models.RegisterResult{
					UsuarioID: 7, Email: req.Email, CodigoRef: "XY99",
				}, nil
			},
		}
		app := appConRepo(repo, cache)

		code, body := pedirJSON(t, app, http.MethodPost, "", "/api/register",
			`{"nombre":"Ana","email":"ana@x.com","password":"12345678","ref":"ab12"}`)
		if code != http.StatusCreated {
			t.Fatalf("esperaba 201, obtuve %d (%s)", code, body)
		}
		var out struct {
			ID        int    `json:"id"`
			Email     string `json:"email"`
			CodigoRef string `json:"codigo_ref"`
		}
		_ = json.Unmarshal(body, &out)
		if out.ID != 7 || out.CodigoRef != "XY99" {
			t.Errorf("respuesta inesperada: %+v", out)
		}
		// El alta crea un nodo en la red → cualquier árbol cacheado es viejo.
		if _, _, _, inc := cache.accesos(); inc["red:ver"] == 0 {
			t.Error("el registro no invalidó la caché de la red")
		}
	})

	t.Run("ref con caracteres no alfanuméricos da 400 sin tocar la base", func(t *testing.T) {
		repo := &fakeRepo{
			RegistrarFn: func(models.RegisterRequest) (*models.RegisterResult, error) {
				t.Error("un código inválido no debe llegar al repositorio")
				return nil, nil
			},
		}
		app := appConRepo(repo, &MockCacheService{})
		code, _ := pedirJSON(t, app, http.MethodPost, "", "/api/register",
			`{"nombre":"Ana","email":"ana@x.com","password":"12345678","ref":"AB-1"}`)
		if code != http.StatusBadRequest {
			t.Errorf("esperaba 400, obtuve %d", code)
		}
	})

	// domain.CodigoEsValido es LAXO con el alfabeto a propósito (acepta I/O/0/1
	// para validar los códigos seed "TU"/"ADMIN"). Este test fija ese
	// comportamiento: si alguien endurece la función sin migrar el seed, los
	// registros existentes dejan de poder referirse.
	t.Run("ref con dígitos y O/I se acepta (alfabeto laxo)", func(t *testing.T) {
		repo := &fakeRepo{
			RegistrarFn: func(models.RegisterRequest) (*models.RegisterResult, error) {
				return &models.RegisterResult{UsuarioID: 2, Email: "a@b.com", CodigoRef: "TU"}, nil
			},
		}
		app := appConRepo(repo, &MockCacheService{})
		code, body := pedirJSON(t, app, http.MethodPost, "", "/api/register",
			`{"nombre":"Ana","email":"ana@x.com","password":"12345678","ref":"AB10"}`)
		if code != http.StatusCreated {
			t.Errorf("esperaba 201 para un código laxo válido, obtuve %d (%s)", code, body)
		}
	})

	t.Run("email duplicado da 409", func(t *testing.T) {
		repo := &fakeRepo{
			RegistrarFn: func(models.RegisterRequest) (*models.RegisterResult, error) {
				return nil, models.ErrEmailDuplicado
			},
		}
		app := appConRepo(repo, &MockCacheService{})
		code, _ := pedirJSON(t, app, http.MethodPost, "", "/api/register",
			`{"nombre":"Ana","email":"ana@x.com","password":"12345678"}`)
		if code != http.StatusConflict {
			t.Errorf("esperaba 409, obtuve %d", code)
		}
	})

	t.Run("password corta da 400", func(t *testing.T) {
		app := appConRepo(&fakeRepo{}, &MockCacheService{})
		code, _ := pedirJSON(t, app, http.MethodPost, "", "/api/register",
			`{"nombre":"Ana","email":"ana@x.com","password":"123"}`)
		if code != http.StatusBadRequest {
			t.Errorf("esperaba 400, obtuve %d", code)
		}
	})
}

// ── Catálogo admin ──────────────────────────────────────────────────────────

func TestProdController_adminList(t *testing.T) {
	t.Run("filtro activo inválido da 400", func(t *testing.T) {
		token := firmarTokenRol(t, 1, "admin@upse.edu.ec", "admin")
		app := appConRepo(&fakeRepo{}, &MockCacheService{})
		code, _ := pedirJSON(t, app, http.MethodGet, token,
			"/api/admin/productos?activo=quiza", "")
		if code != http.StatusBadRequest {
			t.Errorf("esperaba 400, obtuve %d", code)
		}
	})

	t.Run("parámetro no numérico cae al default, no a 0", func(t *testing.T) {
		token := firmarTokenRol(t, 1, "admin@upse.edu.ec", "admin")
		var filtro interfaces.ProductoFiltro
		repo := &fakeRepo{
			GetProductosAdminFn: func(f interfaces.ProductoFiltro) ([]models.Producto, int, error) {
				filtro = f
				return []models.Producto{}, 0, nil
			},
		}
		app := appConRepo(repo, &MockCacheService{})

		code, _ := pedirJSON(t, app, http.MethodGet, token,
			"/api/admin/productos?page=abc&limit=xyz&q=shampoo&activo=true", "")
		if code != http.StatusOK {
			t.Fatalf("esperaba 200, obtuve %d", code)
		}
		if filtro.Page != 1 || filtro.Limit != 20 {
			t.Errorf("esperaba page=1 limit=20, obtuve page=%d limit=%d", filtro.Page, filtro.Limit)
		}
		if filtro.Q != "shampoo" {
			t.Errorf("Q = %q, esperaba shampoo", filtro.Q)
		}
		if filtro.Activo == nil || !*filtro.Activo {
			t.Errorf("Activo = %v, esperaba &true", filtro.Activo)
		}
	})
}

func TestProdController_adminCreateInvalidaCache(t *testing.T) {
	token := firmarTokenRol(t, 1, "admin@upse.edu.ec", "admin")
	cache := newVersionedCache()
	repo := &fakeRepo{
		CreateProductoFn: func(d dto.ProductoCreate) (*models.Producto, error) {
			return &models.Producto{ID: 1, Nombre: d.Nombre, Activo: true}, nil
		},
	}
	app := appConRepo(repo, cache)

	code, _ := pedirJSON(t, app, http.MethodPost, token, "/api/admin/productos",
		`{"nombre":"Shampoo","precio":15.5,"categoria":"capilar","stock":10}`)
	if code != http.StatusCreated {
		t.Fatalf("esperaba 201, obtuve %d", code)
	}
	_, _, del, _ := cache.accesos()
	if del["productos:all"] == 0 {
		t.Errorf("crear un producto no invalidó el caché público; la lista seguirá stale: %v",
			del)
	}
}

func TestProdController_adminCRUD(t *testing.T) {
	token := firmarTokenRol(t, 1, "admin@upse.edu.ec", "admin")

	t.Run("precio no positivo da 400", func(t *testing.T) {
		repo := &fakeRepo{
			CreateProductoFn: func(dto.ProductoCreate) (*models.Producto, error) {
				t.Error("precio inválido no debe llegar al repositorio")
				return nil, nil
			},
		}
		app := appConRepo(repo, &MockCacheService{})
		code, _ := pedirJSON(t, app, http.MethodPost, token, "/api/admin/productos",
			`{"nombre":"X","precio":-1,"categoria":"capilar","stock":1}`)
		if code != http.StatusBadRequest {
			t.Errorf("esperaba 400, obtuve %d", code)
		}
	})

	t.Run("actualizar inexistente da 404", func(t *testing.T) {
		repo := &fakeRepo{
			UpdateProductoFn: func(int, dto.ProductoUpdate) (*models.Producto, error) {
				return nil, models.ErrProductoNoEncontrado
			},
		}
		app := appConRepo(repo, &MockCacheService{})
		code, _ := pedirJSON(t, app, http.MethodPut, token,
			"/api/admin/productos/999", `{"precio":10}`)
		if code != http.StatusNotFound {
			t.Errorf("esperaba 404, obtuve %d", code)
		}
	})

	t.Run("borrar da 204 e invalida caché", func(t *testing.T) {
		cache := newVersionedCache()
		repo := &fakeRepo{SoftDeleteProductoFn: func(int) error { return nil }}
		app := appConRepo(repo, cache)

		code, _ := pedirJSON(t, app, http.MethodDelete, token, "/api/admin/productos/1", "")
		if code != http.StatusNoContent {
			t.Errorf("esperaba 204, obtuve %d", code)
		}
		_, _, del2, _ := cache.accesos()
		if del2["productos:all"] == 0 {
			t.Errorf("el soft-delete no invalidó el caché: %v", del2)
		}
	})

	t.Run("id no numérico da 400", func(t *testing.T) {
		app := appConRepo(&fakeRepo{}, &MockCacheService{})
		code, _ := pedirJSON(t, app, http.MethodDelete, token, "/api/admin/productos/abc", "")
		if code != http.StatusBadRequest {
			t.Errorf("esperaba 400, obtuve %d", code)
		}
	})
}

// ── Pedidos ─────────────────────────────────────────────────────────────────

func TestPedidosController_detalleRespetaPropiedad(t *testing.T) {
	duenio := 5
	repo := &fakeRepo{
		GetPedidoDetalleFn: func(id int) (*models.PedidoDetalle, error) {
			return &models.PedidoDetalle{ID: id, UsuarioID: &duenio, Total: 100}, nil
		},
	}
	app := appConRepo(repo, &MockCacheService{})

	t.Run("el dueño ve su pedido", func(t *testing.T) {
		code, _ := pedirJSON(t, app, http.MethodGet, firmarToken(t, 5, "duenio@x.com"),
			"/api/pedidos/1", "")
		if code != http.StatusOK {
			t.Errorf("esperaba 200, obtuve %d", code)
		}
	})

	// IDOR: un cliente que pide el pedido de otro. Se responde 404, no 403, a
	// propósito — un 403 confirmaría que ese pedido existe.
	t.Run("otro cliente recibe 404, no 403", func(t *testing.T) {
		code, body := pedirJSON(t, app, http.MethodGet, firmarToken(t, 6, "otro@x.com"),
			"/api/pedidos/1", "")
		if code != http.StatusNotFound {
			t.Errorf("esperaba 404, obtuve %d (%s)", code, body)
		}
		if strings.Contains(strings.ToLower(string(body)), "otro") {
			t.Errorf("la respuesta sugiere existencia del recurso: %s", body)
		}
	})

	t.Run("admin ve cualquier pedido", func(t *testing.T) {
		code, _ := pedirJSON(t, app, http.MethodGet,
			firmarTokenRol(t, 1, "admin@upse.edu.ec", "admin"), "/api/pedidos/1", "")
		if code != http.StatusOK {
			t.Errorf("esperaba 200, obtuve %d", code)
		}
	})
}

func TestPedidosController_detalleYListados(t *testing.T) {
	token := firmarTokenRol(t, 1, "admin@upse.edu.ec", "admin")

	t.Run("inexistente da 404", func(t *testing.T) {
		repo := &fakeRepo{
			GetPedidoDetalleFn: func(int) (*models.PedidoDetalle, error) {
				return nil, models.ErrPedidoNoEncontrado
			},
		}
		app := appConRepo(repo, &MockCacheService{})
		code, _ := pedirJSON(t, app, http.MethodGet, token, "/api/pedidos/77", "")
		if code != http.StatusNotFound {
			t.Errorf("esperaba 404, obtuve %d", code)
		}
	})

	t.Run("mis pedidos normaliza paginación y acota limit", func(t *testing.T) {
		var page, limit int
		repo := &fakeRepo{
			GetMisPedidosFn: func(usuarioID, p, l int) ([]models.PedidoResumen, int, error) {
				page, limit = p, l
				return []models.PedidoResumen{}, 0, nil
			},
		}
		app := appConRepo(repo, &MockCacheService{})

		code, body := pedirJSON(t, app, http.MethodGet, firmarToken(t, 5, "u@x.com"),
			"/api/pedidos/mios?page=0&limit=9999", "")
		if code != http.StatusOK {
			t.Fatalf("esperaba 200, obtuve %d (%s)", code, body)
		}
		if page != 1 {
			t.Errorf("page = %d, esperaba 1 (page=0 se normaliza)", page)
		}
		if limit != 100 {
			t.Errorf("limit = %d, esperaba 100 (tope)", limit)
		}
	})

	t.Run("listado admin propaga el filtro de estado", func(t *testing.T) {
		var filtro interfaces.PedidoFiltro
		repo := &fakeRepo{
			GetPedidosAdminFn: func(f interfaces.PedidoFiltro) ([]models.PedidoResumen, int, error) {
				filtro = f
				return []models.PedidoResumen{}, 0, nil
			},
		}
		app := appConRepo(repo, &MockCacheService{})

		code, _ := pedirJSON(t, app, http.MethodGet, token,
			"/api/admin/pedidos?estado=preparando&page=2&limit=10", "")
		if code != http.StatusOK {
			t.Fatalf("esperaba 200, obtuve %d", code)
		}
		if filtro.Estado != "preparando" || filtro.Page != 2 || filtro.Limit != 10 {
			t.Errorf("filtro inesperado: %+v", filtro)
		}
	})
}

// ── Red: caché versionada ───────────────────────────────────────────────────

func TestNetworkController_arbolSeSirveDelCache(t *testing.T) {
	token := firmarTokenRol(t, 1, "admin@upse.edu.ec", "admin")
	cache := newVersionedCache()

	var llamadas int
	repo := &fakeRepo{
		GetRedGlobalFn: func() ([]domain.NodoPlano, error) {
			llamadas++
			return []domain.NodoPlano{
				{ID: 1, Codigo: "AB", Nombre: "Raíz", Ventas: 100},
			}, nil
		},
	}
	app := appConRepo(repo, cache)

	code, _ := pedirJSON(t, app, http.MethodGet, token, "/api/red", "")
	if code != http.StatusOK {
		t.Fatalf("primera llamada falló: %d", code)
	}
	code, body := pedirJSON(t, app, http.MethodGet, token, "/api/red", "")
	if code != http.StatusOK {
		t.Fatalf("segunda llamada falló: %d", code)
	}

	if llamadas != 1 {
		t.Errorf("la base se consultó %d veces; la segunda debía salir del caché", llamadas)
	}
	if !strings.Contains(string(body), "Raíz") {
		t.Errorf("el árbol cacheado perdió los nodos: %s", body)
	}
}

func TestNetworkController_versionSubeEnRegistro(t *testing.T) {
	cache := newVersionedCache()
	repo := &fakeRepo{
		RegistrarFn: func(models.RegisterRequest) (*models.RegisterResult, error) {
			return &models.RegisterResult{UsuarioID: 1, Email: "a@b.com", CodigoRef: "AA"}, nil
		},
	}
	app := appConRepo(repo, cache)

	pedirJSON(t, app, http.MethodPost, "", "/api/register",
		`{"nombre":"Ana","email":"a@b.com","password":"12345678"}`)

	// El árbol se cachea con la versión embebida; si el registro no la sube,
	// los nodos nuevos no aparecen hasta que expire el TTL.
	if _, _, _, inc := cache.accesos(); inc["red:ver"] == 0 {
		t.Error("el alta no consultó/subió la versión del caché de la red")
	}
}

// TestNetworkController_cacheConVariasVersiones comprueba que al subir la
// versión se sirve un árbol nuevo, no el viejo cacheado.
func TestNetworkController_cacheConVariasVersiones(t *testing.T) {
	token := firmarTokenRol(t, 1, "admin@upse.edu.ec", "admin")
	cache := newVersionedCache()

	raiz := "Primera"
	repo := &fakeRepo{
		GetRedGlobalFn: func() ([]domain.NodoPlano, error) {
			return []domain.NodoPlano{{ID: 1, Codigo: "AB", Nombre: raiz}}, nil
		},
	}
	app := appConRepo(repo, cache)

	pedirJSON(t, app, http.MethodGet, token, "/api/red", "")

	// Se sube la versión como haría un checkout y se cambia el árbol en DB.
	cache.Increment(context.Background(), "red:ver") //nolint:errcheck // simula el checkout
	raiz = "Segunda"

	_, body := pedirJSON(t, app, http.MethodGet, token, "/api/red", "")
	if !strings.Contains(string(body), "Segunda") {
		t.Errorf("siguió sirviendo el árbol viejo tras cambiar la versión: %s", body)
	}
}

// TestNetworkController_redMiaAisladaPorUsuario comprueba que cada
// distribuidor recibe SU subárbol, no el global. Es aislamiento de datos entre
// usuarios: si se sirviera el árbol completo, "mi red" filtraría la estructura
// de la organización a un cliente normal.
func TestNetworkController_redMiaAisladaPorUsuario(t *testing.T) {
	cache := newVersionedCache()
	var usuarioPedido int
	repo := &fakeRepo{
		GetRedPropiaFn: func(usuarioID int) ([]domain.NodoPlano, error) {
			usuarioPedido = usuarioID
			return []domain.NodoPlano{
				{ID: 10, UsuarioID: &usuarioID, Codigo: "YO", Nombre: "Mío"},
			}, nil
		},
	}
	app := appConRepo(repo, cache)

	token := firmarToken(t, 42, "distribuidor@x.com")
	code, body := pedirJSON(t, app, http.MethodGet, token, "/api/red/mia", "")
	if code != http.StatusOK {
		t.Fatalf("esperaba 200, obtuve %d (%s)", code, body)
	}
	if usuarioPedido != 42 {
		t.Errorf("se consultó el subárbol del usuario %d, esperaba el 42 del token", usuarioPedido)
	}
	if !strings.Contains(string(body), "Mío") {
		t.Errorf("respuesta inesperada: %s", body)
	}
	if strings.Contains(string(body), "Todos") {
		t.Errorf("la vista personal incluye nodos ajenos: %s", body)
	}
}

// TestCheckoutController_felizCompraPropagaUnContador cubre el camino feliz
// del checkout, que además dispara actualizarContadores en una goroutine.
func TestCheckoutController_felizCompra(t *testing.T) {
	token := firmarToken(t, 5, "comprador@x.com")
	cache := newVersionedCache()

	actualizado := make(chan string, 4)
	repo := &fakeRepo{
		CreatePedidoConUsuarioFn: func(dto.CheckoutRequest, int) (*models.PedidoDetalle, error) {
			return &models.PedidoDetalle{ID: 1, Total: 100}, nil
		},
		ActualizarVentasUplineFn: func(email string, total float64) error {
			actualizado <- email
			return nil
		},
	}
	app := appConRepo(repo, cache)

	code, body := pedirJSON(t, app, http.MethodPost, token, "/api/checkout",
		`{"direccion":"Av. 10 de Agosto 100","ciudad":"Quito","telefono":"0987654321","metodo_pago":"tarjeta","items":[{"id":1,"nombre":"X","precio":100,"cantidad":1}]}`)
	if code != http.StatusCreated {
		t.Fatalf("esperaba 201, obtuve %d (%s)", code, body)
	}

	// El contador corre en una goroutine post-COMMIT: hay que esperarlo.
	select {
	case email := <-actualizado:
		// El checkout toma la identidad del token, no la del body.
		if email != "comprador@x.com" {
			t.Errorf("se acreditó a %q; la identidad debe venir del token", email)
		}
	case <-time.After(3 * time.Second):
		t.Error("actualizarContadores no llegó a ejecutarse")
	}

	// Las invalidaciones de caché ocurren en la MISMA goroutine, después de
	// acreditar el contador. Comprobarlas en orden estricto sería una carrera:
	// el canal se emite antes de los Increment. Se espera a que ocurran.
	esperarCondicion(t, 3*time.Second, func() bool {
		_, _, _, inc := cache.accesos()
		return inc["red:ver"] > 0 && inc["reportes:ver"] > 0
	}, "el checkout no invalidó las cachés de red y de reportes")
}

// esperarCondicion sondea una condición hasta que se cumple o vence el plazo.
// Para efectos que corren en una goroutine, donde el orden dentro de la misma
// no es observable desde fuera.
func esperarCondicion(t *testing.T, timeout time.Duration, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Error(msg)
}

// TestCheckoutController_erreosDeValidacion cubre los rechazos que NO deben
// llegar al repositorio: carrito vacío y método de pago ausente.
func TestCheckoutController_erreosDeValidacion(t *testing.T) {
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
		{"carrito vacío", `{"metodo_pago":"tarjeta","items":[]}`},
		{"sin método de pago", `{"items":[{"id":1,"nombre":"X","precio":10,"cantidad":1}]}`},
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

// TestCheckoutController_stockInsuficienteEs409 distingue "no se puede
// comprar" de "el servidor falló": son códigos distintos porque el cliente
// reacciona distinto (reintentar vs avisar al admin).
func TestCheckoutController_stockInsuficienteEs409(t *testing.T) {
	token := firmarToken(t, 5, "comprador@x.com")
	repo := &fakeRepo{
		CreatePedidoConUsuarioFn: func(dto.CheckoutRequest, int) (*models.PedidoDetalle, error) {
			return nil, models.ErrProductoNoEncontrado
		},
	}
	app := appConRepo(repo, &MockCacheService{})

	code, body := pedirJSON(t, app, http.MethodPost, token, "/api/checkout",
		`{"direccion":"Av. 10 de Agosto 100","ciudad":"Quito","telefono":"0987654321","metodo_pago":"tarjeta","items":[{"id":9,"nombre":"X","precio":10,"cantidad":99}]}`)
	if code != http.StatusConflict {
		t.Fatalf("esperaba 409, obtuve %d (%s)", code, body)
	}
	var ae apiErr
	_ = json.Unmarshal(body, &ae)
	if ae.Code != "STOCK_INSUFICIENTE" {
		t.Errorf("código = %q, esperaba STOCK_INSUFICIENTE", ae.Code)
	}
}

// TestReportesController_invalidaVersionAlCambiarEstado cubre el enlace entre
// el cambio de estado de un pedido y los agregados del dashboard: si no sube
// la versión, el resumen sigue mostrando la venta de un pedido cancelado.
func TestReportesController_invalidaVersionAlCambiarEstado(t *testing.T) {
	token := firmarTokenRol(t, 1, "admin@upse.edu.ec", "admin")
	cache := newVersionedCache()
	repo := &fakeRepo{CambiarEstadoPedidoFn: func(int, domain.EstadoPedido) error { return nil }}
	app := appConRepo(repo, cache)

	code, _ := pedirJSON(t, app, http.MethodPatch, token,
		"/api/admin/pedidos/1/estado", `{"estado":"cancelado"}`)
	if code != http.StatusNoContent {
		t.Fatalf("esperaba 204, obtuve %d", code)
	}
	gets, _, _, inc := cache.accesos()
	if inc["reportes:ver"] == 0 {
		t.Errorf("cambiar el estado no invalidó el caché de reportes: %v", gets)
	}
}
