package controllers_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
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

func TestProductos(t *testing.T) {
	app := setupTestApp()

	t.Run("Get all products", func(t *testing.T) {
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
		if ae.Status != 400 || ae.Message != "El ID del producto debe ser un número entero" {
			t.Errorf("unexpected error: %+v", ae)
		}
	})

	t.Run("Get product by ID - not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/productos/999", nil)
		resp, _ := app.Test(req)
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("expected 404, got %d", resp.StatusCode)
		}
		var ae apiErr
		body, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(body, &ae)
		if ae.Status != 404 || ae.Message != "Producto no encontrado" {
			t.Errorf("unexpected error: %+v", ae)
		}
	})

	t.Run("Get product by ID - found", func(t *testing.T) {
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

	t.Run("Invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/checkout", strings.NewReader("invalid"))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := app.Test(req)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
	})

	t.Run("Missing email", func(t *testing.T) {
		body := `{"nombre":"Test","email":"","items":[{"id":1,"nombre":"Serum","precio":45,"cantidad":1}]}`
		req := httptest.NewRequest(http.MethodPost, "/api/checkout", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := app.Test(req)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
	})

	t.Run("No DB returns 500", func(t *testing.T) {
		body := `{"nombre":"Test","email":"test@email.com","metodo_pago":"tarjeta","items":[{"id":1,"nombre":"Serum","precio":45,"cantidad":1}]}`
		req := httptest.NewRequest(http.MethodPost, "/api/checkout", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := app.Test(req)
		if resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("expected 500 (no DB), got %d", resp.StatusCode)
		}
	})
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