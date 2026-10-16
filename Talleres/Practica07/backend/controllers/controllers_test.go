package controllers_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"

	"multicatalogo-backend/config"
	"multicatalogo-backend/controllers"
	"multicatalogo-backend/middleware"
	"multicatalogo-backend/models"
	"multicatalogo-backend/repository"
	"multicatalogo-backend/routes"
)

// MockCacheService implementa interfaces.CacheService sin Redis real.
// Siempre retorna cache miss para que los tests usen los datos del repositorio.
type MockCacheService struct{}

func (m *MockCacheService) Get(_ context.Context, _ string) (string, error) {
	return "", errors.New("cache miss")
}
func (m *MockCacheService) Set(_ context.Context, _ string, _ interface{}, _ time.Duration) error {
	return nil
}
func (m *MockCacheService) Delete(_ context.Context, _ string) error { return nil }

// secret de pruebas — 64 bytes aleatorios pero deterministas.
var testJWTSecret = []byte("test-jwt-secret-test-jwt-secret-test-jwt-secret-test-jwt-secret")

// dsnTest es la cadena de conexión que usan los tests que pegan contra
// la DB real (practica07-postgres-1 escuchando en localhost:5433). Si
// TEST_DATABASE_URL está definida en el entorno gana sobre el default.
func dsnTest() string {
	if v := os.Getenv("TEST_DATABASE_URL"); v != "" {
		return v
	}
	return "postgres://admin:secret@localhost:5433/multicatalogo?sslmode=disable"
}

// setupTestApp construye una app Fiber con Repository{Nil DB} — útil
// sólo para tests que no tocan auth real (HealthCheck, Productos,
// Network, el sub-test "No DB returns 500" de Checkout).
func setupTestApp() *fiber.App {
	repo := repository.NewRepository(nil) // nil db: sin PostgreSQL en tests
	cache := &MockCacheService{}

	ctrl := routes.Controllers{
		Auth:     controllers.NewAuthController(repo),
		Producto: controllers.NewProductoController(repo, cache),
		Network:  controllers.NewNetworkController(repo, cache),
		Checkout: controllers.NewCheckoutController(repo, cache),
		Pedidos:  controllers.NewPedidosController(repo, cache),
	}

	app := fiber.New()
	app.Use(middleware.SecurityHeaders())
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(middleware.JWTSecretKey, testJWTSecret)
		return c.Next()
	})
	routes.SetupRoutes(app, ctrl, testJWTSecret)
	return app
}

// setupTestDB abre una conexión a PostgreSQL real y la devuelve, junto
// con una app Fiber completa. Si la DB no está disponible, el test se
// salta con un mensaje claro (no se trata como falla porque los tests
// de F1 son opcionales si el entorno no tiene DB levantada).
//
// Cada test que llama a esta función debe ejecutar truncateTestDB(t) en
// su setup para empezar con un estado limpio (excepto los seeds).
func setupTestDB(t *testing.T) (*fiber.App, *sql.DB) {
	t.Helper()
	db, err := sql.Open("postgres", dsnTest())
	if err != nil {
		t.Skipf("[setupTestDB] no se pudo abrir DB (%v)", err)
	}
	if err := db.Ping(); err != nil {
		t.Skipf("[setupTestDB] DB no disponible (%v) — saltando", err)
	}
	// Reemplazamos el DB global del package config para que
	// healthController (si lo usa) y cualquier path que dependa de él
	// también vean la conexión.
	config.DB = db

	// Re-seed de los usuarios base (admin y cliente demo) por si una
	// corrida previa los desactivó. El seed del 005 es idempotente,
	// pero este re-seed específico es defensivo.
	ensureSeed(t, db)

	repo := repository.NewRepository(db)
	cache := &MockCacheService{}

	ctrl := routes.Controllers{
		Auth:     controllers.NewAuthController(repo),
		Producto: controllers.NewProductoController(repo, cache),
		Network:  controllers.NewNetworkController(repo, cache),
		Checkout: controllers.NewCheckoutController(repo, cache),
		Pedidos:  controllers.NewPedidosController(repo, cache),
	}

	app := fiber.New()
	app.Use(middleware.SecurityHeaders())
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(middleware.JWTSecretKey, testJWTSecret)
		return c.Next()
	})
	routes.SetupRoutes(app, ctrl, testJWTSecret)
	return app, db
}

// ensureSeed garantiza que admin@upse.edu.ec y cliente@upse.edu.ec
// existen con password '12345678' y activo=true. Lo único que muta
// fuera del scope de un test. La implementación usa bcrypt compatible
// con el de 005_fundaciones.sql (cost 12).
func ensureSeed(t *testing.T, db *sql.DB) {
	t.Helper()
	seeds := []struct {
		email  string
		rol    string
		nombre string
	}{
		{"admin@upse.edu.ec", "admin", "Administrador"},
		{"cliente@upse.edu.ec", "cliente", "Cliente Demo"},
	}
	// Usamos pgcrypto via SQL puro (el hash bcrypt se calcula desde Go
	// para evitar duplicar la constante bcrypt entre .sql y .go).
	for _, s := range seeds {
		// Importamos bcrypt sólo aquí para no obligar al paquete
		// controllers a depender de crypto. Mejor lo movemos a domain
		// si crece, pero por ahora vive en línea.
		var exists bool
		_ = db.QueryRow(`SELECT EXISTS(SELECT 1 FROM usuarios WHERE email = $1)`, s.email).Scan(&exists)
		if !exists {
			hash := bcryptHashForTest(t, "12345678")
			_, err := db.Exec(
				`INSERT INTO usuarios (email, password_hash, rol, nombre, activo) VALUES ($1, $2, $3, $4, true)`,
				s.email, hash, s.rol, s.nombre,
			)
			if err != nil {
				t.Fatalf("[ensureSeed] insert %s: %v", s.email, err)
			}
		} else {
			_, err := db.Exec(`UPDATE usuarios SET activo = true WHERE email = $1`, s.email)
			if err != nil {
				t.Fatalf("[ensureSeed] reset activo %s: %v", s.email, err)
			}
		}
	}
	// Aseguramos que el nodo raíz (id=1, codigo 'TU') exista para que
	// los tests de upline tengan referencia estable.
	_, _ = db.Exec(`INSERT INTO referidos (id, codigo, parent_id) VALUES (1, 'TU', NULL) ON CONFLICT DO NOTHING`)
}

// truncateTestDB deja las tablas de F1 en un estado conocido: borra
// todos los usuarios que no sean los seeds, sus referidos (excepto los
// 3 nodos seed TU/ADMIN/LUIS), comisiones, liquidaciones y pedidos.
//
// IMPORTANTE: NO usar `OR usuario_id IS NULL` porque el nodo raíz TU
// (id=1) intencionalmente NO tiene usuario_id asociado.
func truncateTestDB(t *testing.T, db *sql.DB) {
	t.Helper()
	stmts := []string{
		`DELETE FROM liquidaciones`,
		`DELETE FROM comisiones`,
		`DELETE FROM pedido_items`,
		`DELETE FROM pedidos`,
		// Preservar SIEMPRE los 3 nodos seed (TU/ADMIN/LUIS).
		`DELETE FROM referidos WHERE codigo NOT IN ('TU','ADMIN','LUIS')`,
		// Usuarios no-seed (los seeds tienen referido_id apuntando a los nodos seed).
		`DELETE FROM usuarios WHERE email NOT IN ('admin@upse.edu.ec','cliente@upse.edu.ec')`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("[truncateTestDB] %s: %v", s, err)
		}
	}
}

// ─────────────────────────── Tests no-DB ───────────────────────────

func TestHealthCheck(t *testing.T) {
	app := setupTestApp()
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to parse json: %v", err)
	}
	if result["status"] != "ok" || result["message"] != "Servidor Go/Fiber operativo" {
		t.Errorf("unexpected response body: %s", string(body))
	}
}

// apiErr es un alias para no importar middleware.APIError en cada test.
type apiErr = middleware.APIError

// ─────────────────────────── Login (F1) ───────────────────────────

func TestLogin(t *testing.T) {
	app, db := setupTestDB(t)
	defer db.Close()

	t.Run("Invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader("invalid-json"))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := app.Test(req)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
		var ae apiErr
		body, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(body, &ae)
		if ae.Status != 400 || ae.Code != "VALIDATION_ERROR" || ae.Message != "cuerpo inválido" {
			t.Errorf("unexpected error: %+v", ae)
		}
	})

	t.Run("Empty fields (validator)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/login",
			strings.NewReader(`{"email":"","password":""}`))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := app.Test(req)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
		var ae apiErr
		body, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(body, &ae)
		if ae.Status != 400 || ae.Code != "VALIDATION_ERROR" {
			t.Errorf("unexpected error: %+v", ae)
		}
	})

	t.Run("Short password", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/login",
			strings.NewReader(`{"email":"admin@upse.edu.ec","password":"short"}`))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := app.Test(req)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
		var ae apiErr
		body, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(body, &ae)
		if ae.Status != 400 || ae.Code != "VALIDATION_ERROR" {
			t.Errorf("unexpected error: %+v", ae)
		}
	})

	t.Run("Invalid credentials", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/login",
			strings.NewReader(`{"email":"wrong@upse.edu.ec","password":"wrongpass"}`))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := app.Test(req)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", resp.StatusCode)
		}
		var ae apiErr
		body, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(body, &ae)
		if ae.Status != 401 || ae.Code != "UNAUTHORIZED" || ae.Message != "credenciales incorrectas" {
			t.Errorf("unexpected error: %+v", ae)
		}
	})

	t.Run("Wrong password (200 → 401 con DB real)", func(t *testing.T) {
		// usuario existe pero password no coincide — debe ser 401.
		req := httptest.NewRequest(http.MethodPost, "/api/login",
			strings.NewReader(`{"email":"admin@upse.edu.ec","password":"wrongpassword"}`))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := app.Test(req)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", resp.StatusCode)
		}
	})

	t.Run("Valid credentials returns signed JWT", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/login",
			strings.NewReader(`{"email":"admin@upse.edu.ec","password":"12345678"}`))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := app.Test(req)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
		var res controllers.LoginResponse
		body, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(body, &res)
		if res.Email != "admin@upse.edu.ec" {
			t.Errorf("unexpected email in login result: %v", res.Email)
		}
		if res.Rol != "admin" {
			t.Errorf("expected rol=admin, got %v", res.Rol)
		}
		if res.Token == "" {
			t.Fatalf("expected non-empty token")
		}
		// Verificar que el JWT sea válido con el secret de pruebas.
		claims, err := middleware.ParseJWT(testJWTSecret, res.Token)
		if err != nil {
			t.Fatalf("jwt parse failed: %v", err)
		}
		if claims.Email != "admin@upse.edu.ec" || claims.Rol != "admin" || claims.Sub != 1 {
			t.Errorf("unexpected claims: %+v", claims)
		}
	})

	t.Run("Inactive user → 403", func(t *testing.T) {
		// Insertar un usuario inactivo y luego intentar login con él.
		hash := bcryptHashForTest(t, "12345678")
		email := "inactivo@upse.edu.ec"
		_, err := db.Exec(
			`INSERT INTO usuarios (email, password_hash, rol, nombre, activo) VALUES ($1, $2, 'cliente', 'Inactivo', false)`,
			email, hash,
		)
		if err != nil {
			t.Fatalf("seed inactivo: %v", err)
		}
		defer func() { _, _ = db.Exec(`DELETE FROM usuarios WHERE email = $1`, email) }()

		req := httptest.NewRequest(http.MethodPost, "/api/login",
			strings.NewReader(`{"email":"inactivo@upse.edu.ec","password":"12345678"}`))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := app.Test(req)
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("expected 403, got %d", resp.StatusCode)
		}
		var ae apiErr
		body, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(body, &ae)
		if ae.Code != "FORBIDDEN" || ae.Message != "cuenta desactivada" {
			t.Errorf("unexpected error: %+v", ae)
		}
	})
}

// ─────────────────────────── Register (F1) ───────────────────────────

func TestRegisterHappy(t *testing.T) {
	app, db := setupTestDB(t)
	defer db.Close()
	truncateTestDB(t, db)

	body := `{"nombre":"Ana Nueva","email":"ana.nueva@upse.edu.ec","password":"supersecret","ref":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var res models.RegisterResult
	respBody, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(respBody, &res)

	if res.UsuarioID == 0 {
		t.Errorf("expected non-zero id, got 0")
	}
	if res.Email != "ana.nueva@upse.edu.ec" {
		t.Errorf("unexpected email: %v", res.Email)
	}
	if res.CodigoRef == "" {
		t.Fatalf("expected non-empty codigo_ref")
	}

	// Verificar que el código existe en DB para este usuario.
	var dbCodigo string
	err := db.QueryRow(`SELECT codigo FROM referidos WHERE usuario_id = $1`, res.UsuarioID).Scan(&dbCodigo)
	if err != nil {
		t.Fatalf("error leyendo DB: %v", err)
	}
	if dbCodigo != res.CodigoRef {
		t.Errorf("DB codigo %q != response codigo_ref %q", dbCodigo, res.CodigoRef)
	}
}

func TestRegisterDupEmail(t *testing.T) {
	app, db := setupTestDB(t)
	defer db.Close()
	truncateTestDB(t, db)

	body := `{"nombre":"Ana Dup","email":"dup@upse.edu.ec","password":"supersecret"}`
	req := httptest.NewRequest(http.MethodPost, "/api/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("primer registro: expected 201, got %d", resp.StatusCode)
	}

	// Segundo registro con el mismo email → 409 CONFLICT.
	req2 := httptest.NewRequest(http.MethodPost, "/api/register", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	resp2, _ := app.Test(req2)
	if resp2.StatusCode != http.StatusConflict {
		t.Errorf("expected 409, got %d", resp2.StatusCode)
	}
	var ae apiErr
	respBody, _ := io.ReadAll(resp2.Body)
	_ = json.Unmarshal(respBody, &ae)
	if ae.Code != "CONFLICT" {
		t.Errorf("expected code CONFLICT, got %v", ae.Code)
	}
}

func TestRegisterInvalidRef(t *testing.T) {
	app, db := setupTestDB(t)
	defer db.Close()
	truncateTestDB(t, db)

	// 'XXXXXX' pasa el validator (alphanum+len=6) pero el controller
	// lo rechaza porque no está en el alfabeto permitido.
	body := `{"nombre":"Ana Ref","email":"ana.ref@upse.edu.ec","password":"supersecret","ref":"XXXXXX"}`
	req := httptest.NewRequest(http.MethodPost, "/api/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
	var ae apiErr
	respBody, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(respBody, &ae)
	if ae.Code != "VALIDATION_ERROR" {
		t.Errorf("expected code VALIDATION_ERROR, got %v", ae.Code)
	}
}

func TestRegisterInvalidRefNoExiste(t *testing.T) {
	app, db := setupTestDB(t)
	defer db.Close()
	truncateTestDB(t, db)

	// 'ABCDEF' pasa todas las validaciones de forma, pero no existe
	// en la tabla referidos → el repository devuelve ErrCodigoRefInvalido.
	body := `{"nombre":"Ana Ref2","email":"ana.ref2@upse.edu.ec","password":"supersecret","ref":"ABCDEF"}`
	req := httptest.NewRequest(http.MethodPost, "/api/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestRegisterValidation(t *testing.T) {
	app, _ := setupTestDB(t)
	defer func() { /* db.Close no necesario aquí */ }()

	// email vacío → falla validator
	body := `{"nombre":"X","email":"","password":"supersecret"}`
	req := httptest.NewRequest(http.MethodPost, "/api/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestRegisterRateLimit(t *testing.T) {
	app, db := setupTestDB(t)
	defer db.Close()
	truncateTestDB(t, db)

	// 10 registros (límite) deben pasar; el 11 debe rebotar con 429.
	// Cada request usa un email único para que la lógica del rate-limit
	// (basado en IP) sea el único factor limitante, no el dup de email.
	for i := 0; i < 10; i++ {
		body := `{"nombre":"Rate User","email":"r` + strconv.Itoa(i) + `@upse.edu.ec","password":"supersecret"}`
		req := httptest.NewRequest(http.MethodPost, "/api/register", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := app.Test(req)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("iter %d: expected 201, got %d", i, resp.StatusCode)
		}
	}
	// El 11 debería rebotar.
	body := `{"nombre":"R","email":"r11@upse.edu.ec","password":"supersecret"}`
	req := httptest.NewRequest(http.MethodPost, "/api/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", resp.StatusCode)
	}
	var ae apiErr
	respBody, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(respBody, &ae)
	if ae.Code != "RATE_LIMITED" {
		t.Errorf("expected code RATE_LIMITED, got %v", ae.Code)
	}
}

// ─────────────────────────── Productos / Red / Checkout (mock-based) ───────────────────────────

// TestProductos ejercita la vista pública contra la DB real (F2 migró el
// catálogo del mock a PostgreSQL). Truncamos la tabla `productos` en cada
// sub-test para empezar con estado conocido y luego insertamos 4 seeds.
func TestProductos(t *testing.T) {
	app, db := setupTestDB(t)
	defer db.Close()

	t.Run("Get all products", func(t *testing.T) {
		resetProductos(t, db)
		seedProductos(t, db)

		req := httptest.NewRequest(http.MethodGet, "/api/productos", nil)
		resp, _ := app.Test(req)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
		var prods []models.Producto
		body, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(body, &prods)
		if len(prods) != 4 {
			t.Errorf("expected 4 products, got %d", len(prods))
		}
		// El listado público sólo incluye activo=true. Si los seeds están
		// inactivos no aparecerían — los sembramos con activo=true.
		for _, p := range prods {
			if !p.Activo {
				t.Errorf("public list should not include inactive products, got %+v", p)
			}
		}
	})

	t.Run("Get product by ID - invalid ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/productos/abc", nil)
		resp, _ := app.Test(req)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
		var ae apiErr
		body, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(body, &ae)
		if ae.Status != 400 || ae.Code != "VALIDATION_ERROR" || ae.Message != "id inválido" {
			t.Errorf("unexpected error: %+v", ae)
		}
	})

	t.Run("Get product by ID - not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/productos/9999", nil)
		resp, _ := app.Test(req)
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("expected 404, got %d", resp.StatusCode)
		}
		var ae apiErr
		body, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(body, &ae)
		if ae.Status != 404 || ae.Code != "NOT_FOUND" || ae.Message != "producto no encontrado" {
			t.Errorf("unexpected error: %+v", ae)
		}
	})

	t.Run("Get product by ID - found", func(t *testing.T) {
		resetProductos(t, db)
		seedProductos(t, db)

		req := httptest.NewRequest(http.MethodGet, "/api/productos/1", nil)
		resp, _ := app.Test(req)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
		var prod models.Producto
		body, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(body, &prod)
		if prod.ID != 1 || prod.Nombre != "Serum Revitalizante" {
			t.Errorf("unexpected product: %+v", prod)
		}
	})

	t.Run("Get product by alias route /api/producto/:id", func(t *testing.T) {
		resetProductos(t, db)
		seedProductos(t, db)

		req := httptest.NewRequest(http.MethodGet, "/api/producto/2", nil)
		resp, _ := app.Test(req)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
		var prod models.Producto
		body, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(body, &prod)
		if prod.ID != 2 || prod.Nombre != "Crema Hidratante Pro" {
			t.Errorf("unexpected product: %+v", prod)
		}
	})

	t.Run("Inactive products excluded from public list", func(t *testing.T) {
		resetProductos(t, db)
		// Insertamos 2 productos, uno activo y otro no.
		_, err := db.Exec(`INSERT INTO productos (nombre, precio, categoria, stock, img, activo) VALUES ($1,$2,$3,$4,$5,$6)`,
			"Visible", 10.00, "rostro", 5, "https://example.com/v.jpg", true)
		if err != nil {
			t.Fatalf("seed activo: %v", err)
		}
		_, err = db.Exec(`INSERT INTO productos (nombre, precio, categoria, stock, img, activo) VALUES ($1,$2,$3,$4,$5,$6)`,
			"Oculto", 20.00, "rostro", 5, "https://example.com/o.jpg", false)
		if err != nil {
			t.Fatalf("seed inactivo: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/api/productos", nil)
		resp, _ := app.Test(req)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		var prods []models.Producto
		body, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(body, &prods)
		if len(prods) != 1 {
			t.Errorf("expected 1 product (only activo=true), got %d", len(prods))
		}
		if len(prods) > 0 && prods[0].Nombre != "Visible" {
			t.Errorf("unexpected product in list: %+v", prods[0])
		}
	})
}

func TestNetwork(t *testing.T) {
	app := setupTestApp()
	req := httptest.NewRequest(http.MethodGet, "/api/red", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var red []models.Referido
	body, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(body, &red)
	if len(red) != 3 {
		t.Errorf("expected 3 referidos, got %d", len(red))
	}
}

func TestCheckout(t *testing.T) {
	app := setupTestApp()

	t.Run("Sin auth → 401", func(t *testing.T) {
		// F3 migró /api/checkout a RequireAuth — sin token debe ser 401.
		body := `{"nombre":"Test","email":"test@email.com","metodo_pago":"tarjeta","items":[{"id":1,"nombre":"Serum","precio":45,"cantidad":1}]}`
		req := httptest.NewRequest(http.MethodPost, "/api/checkout", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := app.Test(req)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", resp.StatusCode)
		}
	})

	t.Run("Auth + Invalid JSON → 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/checkout", strings.NewReader("invalid"))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", clientAuthHeader(t))
		resp, _ := app.Test(req)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
	})

	t.Run("Auth + Carrito vacío → 400", func(t *testing.T) {
		// Carrito vacío lo valida el controller tras el BodyParser.
		body := `{"nombre":"Test","email":"test@email.com","metodo_pago":"tarjeta","items":[]}`
		req := httptest.NewRequest(http.MethodPost, "/api/checkout", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", clientAuthHeader(t))
		resp, _ := app.Test(req)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
	})

	t.Run("Auth + sin DB → 500", func(t *testing.T) {
		// Pasamos auth pero el repo tiene db=nil → 500 INTERNAL.
		body := `{"nombre":"Test","email":"test@email.com","metodo_pago":"tarjeta","items":[{"id":1,"nombre":"Serum","precio":45,"cantidad":1}]}`
		req := httptest.NewRequest(http.MethodPost, "/api/checkout", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", clientAuthHeader(t))
		resp, _ := app.Test(req)
		if resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("expected 500 (no DB), got %d", resp.StatusCode)
		}
	})
}

// ─────────────────────────── Productos admin (F2) ───────────────────────────
//
// Las pruebas siguientes ejercitan /api/admin/productos, que está protegida
// por RequireAuth + RequireAdmin. Firma manual con middleware.SignJWT para
// evitar acoplar al endpoint /api/login (probado en TestLogin).
//
// Patrón:
//
//	1. resetProductos() — vacía la tabla `productos`.
//	2. seedProductos()  — inserta 4 filas conocidas (coinciden con 005_seed).
//	3. Petición HTTP con Authorization: Bearer <jwt admin>.

// adminAuthHeader devuelve el header Authorization con un JWT de admin
// firmado con el secret de tests. El rol y email son los del seed
// admin@upse.edu.ec creado por ensureSeed.
func adminAuthHeader(t *testing.T) string {
	t.Helper()
	tok, err := middleware.SignJWT(testJWTSecret, middleware.JWTClaims{
		Sub:   1,
		Email: "admin@upse.edu.ec",
		Rol:   "admin",
		Exp:   time.Now().Add(1 * time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("SignJWT admin: %v", err)
	}
	return "Bearer " + tok
}

// clientAuthHeader devuelve el header Authorization con un JWT de cliente
// (rol=cliente). Útil para probar que /api/admin/* rechaza no-admins.
func clientAuthHeader(t *testing.T) string {
	t.Helper()
	tok, err := middleware.SignJWT(testJWTSecret, middleware.JWTClaims{
		Sub:   2,
		Email: "cliente@upse.edu.ec",
		Rol:   "cliente",
		Exp:   time.Now().Add(1 * time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("SignJWT cliente: %v", err)
	}
	return "Bearer " + tok
}

func TestAdminCreate(t *testing.T) {
	app, db := setupTestDB(t)
	defer db.Close()
	resetProductos(t, db)

	body := `{"nombre":"Nuevo Sérum","precio":99.99,"categoria":"rostro","stock":10,"img":"https://example.com/x.jpg"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/productos", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", adminAuthHeader(t))
	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	var prod models.Producto
	respBody, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(respBody, &prod)
	if prod.ID == 0 {
		t.Errorf("expected non-zero id, got 0")
	}
	if prod.Nombre != "Nuevo Sérum" {
		t.Errorf("unexpected nombre: %q", prod.Nombre)
	}
	if !prod.Activo {
		t.Errorf("expected activo=true on creation, got false")
	}

	// GET público debe devolverlo (la creación invalida el caché).
	getReq := httptest.NewRequest(http.MethodGet, "/api/productos", nil)
	getResp, _ := app.Test(getReq)
	var list []models.Producto
	listBody, _ := io.ReadAll(getResp.Body)
	_ = json.Unmarshal(listBody, &list)
	found := false
	for _, p := range list {
		if p.ID == prod.ID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("created product %d not present in public list", prod.ID)
	}
}

func TestAdminCreateForbidden(t *testing.T) {
	app, db := setupTestDB(t)
	defer db.Close()
	resetProductos(t, db)

	body := `{"nombre":"X","precio":10,"categoria":"rostro","stock":1}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/productos", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", clientAuthHeader(t)) // rol=cliente
	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}
	var ae apiErr
	respBody, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(respBody, &ae)
	if ae.Code != "FORBIDDEN" {
		t.Errorf("expected FORBIDDEN, got %q", ae.Code)
	}
}

func TestAdminCreateNoAuth(t *testing.T) {
	app, db := setupTestDB(t)
	defer db.Close()
	resetProductos(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/productos",
		strings.NewReader(`{"nombre":"X","precio":10,"categoria":"rostro","stock":1}`))
	req.Header.Set("Content-Type", "application/json")
	// sin Authorization
	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestAdminCreateValidation(t *testing.T) {
	app, db := setupTestDB(t)
	defer db.Close()
	resetProductos(t, db)

	cases := []struct {
		name string
		body string
	}{
		{"missing nombre", `{"precio":10,"categoria":"rostro","stock":1}`},
		{"empty precio", `{"nombre":"X","categoria":"rostro","stock":1}`},
		{"negative precio", `{"nombre":"X","precio":-5,"categoria":"rostro","stock":1}`},
		{"missing categoria", `{"nombre":"X","precio":10,"stock":1}`},
		{"negative stock", `{"nombre":"X","precio":10,"categoria":"rostro","stock":-1}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/admin/productos", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", adminAuthHeader(t))
			resp, _ := app.Test(req)
			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", resp.StatusCode)
			}
			var ae apiErr
			respBody, _ := io.ReadAll(resp.Body)
			_ = json.Unmarshal(respBody, &ae)
			if ae.Code != "VALIDATION_ERROR" {
				t.Errorf("expected VALIDATION_ERROR, got %q", ae.Code)
			}
		})
	}
}

func TestAdminUpdate(t *testing.T) {
	app, db := setupTestDB(t)
	defer db.Close()
	resetProductos(t, db)
	seedProductos(t, db)

	// PUT parcial: sólo cambiamos precio y stock.
	body := `{"precio":99.99,"stock":42}`
	req := httptest.NewRequest(http.MethodPut, "/api/admin/productos/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", adminAuthHeader(t))
	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var prod models.Producto
	respBody, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(respBody, &prod)
	if prod.Precio != 99.99 {
		t.Errorf("expected precio 99.99, got %v", prod.Precio)
	}
	if prod.Stock != 42 {
		t.Errorf("expected stock 42, got %d", prod.Stock)
	}
	// campos no enviados deben quedar igual
	if prod.Nombre != "Serum Revitalizante" {
		t.Errorf("nombre should not change, got %q", prod.Nombre)
	}
	if prod.Categoria != "rostro" {
		t.Errorf("categoria should not change, got %q", prod.Categoria)
	}

	// PUT a id inexistente → 404
	req2 := httptest.NewRequest(http.MethodPut, "/api/admin/productos/9999",
		strings.NewReader(`{"precio":1}`))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", adminAuthHeader(t))
	resp2, _ := app.Test(req2)
	if resp2.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 on missing id, got %d", resp2.StatusCode)
	}
}

func TestAdminSoftDelete(t *testing.T) {
	app, db := setupTestDB(t)
	defer db.Close()
	resetProductos(t, db)
	seedProductos(t, db)

	// Insertar un pedido + pedido_item referenciando producto 1 para
	// verificar que el soft-delete preserva la FK.
	var pedidoID int
	if err := db.QueryRow(
		`INSERT INTO pedidos (email_usuario, total, metodo_pago, estado) VALUES ($1,$2,$3,'pendiente') RETURNING id`,
		"fakeperson@upse.edu.ec", 45.00, "tarjeta",
	).Scan(&pedidoID); err != nil {
		t.Fatalf("seed pedido: %v", err)
	}
	var itemID int
	if err := db.QueryRow(
		`INSERT INTO pedido_items (pedido_id, producto_id, nombre, precio, cantidad) VALUES ($1,$2,$3,$4,$5) RETURNING id`,
		pedidoID, 1, "Serum Revitalizante", 45.00, 1,
	).Scan(&itemID); err != nil {
		t.Fatalf("seed pedido_item: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/productos/1", nil)
	req.Header.Set("Authorization", adminAuthHeader(t))
	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	// El producto en DB debe seguir existiendo pero con activo=false.
	var activo bool
	if err := db.QueryRow(`SELECT activo FROM productos WHERE id = 1`).Scan(&activo); err != nil {
		t.Fatalf("query producto: %v", err)
	}
	if activo {
		t.Errorf("expected activo=false after soft-delete, got true")
	}

	// GET público lo debe excluir.
	getReq := httptest.NewRequest(http.MethodGet, "/api/productos", nil)
	getResp, _ := app.Test(getReq)
	var list []models.Producto
	listBody, _ := io.ReadAll(getResp.Body)
	_ = json.Unmarshal(listBody, &list)
	for _, p := range list {
		if p.ID == 1 {
			t.Errorf("soft-deleted product still in public list: %+v", p)
		}
	}

	// El pedido_item sigue intacto (FK preservada).
	var itemExists bool
	if err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM pedido_items WHERE id = $1)`, itemID).Scan(&itemExists); err != nil {
		t.Fatalf("query pedido_item: %v", err)
	}
	if !itemExists {
		t.Errorf("pedido_item was deleted (should be preserved by soft-delete)")
	}

	// Soft-delete idempotente: borrar dos veces no debe fallar.
	req2 := httptest.NewRequest(http.MethodDelete, "/api/admin/productos/1", nil)
	req2.Header.Set("Authorization", adminAuthHeader(t))
	resp2, _ := app.Test(req2)
	if resp2.StatusCode != http.StatusNoContent {
		t.Errorf("second DELETE should be idempotent (204), got %d", resp2.StatusCode)
	}

	// Soft-delete de id inexistente → 404.
	req3 := httptest.NewRequest(http.MethodDelete, "/api/admin/productos/9999", nil)
	req3.Header.Set("Authorization", adminAuthHeader(t))
	resp3, _ := app.Test(req3)
	if resp3.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 on missing id, got %d", resp3.StatusCode)
	}
}

func TestAdminListPagination(t *testing.T) {
	app, db := setupTestDB(t)
	defer db.Close()
	resetProductos(t, db)

	// Insertar 5 productos manualmente.
	for i := 1; i <= 5; i++ {
		_, err := db.Exec(
			`INSERT INTO productos (nombre, precio, categoria, stock, img, activo) VALUES ($1,$2,$3,$4,$5,true)`,
			fmt.Sprintf("Producto %02d", i), float64(i), "rostro", i, "https://example.com/"+strconv.Itoa(i),
		)
		if err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
	}

	// ?page=2&limit=2 debe devolver productos 3 y 4.
	req := httptest.NewRequest(http.MethodGet, "/api/admin/productos?page=2&limit=2", nil)
	req.Header.Set("Authorization", adminAuthHeader(t))
	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var res struct {
		Data  []models.Producto `json:"data"`
		Page  int               `json:"page"`
		Limit int               `json:"limit"`
		Total int               `json:"total"`
	}
	respBody, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(respBody, &res)
	if res.Page != 2 || res.Limit != 2 || res.Total != 5 {
		t.Errorf("unexpected meta: page=%d limit=%d total=%d", res.Page, res.Limit, res.Total)
	}
	if len(res.Data) != 2 {
		t.Errorf("expected 2 items, got %d", len(res.Data))
	}
	if len(res.Data) >= 1 && res.Data[0].Nombre != "Producto 03" {
		t.Errorf("expected Producto 03 first, got %q", res.Data[0].Nombre)
	}
}

func TestAdminListSearch(t *testing.T) {
	app, db := setupTestDB(t)
	defer db.Close()
	resetProductos(t, db)

	// Sembramos dos productos diferenciables por nombre.
	for _, p := range []struct {
		nombre string
		cat    string
	}{
		{"Serum Especial X", "rostro"},
		{"Crema Hidratante Y", "rostro"},
		{"Tónico Z", "tratamiento"},
	} {
		_, err := db.Exec(
			`INSERT INTO productos (nombre, precio, categoria, stock, img, activo) VALUES ($1,$2,$3,$4,$5,true)`,
			p.nombre, 10.0, p.cat, 5, "https://example.com/x.jpg",
		)
		if err != nil {
			t.Fatalf("seed %s: %v", p.nombre, err)
		}
	}

	// q=serum debe devolver sólo Serum Especial X.
	req := httptest.NewRequest(http.MethodGet, "/api/admin/productos?q=serum", nil)
	req.Header.Set("Authorization", adminAuthHeader(t))
	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var res struct {
		Data  []models.Producto `json:"data"`
		Total int               `json:"total"`
	}
	respBody, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(respBody, &res)
	if res.Total != 1 || len(res.Data) != 1 {
		t.Errorf("expected 1 match, got total=%d len=%d", res.Total, len(res.Data))
	}
	if len(res.Data) == 1 && res.Data[0].Nombre != "Serum Especial X" {
		t.Errorf("expected Serum Especial X, got %q", res.Data[0].Nombre)
	}

	// q=rostro debe devolver los 2 con categoria=rostro.
	req2 := httptest.NewRequest(http.MethodGet, "/api/admin/productos?q=rostro", nil)
	req2.Header.Set("Authorization", adminAuthHeader(t))
	resp2, _ := app.Test(req2)
	var res2 struct {
		Data  []models.Producto `json:"data"`
		Total int               `json:"total"`
	}
	respBody2, _ := io.ReadAll(resp2.Body)
	_ = json.Unmarshal(respBody2, &res2)
	if res2.Total != 2 {
		t.Errorf("expected 2 matches in categoria=rostro, got %d", res2.Total)
	}
}

func TestAdminListActivoFilter(t *testing.T) {
	app, db := setupTestDB(t)
	defer db.Close()
	resetProductos(t, db)

	_, _ = db.Exec(`INSERT INTO productos (nombre, precio, categoria, stock, img, activo) VALUES ('Activo',  10, 'rostro', 1, '', true)`)
	_, _ = db.Exec(`INSERT INTO productos (nombre, precio, categoria, stock, img, activo) VALUES ('Inactivo', 10, 'rostro', 1, '', false)`)

	// activo=true → 1 fila.
	req := httptest.NewRequest(http.MethodGet, "/api/admin/productos?activo=true", nil)
	req.Header.Set("Authorization", adminAuthHeader(t))
	resp, _ := app.Test(req)
	var res struct {
		Total int `json:"total"`
	}
	respBody, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(respBody, &res)
	if res.Total != 1 {
		t.Errorf("expected 1 activo=true, got %d", res.Total)
	}

	// activo=false → 1 fila.
	req2 := httptest.NewRequest(http.MethodGet, "/api/admin/productos?activo=false", nil)
	req2.Header.Set("Authorization", adminAuthHeader(t))
	resp2, _ := app.Test(req2)
	var res2 struct {
		Total int `json:"total"`
	}
	respBody2, _ := io.ReadAll(resp2.Body)
	_ = json.Unmarshal(respBody2, &res2)
	if res2.Total != 1 {
		t.Errorf("expected 1 activo=false, got %d", res2.Total)
	}

	// sin filtro → 2 filas.
	req3 := httptest.NewRequest(http.MethodGet, "/api/admin/productos", nil)
	req3.Header.Set("Authorization", adminAuthHeader(t))
	resp3, _ := app.Test(req3)
	var res3 struct {
		Total int `json:"total"`
	}
	respBody3, _ := io.ReadAll(resp3.Body)
	_ = json.Unmarshal(respBody3, &res3)
	if res3.Total != 2 {
		t.Errorf("expected 2 sin filtro, got %d", res3.Total)
	}
}

// ─────────────────────────── Cache invalidation (F2) ───────────────────────────

// recordingCache implementa interfaces.CacheService registrando Get/Set/Delete
// para validar que los handlers del catálogo sí escriben y borran la clave
// correcta. Es in-memory y no toca Redis.
type recordingCache struct {
	data    map[string]string
	deletes map[string]int
	gets    map[string]int
}

func newRecordingCache() *recordingCache {
	return &recordingCache{
		data:    map[string]string{},
		deletes: map[string]int{},
		gets:    map[string]int{},
	}
}

func (rc *recordingCache) Get(_ context.Context, k string) (string, error) {
	rc.gets[k]++
	if v, ok := rc.data[k]; ok {
		return v, nil
	}
	return "", redis.Nil
}
func (rc *recordingCache) Set(_ context.Context, k string, v interface{}, _ time.Duration) error {
	switch s := v.(type) {
	case string:
		rc.data[k] = s
	case []byte:
		rc.data[k] = string(s)
	default:
		// serializar a JSON como fallback
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		rc.data[k] = string(b)
	}
	return nil
}
func (rc *recordingCache) Delete(_ context.Context, k string) error {
	rc.deletes[k]++
	delete(rc.data, k)
	return nil
}

func TestProductoCacheInvalidation(t *testing.T) {
	// Reconstruimos una app con recordingCache para verificar que el handler
	// escribe `productos:all` en el primer GET y la borra tras un write admin.
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
	resetProductos(t, db)
	seedProductos(t, db)

	repo := repository.NewRepository(db)
	cache := newRecordingCache()
	ctrl := routes.Controllers{
		Auth:     controllers.NewAuthController(repo),
		Producto: controllers.NewProductoController(repo, cache),
		Network:  controllers.NewNetworkController(repo, cache),
		Checkout: controllers.NewCheckoutController(repo, cache),
		Pedidos:  controllers.NewPedidosController(repo, cache),
	}
	app := fiber.New()
	app.Use(middleware.SecurityHeaders())
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(middleware.JWTSecretKey, testJWTSecret)
		return c.Next()
	})
	routes.SetupRoutes(app, ctrl, testJWTSecret)

	// 1er GET → cache miss + Set.
	req := httptest.NewRequest(http.MethodGet, "/api/productos", nil)
	if _, err := app.Test(req); err != nil {
		t.Fatalf("GET: %v", err)
	}
	if cache.gets["productos:all"] == 0 {
		t.Errorf("expected GET to consult cache")
	}
	if _, ok := cache.data["productos:all"]; !ok {
		t.Errorf("expected cache key productos:all to be set after GET")
	}

	// 2do GET → cache hit.
	req2 := httptest.NewRequest(http.MethodGet, "/api/productos", nil)
	if _, err := app.Test(req2); err != nil {
		t.Fatalf("GET2: %v", err)
	}
	if cache.gets["productos:all"] < 2 {
		t.Errorf("expected cache to be consulted again")
	}

	// POST admin → debe invalidar cache.
	postBody := `{"nombre":"Nuevo Cache","precio":10,"categoria":"rostro","stock":1}`
	postReq := httptest.NewRequest(http.MethodPost, "/api/admin/productos", strings.NewReader(postBody))
	postReq.Header.Set("Content-Type", "application/json")
	postReq.Header.Set("Authorization", adminAuthHeader(t))
	if _, err := app.Test(postReq); err != nil {
		t.Fatalf("POST: %v", err)
	}
	if cache.deletes["productos:all"] == 0 {
		t.Errorf("expected cache key productos:all to be deleted after POST")
	}

	// PUT admin → debe invalidar cache también.
	putReq := httptest.NewRequest(http.MethodPut, "/api/admin/productos/1", strings.NewReader(`{"precio":1.5}`))
	putReq.Header.Set("Content-Type", "application/json")
	putReq.Header.Set("Authorization", adminAuthHeader(t))
	if _, err := app.Test(putReq); err != nil {
		t.Fatalf("PUT: %v", err)
	}

	// DELETE admin → debe invalidar cache también.
	delReq := httptest.NewRequest(http.MethodDelete, "/api/admin/productos/2", nil)
	delReq.Header.Set("Authorization", adminAuthHeader(t))
	if _, err := app.Test(delReq); err != nil {
		t.Fatalf("DELETE: %v", err)
	}
	if cache.deletes["productos:all"] < 3 {
		t.Errorf("expected 3 deletes on cache key productos:all (POST+PUT+DELETE), got %d", cache.deletes["productos:all"])
	}
}

// ─────────────────────────── helpers ───────────────────────────

// bcryptHashForTest genera un hash bcrypt cost 12 de plaintext. Se usa
// en los tests de F1 para sembrar usuarios adicionales con la misma
// fortaleza que el SQL inicial (005_fundaciones.sql).
func bcryptHashForTest(t *testing.T, plaintext string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(plaintext), 12)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	return string(h)
}

// resetProductos borra la tabla `productos` antes de cada sub-test que
// requiera estado conocido. Limpia primero pedido_items porque la FK
// pedido_items.producto_id tiene ON DELETE RESTRICT. También borramos
// pedidos (cabecera) para no dejar huérfanos que contaminen otros tests.
// Es seguro aunque las tablas estén vacías.
func resetProductos(t *testing.T, db *sql.DB) {
	t.Helper()
	stmts := []string{
		`DELETE FROM pedido_items`,
		`DELETE FROM pedidos`,
		`DELETE FROM productos`,
		`ALTER SEQUENCE productos_id_seq RESTART WITH 1`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("[resetProductos] %s: %v", s, err)
		}
	}
}

// seedProductos inserta las 4 filas que el seed 005_fundaciones.sql
// crea en producción. Coinciden en id/nombre/categoria/stock/img con el
// entorno real — los tests que esperan estos nombres no se rompen al
// cambiar de entorno. Activo=true para que la vista pública los vea.
func seedProductos(t *testing.T, db *sql.DB) {
	t.Helper()
	seeds := []struct {
		nombre    string
		precio    float64
		categoria string
		stock     int
		img       string
	}{
		{"Serum Revitalizante", 45.00, "rostro", 50, "https://picsum.photos/seed/serum/150"},
		{"Crema Hidratante Pro", 32.50, "rostro", 40, "https://picsum.photos/seed/crema/150"},
		{"Tónico Purificante", 28.00, "rostro", 35, "https://picsum.photos/seed/tonico/150"},
		{"Mascarilla Nocturna", 50.00, "tratamiento", 25, "https://picsum.photos/seed/mascarilla/150"},
	}
	for _, s := range seeds {
		_, err := db.Exec(
			`INSERT INTO productos (nombre, precio, categoria, stock, img, activo)
			 VALUES ($1, $2, $3, $4, $5, true)`,
			s.nombre, s.precio, s.categoria, s.stock, s.img,
		)
		if err != nil {
			t.Fatalf("[seedProductos] %s: %v", s.nombre, err)
		}
	}
}

// ─────────────────────────── Pedidos (F3) ───────────────────────────

// authHeaderFor devuelve "Bearer <jwt>" firmado con el secret de tests
// para un usuario arbitrario (sub/email/rol). A diferencia de los
// helpers adminAuthHeader/clientAuthHeader (que están hard-coded a los
// seeds), este helper deja parametrizar el id y email — necesario para
// los tests F3 que crean usuarios extra dinámicamente.
func authHeaderFor(t *testing.T, sub int, email, rol string) string {
	t.Helper()
	tok, err := middleware.SignJWT(testJWTSecret, middleware.JWTClaims{
		Sub:   sub,
		Email: email,
		Rol:   rol,
		Exp:   time.Now().Add(1 * time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("SignJWT: %v", err)
	}
	return "Bearer " + tok
}

// doJSON ejecuta un request HTTP contra la app Fiber y devuelve status +
// body. Encapsula el patrón repetido en los tests de F3.
func doJSON(t *testing.T, app *fiber.App, method, path, body, authHeader string) (int, []byte) {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, path, nil)
	} else {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	}
	if authHeader != "" {
		r.Header.Set("Authorization", authHeader)
	}
	resp, err := app.Test(r, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, b
}

// seedUsuario inserta un usuario extra (no-seed) con referido_nulo para
// tests que necesiten un cliente "limpio" (sin pedidos previos).
//
// Si `linkUpline` es true, además crea un nodo en `referidos` con
// parent_id=1 (TU) y actualiza usuarios.referido_id, de modo que
// ActualizarComisiones pueda caminar el árbol y registrar comisiones.
// Devuelve el id asignado.
func seedUsuario(t *testing.T, db *sql.DB, email, rol string, linkUpline bool) int {
	t.Helper()
	hash := bcryptHashForTest(t, "12345678")
	var id int
	err := db.QueryRow(
		`INSERT INTO usuarios (email, password_hash, rol, nombre, activo)
		 VALUES ($1, $2, $3, $4, true)
		 ON CONFLICT (email) DO UPDATE SET activo = true
		 RETURNING id`,
		email, hash, rol, "Test User",
	).Scan(&id)
	if err != nil {
		t.Fatalf("[seedUsuario] %s: %v", email, err)
	}
	if linkUpline {
		// Crear nodo de referido único para este usuario, hijo de TU (id=1).
		// El código se genera determinísticamente a partir del id para que
		// truncateTestDB + re-run no choquen con UNIQUE(codigo).
		codigo := fmt.Sprintf("T%07d", id)
		var rid int
		if err := db.QueryRow(
			`INSERT INTO referidos (codigo, parent_id, usuario_id)
			 VALUES ($1, 1, $2) RETURNING id`,
			codigo, id,
		).Scan(&rid); err != nil {
			t.Fatalf("[seedUsuario] insert referidos para %s: %v", email, err)
		}
		if _, err := db.Exec(
			`UPDATE usuarios SET referido_id = $1 WHERE id = $2`,
			rid, id,
		); err != nil {
			t.Fatalf("[seedUsuario] update usuarios.referido_id para %s: %v", email, err)
		}
	}
	return id
}

// seedPedido crea un pedido + items directamente en DB (sin pasar por el
// handler de checkout). Útil para los tests de PATCH y de detalle que
// necesitan un pedido preexistente en un estado concreto. Devuelve el id
// del pedido.
//
// Items recibe {producto_id, cantidad} para poder controlar el stock
// inicial que se descuenta.
func seedPedido(t *testing.T, db *sql.DB, usuarioID int, email string, estado string,
	items []struct{ ProductoID, Cantidad int }) int {
	t.Helper()

	var total float64
	type linea struct {
		productoID, cantidad int
		nombre              string
		precio              float64
	}
	lineas := make([]linea, 0, len(items))
	for _, it := range items {
		var nombre string
		var precio float64
		if err := db.QueryRow(
			`SELECT nombre, precio FROM productos WHERE id = $1`,
			it.ProductoID,
		).Scan(&nombre, &precio); err != nil {
			t.Fatalf("[seedPedido] leyendo producto %d: %v", it.ProductoID, err)
		}
		lineas = append(lineas, linea{
			productoID: it.ProductoID, cantidad: it.Cantidad,
			nombre: nombre, precio: precio,
		})
		total += precio * float64(it.Cantidad)
	}

	var pedidoID int
	if err := db.QueryRow(
		`INSERT INTO pedidos (email_usuario, usuario_id, total, metodo_pago, estado)
		 VALUES ($1, $2, $3, 'tarjeta', $4) RETURNING id`,
		email, usuarioID, total, estado,
	).Scan(&pedidoID); err != nil {
		t.Fatalf("[seedPedido] insert pedido: %v", err)
	}
	for _, l := range lineas {
		if _, err := db.Exec(
			`INSERT INTO pedido_items (pedido_id, producto_id, nombre, precio, cantidad)
			 VALUES ($1, $2, $3, $4, $5)`,
			pedidoID, l.productoID, l.nombre, l.precio, l.cantidad,
		); err != nil {
			t.Fatalf("[seedPedido] insert item: %v", err)
		}
	}
	return pedidoID
}

// ── Tests F3 ───────────────────────────────────────────────

func TestMisPedidos(t *testing.T) {
	app, db := setupTestDB(t)
	defer db.Close()
	resetProductos(t, db)
	seedProductos(t, db)

	// 2 clientes con pedidos propios.
	uidA := seedUsuario(t, db, "clienteA@upse.edu.ec", "cliente", false)
	uidB := seedUsuario(t, db, "clienteB@upse.edu.ec", "cliente", false)
	_ = seedPedido(t, db, uidA, "clienteA@upse.edu.ec", "pendiente",
		[]struct{ ProductoID, Cantidad int }{{1, 1}})
	_ = seedPedido(t, db, uidA, "clienteA@upse.edu.ec", "pendiente",
		[]struct{ ProductoID, Cantidad int }{{2, 1}})
	_ = seedPedido(t, db, uidB, "clienteB@upse.edu.ec", "pendiente",
		[]struct{ ProductoID, Cantidad int }{{3, 1}})

	authA := authHeaderFor(t, uidA, "clienteA@upse.edu.ec", "cliente")
	authB := authHeaderFor(t, uidB, "clienteB@upse.edu.ec", "cliente")

	t.Run("Cliente A ve sus 2 pedidos", func(t *testing.T) {
		status, body := doJSON(t, app, http.MethodGet, "/api/pedidos/mios", "", authA)
		if status != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", status, body)
		}
		var res struct {
			Data  []models.PedidoResumen `json:"data"`
			Total int                    `json:"total"`
		}
		_ = json.Unmarshal(body, &res)
		if res.Total != 2 || len(res.Data) != 2 {
			t.Errorf("esperaba 2 pedidos del cliente A, got total=%d len=%d",
				res.Total, len(res.Data))
		}
	})

	t.Run("Cliente B ve su único pedido", func(t *testing.T) {
		status, body := doJSON(t, app, http.MethodGet, "/api/pedidos/mios", "", authB)
		if status != http.StatusOK {
			t.Fatalf("expected 200, got %d", status)
		}
		var res struct {
			Data  []models.PedidoResumen `json:"data"`
			Total int                    `json:"total"`
		}
		_ = json.Unmarshal(body, &res)
		if res.Total != 1 || len(res.Data) != 1 {
			t.Errorf("esperaba 1 pedido del cliente B, got total=%d len=%d",
				res.Total, len(res.Data))
		}
	})
}

func TestMisPedidosSinAuth(t *testing.T) {
	app, db := setupTestDB(t)
	defer db.Close()

	status, body := doJSON(t, app, http.MethodGet, "/api/pedidos/mios", "", "")
	if status != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d body=%s", status, body)
	}
}

func TestPedidosAdmin(t *testing.T) {
	app, db := setupTestDB(t)
	defer db.Close()
	resetProductos(t, db)
	seedProductos(t, db)

	uidC := seedUsuario(t, db, "clienteC@upse.edu.ec", "cliente", false)
	// 3 pedidos de clienteC + 1 pedido admin.
	_ = seedPedido(t, db, uidC, "clienteC@upse.edu.ec", "pendiente",
		[]struct{ ProductoID, Cantidad int }{{1, 1}})
	_ = seedPedido(t, db, uidC, "clienteC@upse.edu.ec", "preparando",
		[]struct{ ProductoID, Cantidad int }{{2, 1}})
	_ = seedPedido(t, db, uidC, "clienteC@upse.edu.ec", "cancelado",
		[]struct{ ProductoID, Cantidad int }{{3, 1}})
	_ = seedPedido(t, db, 1, "admin@upse.edu.ec", "entregado",
		[]struct{ ProductoID, Cantidad int }{{4, 1}})

	adminAuth := adminAuthHeader(t)
	clientAuth := clientAuthHeader(t)

	t.Run("Admin ve todos (sin filtro)", func(t *testing.T) {
		status, body := doJSON(t, app, http.MethodGet, "/api/admin/pedidos", "", adminAuth)
		if status != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", status, body)
		}
		var res struct {
			Data  []models.PedidoResumen `json:"data"`
			Total int                    `json:"total"`
		}
		_ = json.Unmarshal(body, &res)
		if res.Total != 4 {
			t.Errorf("esperaba 4 pedidos total, got %d", res.Total)
		}
	})

	t.Run("Admin filtra por estado=cancelado", func(t *testing.T) {
		status, body := doJSON(t, app, http.MethodGet, "/api/admin/pedidos?estado=cancelado", "", adminAuth)
		if status != http.StatusOK {
			t.Fatalf("expected 200, got %d", status)
		}
		var res struct {
			Data  []models.PedidoResumen `json:"data"`
			Total int                    `json:"total"`
		}
		_ = json.Unmarshal(body, &res)
		if res.Total != 1 {
			t.Errorf("esperaba 1 pedido cancelado, got %d", res.Total)
		}
		if len(res.Data) >= 1 && res.Data[0].Estado != "cancelado" {
			t.Errorf("esperaba estado cancelado, got %q", res.Data[0].Estado)
		}
	})

	t.Run("Cliente no-admin → 403", func(t *testing.T) {
		status, body := doJSON(t, app, http.MethodGet, "/api/admin/pedidos", "", clientAuth)
		if status != http.StatusForbidden {
			t.Errorf("esperaba 403, got %d body=%s", status, body)
		}
	})

	t.Run("Sin auth → 401", func(t *testing.T) {
		status, _ := doJSON(t, app, http.MethodGet, "/api/admin/pedidos", "", "")
		if status != http.StatusUnauthorized {
			t.Errorf("esperaba 401, got %d", status)
		}
	})
}

func TestPedidoDetalle(t *testing.T) {
	app, db := setupTestDB(t)
	defer db.Close()
	resetProductos(t, db)
	seedProductos(t, db)

	uidA := seedUsuario(t, db, "clienteDetA@upse.edu.ec", "cliente", false)
	uidB := seedUsuario(t, db, "clienteDetB@upse.edu.ec", "cliente", false)
	pedidoA := seedPedido(t, db, uidA, "clienteDetA@upse.edu.ec", "pendiente",
		[]struct{ ProductoID, Cantidad int }{{1, 2}, {2, 1}})

	authA := authHeaderFor(t, uidA, "clienteDetA@upse.edu.ec", "cliente")
	authB := authHeaderFor(t, uidB, "clienteDetB@upse.edu.ec", "cliente")
	adminAuth := adminAuthHeader(t)

	t.Run("Dueño ve el detalle", func(t *testing.T) {
		path := fmt.Sprintf("/api/pedidos/%d", pedidoA)
		status, body := doJSON(t, app, http.MethodGet, path, "", authA)
		if status != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", status, body)
		}
		var det models.PedidoDetalle
		_ = json.Unmarshal(body, &det)
		if det.ID != pedidoA {
			t.Errorf("id esperado %d, got %d", pedidoA, det.ID)
		}
		if len(det.Items) != 2 {
			t.Errorf("esperaba 2 items, got %d", len(det.Items))
		}
	})

	t.Run("Admin ve cualquier pedido", func(t *testing.T) {
		path := fmt.Sprintf("/api/pedidos/%d", pedidoA)
		status, body := doJSON(t, app, http.MethodGet, path, "", adminAuth)
		if status != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", status, body)
		}
	})

	t.Run("Otro cliente → 404 (no exponer existencia)", func(t *testing.T) {
		path := fmt.Sprintf("/api/pedidos/%d", pedidoA)
		status, body := doJSON(t, app, http.MethodGet, path, "", authB)
		if status != http.StatusNotFound {
			t.Errorf("esperaba 404, got %d body=%s", status, body)
		}
	})
}

func TestPedidoDetalleNoExiste(t *testing.T) {
	app, db := setupTestDB(t)
	defer db.Close()

	uidX := seedUsuario(t, db, "cliente404@upse.edu.ec", "cliente", false)
	authX := authHeaderFor(t, uidX, "cliente404@upse.edu.ec", "cliente")

	status, body := doJSON(t, app, http.MethodGet, "/api/pedidos/999999", "", authX)
	if status != http.StatusNotFound {
		t.Errorf("esperaba 404, got %d body=%s", status, body)
	}
}

func TestCambiarEstadoHappy(t *testing.T) {
	app, db := setupTestDB(t)
	defer db.Close()
	resetProductos(t, db)
	seedProductos(t, db)

	uid := seedUsuario(t, db, "clienteHappy@upse.edu.ec", "cliente", false)
	pedidoID := seedPedido(t, db, uid, "clienteHappy@upse.edu.ec", "pendiente",
		[]struct{ ProductoID, Cantidad int }{{1, 1}})
	adminAuth := adminAuthHeader(t)

	path := fmt.Sprintf("/api/admin/pedidos/%d/estado", pedidoID)
	for _, estado := range []string{"preparando", "enviado", "entregado"} {
		body := fmt.Sprintf(`{"estado":"%s"}`, estado)
		status, b := doJSON(t, app, http.MethodPatch, path, body, adminAuth)
		if status != http.StatusNoContent {
			t.Errorf("PATCH a %s: expected 204, got %d body=%s", estado, status, b)
		}
	}

	// Verificación final en DB.
	var estadoActual string
	if err := db.QueryRow(`SELECT estado FROM pedidos WHERE id = $1`, pedidoID).Scan(&estadoActual); err != nil {
		t.Fatalf("leyendo estado final: %v", err)
	}
	if estadoActual != "entregado" {
		t.Errorf("estado final esperado=entregado, got=%q", estadoActual)
	}
}

func TestCambiarEstadoInvalido(t *testing.T) {
	app, db := setupTestDB(t)
	defer db.Close()
	resetProductos(t, db)
	seedProductos(t, db)

	uid := seedUsuario(t, db, "clienteInv@upse.edu.ec", "cliente", false)
	// Pedido ya en 'entregado' → intentar volver a 'pendiente' es ilegal.
	pedidoID := seedPedido(t, db, uid, "clienteInv@upse.edu.ec", "entregado",
		[]struct{ ProductoID, Cantidad int }{{1, 1}})
	adminAuth := adminAuthHeader(t)

	path := fmt.Sprintf("/api/admin/pedidos/%d/estado", pedidoID)
	body := `{"estado":"pendiente"}`
	status, respBody := doJSON(t, app, http.MethodPatch, path, body, adminAuth)
	if status != http.StatusConflict {
		t.Fatalf("esperaba 409, got %d body=%s", status, respBody)
	}
	var ae apiErr
	_ = json.Unmarshal(respBody, &ae)
	if ae.Code != "INVALID_TRANSITION" {
		t.Errorf("esperaba code=INVALID_TRANSITION, got %q", ae.Code)
	}
}

func TestCancelarDevuelveStock(t *testing.T) {
	app, db := setupTestDB(t)
	defer db.Close()
	resetProductos(t, db)
	seedProductos(t, db)

	uid := seedUsuario(t, db, "clienteStock@upse.edu.ec", "cliente", false)

	// Stock inicial del producto 1 (Serum) = 50.
	// Crear pedido vía POST /api/checkout (usa CreatePedidoConUsuario,
	// que descuenta stock en la tx).
	authC := authHeaderFor(t, uid, "clienteStock@upse.edu.ec", "cliente")
	checkoutBody := `{
		"nombre":"Cliente Stock",
		"email":"clienteStock@upse.edu.ec",
		"direccion":"Calle 1",
		"ciudad":"Loja",
		"telefono":"0999999999",
		"metodo_pago":"tarjeta",
		"items":[{"id":1,"nombre":"Serum Revitalizante","precio":45.0,"cantidad":3}]
	}`
	status, body := doJSON(t, app, http.MethodPost, "/api/checkout", checkoutBody, authC)
	if status != http.StatusCreated {
		t.Fatalf("checkout: expected 201, got %d body=%s", status, body)
	}
	var det models.PedidoDetalle
	_ = json.Unmarshal(body, &det)

	// Verificar stock descontado: 50 - 3 = 47.
	var stockAfterCheckout int
	if err := db.QueryRow(`SELECT stock FROM productos WHERE id = 1`).Scan(&stockAfterCheckout); err != nil {
		t.Fatalf("leyendo stock post-checkout: %v", err)
	}
	if stockAfterCheckout != 47 {
		t.Errorf("stock post-checkout esperado=47, got=%d", stockAfterCheckout)
	}

	// Cancelar el pedido.
	adminAuth := adminAuthHeader(t)
	patchPath := fmt.Sprintf("/api/admin/pedidos/%d/estado", det.ID)
	patchBody := `{"estado":"cancelado"}`
	status, _ = doJSON(t, app, http.MethodPatch, patchPath, patchBody, adminAuth)
	if status != http.StatusNoContent {
		t.Fatalf("PATCH cancelar: expected 204, got %d", status)
	}

	// Stock debe haber vuelto a 50.
	var stockAfterCancel int
	if err := db.QueryRow(`SELECT stock FROM productos WHERE id = 1`).Scan(&stockAfterCancel); err != nil {
		t.Fatalf("leyendo stock post-cancel: %v", err)
	}
	if stockAfterCancel != 50 {
		t.Errorf("stock post-cancel esperado=50, got=%d", stockAfterCancel)
	}

	// Verificar estado del pedido en DB.
	var estado string
	if err := db.QueryRow(`SELECT estado FROM pedidos WHERE id = $1`, det.ID).Scan(&estado); err != nil {
		t.Fatalf("leyendo estado: %v", err)
	}
	if estado != "cancelado" {
		t.Errorf("estado pedido esperado=cancelado, got=%q", estado)
	}
}

func TestCancelarRevierteComisiones(t *testing.T) {
	app, db := setupTestDB(t)
	defer db.Close()
	truncateTestDB(t, db)
	resetProductos(t, db)
	seedProductos(t, db)

	// linkUpline=true crea el nodo de referido hijo de TU (id=1) para
	// que ActualizarComisiones registre al menos 1 comisión con
	// pedido_id enlazado.
	uid := seedUsuario(t, db, "clienteCom@upse.edu.ec", "cliente", true)
	authC := authHeaderFor(t, uid, "clienteCom@upse.edu.ec", "cliente")

	// Checkout para generar comisiones con pedido_id enlazado.
	checkoutBody := `{
		"nombre":"Cliente Com",
		"email":"clienteCom@upse.edu.ec",
		"direccion":"Calle 2",
		"ciudad":"Loja",
		"telefono":"0999999999",
		"metodo_pago":"tarjeta",
		"items":[{"id":1,"nombre":"Serum Revitalizante","precio":45.0,"cantidad":1}]
	}`
	status, body := doJSON(t, app, http.MethodPost, "/api/checkout", checkoutBody, authC)
	if status != http.StatusCreated {
		t.Fatalf("checkout: expected 201, got %d body=%s", status, body)
	}
	var det models.PedidoDetalle
	_ = json.Unmarshal(body, &det)

	// Esperar a que la goroutine de comisiones termine (sleep 200ms en el
	// controller + latencia). Sin esto, podemos leer antes de que escriba.
	time.Sleep(600 * time.Millisecond)

	// Verificar que hay al menos una comisión con pedido_id = det.ID en
	// estado 'pendiente'.
	var countPendiente int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM comisiones
		  WHERE pedido_id = $1 AND estado = 'pendiente'`,
		det.ID,
	).Scan(&countPendiente); err != nil {
		t.Fatalf("leyendo comisiones: %v", err)
	}
	if countPendiente == 0 {
		t.Fatalf("esperaba >=1 comision pendiente con pedido_id=%d, got 0", det.ID)
	}

	// Cancelar el pedido.
	adminAuth := adminAuthHeader(t)
	patchPath := fmt.Sprintf("/api/admin/pedidos/%d/estado", det.ID)
	patchBody := `{"estado":"cancelado"}`
	status, _ = doJSON(t, app, http.MethodPatch, patchPath, patchBody, adminAuth)
	if status != http.StatusNoContent {
		t.Fatalf("PATCH cancelar: expected 204, got %d", status)
	}

	// Las comisiones pendientes de ese pedido deben estar ahora 'revertida'.
	var countRevertida int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM comisiones
		  WHERE pedido_id = $1 AND estado = 'revertida'`,
		det.ID,
	).Scan(&countRevertida); err != nil {
		t.Fatalf("leyendo comisiones revertidas: %v", err)
	}
	if countRevertida == 0 {
		t.Errorf("esperaba >=1 comision revertida con pedido_id=%d, got 0", det.ID)
	}

	// Y ya no debe haber pendientes del mismo pedido.
	var countPendientePost int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM comisiones
		  WHERE pedido_id = $1 AND estado = 'pendiente'`,
		det.ID,
	).Scan(&countPendientePost); err != nil {
		t.Fatalf("leyendo comisiones pendientes post-cancel: %v", err)
	}
	if countPendientePost != 0 {
		t.Errorf("esperaba 0 comisiones pendientes post-cancel, got %d", countPendientePost)
	}
}