package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"multicatalogo-backend/models"
)

// Repository implementa interfaces.DBRepository.
// Usa datos mock en memoria para productos/red/auth.
// Usa PostgreSQL para transacciones de checkout.
type Repository struct {
	db        *sql.DB
	productos []models.Producto
	red       []models.Referido
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{
		db: db,
		productos: []models.Producto{
			{ID: 1, Nombre: "Serum Revitalizante", Precio: 45.00, Img: "https://picsum.photos/seed/serum/150"},
			{ID: 2, Nombre: "Crema Hidratante Pro", Precio: 32.50, Img: "https://picsum.photos/seed/crema/150"},
			{ID: 3, Nombre: "Tónico Purificante", Precio: 28.00, Img: "https://picsum.photos/seed/tonico/150"},
			{ID: 4, Nombre: "Mascarilla Nocturna", Precio: 50.00, Img: "https://picsum.photos/seed/mascarilla/150"},
		},
		red: []models.Referido{
			{ID: 1, Nombre: "Ana García", Nivel: "Nivel 1", Ventas: "$1,200"},
			{ID: 2, Nombre: "Luis Poveda", Nivel: "Nivel 1", Ventas: "$850"},
			{ID: 3, Nombre: "Marta Sánchez", Nivel: "Nivel 2", Ventas: "$430"},
		},
	}
}

func (r *Repository) Authenticate(email, password string) (bool, string) {
	if email == "admin@upse.edu.ec" && password == "123456" {
		return true, "fake-jwt-token-123"
	}
	return false, ""
}

func (r *Repository) GetAllProducts() []models.Producto {
	return r.productos
}

func (r *Repository) GetProductByID(id int) (*models.Producto, error) {
	for _, p := range r.productos {
		if p.ID == id {
			return &p, nil
		}
	}
	return nil, errors.New("producto no encontrado")
}

func (r *Repository) GetNetwork() []models.Referido {
	return r.red
}

// ProcessCheckout ejecuta una transacción ACID: inserta pedido + items.
// Si cualquier paso falla, el defer tx.Rollback() revierte todo (ROLLBACK).
// Solo si todo es exitoso, tx.Commit() persiste los datos (COMMIT).
func (r *Repository) ProcessCheckout(req models.CheckoutRequest) (*models.Pedido, error) {
	if r.db == nil {
		return nil, errors.New("base de datos no disponible")
	}

	// Calcular total
	var total float64
	for _, item := range req.Items {
		total += item.Precio * float64(item.Cantidad)
	}

	// BEGIN — iniciar transacción
	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("error iniciando transacción: %w", err)
	}
	// defer garantiza ROLLBACK si no se llega al COMMIT
	defer tx.Rollback()

	// Paso 1: INSERT en tabla pedidos
	var pedidoID int
	err = tx.QueryRow(
		"INSERT INTO pedidos (email_usuario, total, metodo_pago) VALUES ($1, $2, $3) RETURNING id",
		req.Email, total, req.MetodoPago,
	).Scan(&pedidoID)
	if err != nil {
		return nil, fmt.Errorf("error insertando pedido: %w", err)
	}

	// Paso 2: INSERT en tabla pedido_items por cada producto del carrito
	for _, item := range req.Items {
		_, err = tx.Exec(
			"INSERT INTO pedido_items (pedido_id, producto_id, nombre, precio, cantidad) VALUES ($1, $2, $3, $4, $5)",
			pedidoID, item.ID, item.Nombre, item.Precio, item.Cantidad,
		)
		if err != nil {
			return nil, fmt.Errorf("error insertando item %d: %w", item.ID, err)
		}
	}

	// COMMIT — persistir todo si no hubo errores
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("error en commit: %w", err)
	}

	log.Printf("[DB] Pedido #%d creado correctamente (total: $%.2f)", pedidoID, total)
	return &models.Pedido{ID: pedidoID, Total: total, Estado: "confirmado"}, nil
}

// ActualizarComisiones simula la actualización de comisiones en la red multinivel.
// Es llamado desde una Goroutine (asíncrono) — no bloquea la respuesta HTTP.
func (r *Repository) ActualizarComisiones(total float64, emailUsuario string) {
	if r.db == nil {
		log.Printf("[Goroutine] DB no disponible, comisiones no actualizadas")
		return
	}
	// Distribuye: Nivel 1 = 10%, Nivel 2 = 5%, Nivel 3 = 2%
	porcentajes := []float64{0.10, 0.05, 0.02}
	for nivel, pct := range porcentajes {
		comision := total * pct
		_, err := r.db.Exec(
			"INSERT INTO comisiones (nivel, monto, origen_email) VALUES ($1, $2, $3)",
			nivel+1, comision, emailUsuario,
		)
		if err != nil {
			log.Printf("[Goroutine] Error guardando comisión nivel %d: %v", nivel+1, err)
			return
		}
	}
	log.Printf("[Goroutine] Comisiones de la red actualizadas para pedido de %s (total=%.2f)", emailUsuario, total)
}
