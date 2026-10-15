package interfaces

import (
	"context"
	"time"
	"multicatalogo-backend/models"
)

type DBRepository interface {
	Authenticate(email, password string) (bool, string)
	GetAllProducts() []models.Producto
	GetProductByID(id int) (*models.Producto, error)
	GetNetwork() []models.Referido
	ProcessCheckout(req models.CheckoutRequest) (*models.Pedido, error)
	ActualizarComisiones(total float64, emailUsuario string)
}

type CacheService interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}
