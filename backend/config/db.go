package config

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func InitDB() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://admin:secret@localhost:5433/multicatalogo?sslmode=disable"
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Printf("[DB] Error abriendo conexión: %v", err)
		return
	}
	if err := db.Ping(); err != nil {
		log.Printf("[DB] No se pudo conectar a PostgreSQL: %v", err)
		return
	}
	DB = db
	log.Println("[DB] Conexión a PostgreSQL establecida")
}
