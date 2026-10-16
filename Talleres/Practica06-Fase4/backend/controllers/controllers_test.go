package controllers_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"multicatalogo-backend/controllers"
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
	routes.SetupRoutes(app, ctrl)
	return app
}

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

func TestLogin(t *testing.T) {
	app := setupTestApp()

	t.Run("Invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader("invalid-json"))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := app.Test(req)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
		var apiErr models.APIError
		body, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(body, &apiErr)
		if apiErr.Status != 400 || apiErr.Message != "Cuerpo de petición inválido" {
			t.Errorf("unexpected error: %+v", apiErr)
		}
	})

	t.Run("Empty fields", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(`{"email":"","password":""}`))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := app.Test(req)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
		var apiErr models.APIError
		body, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(body, &apiErr)
		if apiErr.Status != 400 || apiErr.Message != "El correo y la contraseña son obligatorios" {
			t.Errorf("unexpected error: %+v", apiErr)
		}
	})

	t.Run("Invalid credentials", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(`{"email":"wrong@upse.edu.ec","password":"wrong"}`))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := app.Test(req)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", resp.StatusCode)
		}
		var apiErr models.APIError
		body, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(body, &apiErr)
		if apiErr.Status != 401 || apiErr.Message != "Credenciales incorrectas" {
			t.Errorf("unexpected error: %+v", apiErr)
		}
	})

	t.Run("Valid credentials", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(`{"email":"admin@upse.edu.ec","password":"123456"}`))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := app.Test(req)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
		var res map[string]string
		body, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(body, &res)
		if res["token"] != "fake-jwt-token-123" || res["email"] != "admin@upse.edu.ec" {
			t.Errorf("unexpected login result: %v", res)
		}
	})
}

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
		var apiErr models.APIError
		body, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(body, &apiErr)
		if apiErr.Status != 400 || apiErr.Message != "El ID del producto debe ser un número entero" {
			t.Errorf("unexpected error: %+v", apiErr)
		}
	})

	t.Run("Get product by ID - not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/productos/999", nil)
		resp, _ := app.Test(req)
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("expected 404, got %d", resp.StatusCode)
		}
		var apiErr models.APIError
		body, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(body, &apiErr)
		if apiErr.Status != 404 || apiErr.Message != "Producto no encontrado" {
			t.Errorf("unexpected error: %+v", apiErr)
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
