package repository

import (
	"errors"
	"multicatalogo-backend/models"
)

// Datos mockeados en memoria
var productos = []models.Producto{
	{ID: 1, Nombre: "Serum Revitalizante", Precio: 45.00, Img: "https://picsum.photos/seed/serum/150"},
	{ID: 2, Nombre: "Crema Hidratante Pro", Precio: 32.50, Img: "https://picsum.photos/seed/crema/150"},
	{ID: 3, Nombre: "Tónico Purificante", Precio: 28.00, Img: "https://picsum.photos/seed/tonico/150"},
	{ID: 4, Nombre: "Mascarilla Nocturna", Precio: 50.00, Img: "https://picsum.photos/seed/mascarilla/150"},
}

var red = []models.Referido{
	{ID: 1, Nombre: "Ana García", Nivel: "Nivel 1", Ventas: "$1,200"},
	{ID: 2, Nombre: "Luis Poveda", Nivel: "Nivel 1", Ventas: "$850"},
	{ID: 3, Nombre: "Marta Sánchez", Nivel: "Nivel 2", Ventas: "$430"},
}

// Authenticate valida las credenciales y retorna si es válido y el token generado.
func Authenticate(email, password string) (bool, string) {
	if email == "admin@upse.edu.ec" && password == "123456" {
		return true, "fake-jwt-token-123"
	}
	return false, ""
}

// GetAllProducts retorna la lista completa de productos.
func GetAllProducts() []models.Producto {
	return productos
}

// GetProductByID busca y retorna un producto por su ID o un error si no existe.
func GetProductByID(id int) (*models.Producto, error) {
	for _, p := range productos {
		if p.ID == id {
			return &p, nil
		}
	}
	return nil, errors.New("producto no encontrado")
}

// GetNetwork retorna la lista de referidos en la red.
func GetNetwork() []models.Referido {
	return red
}
