package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/lib/pq"

	"golang.org/x/crypto/bcrypt"

	"multicatalogo-backend/domain"
	"multicatalogo-backend/internal/dto"
	"multicatalogo-backend/interfaces"
	"multicatalogo-backend/models"
)

// Repository implementa interfaces.DBRepository.
// Sprint F2: el catálogo de productos sale del slice en memoria y pasa a
// PostgreSQL (tabla `productos`). La red sigue siendo mock hasta que otro
// sprint la migre.
type Repository struct {
	db  *sql.DB
	red []models.Referido
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{
		db: db,
		red: []models.Referido{
			{ID: 1, Codigo: "MOCK01"},
			{ID: 2, Codigo: "MOCK02"},
			{ID: 3, Codigo: "MOCK03"},
		},
	}
}

// ─────────────────────────── Auth (F1) ───────────────────────────

// Login busca el usuario por email, valida la contraseña con bcrypt y
// respeta los flags de cuenta (activo). Los códigos de error son
// sentinelas de models (ErrCredencialesInvalidas / ErrCuentaDesactivada)
// para que el controller los mapee a 401/403 sin enumerar.
func (r *Repository) Login(email, password string) (*models.Usuario, error) {
	if r.db == nil {
		return nil, errors.New("base de datos no disponible")
	}

	var u models.Usuario
	var passwordHash sql.NullString
	err := r.db.QueryRow(
		`SELECT id, email, password_hash, rol, COALESCE(nombre,''), activo, referido_id
		   FROM usuarios WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.Email, &passwordHash, &u.Rol, &u.Nombre, &u.Activo, &u.ReferidoID)
	if errors.Is(err, sql.ErrNoRows) {
		// No enumerar: misma respuesta que contraseña mala (401).
		return nil, models.ErrCredencialesInvalidas
	}
	if err != nil {
		return nil, fmt.Errorf("error consultando usuario: %w", err)
	}

	if !u.Activo {
		return nil, models.ErrCuentaDesactivada
	}

	if !passwordHash.Valid ||
		bcrypt.CompareHashAndPassword([]byte(passwordHash.String), []byte(password)) != nil {
		return nil, models.ErrCredencialesInvalidas
	}

	return &u, nil
}

// Registrar crea el usuario, le asigna un código de referido único y,
// opcionalmente, lo enlaza al árbol bajo el referente de `req.Ref`.
// Todo en una transacción (BEGIN/COMMIT/ROLLBACK) para que no queden
// usuarios sin código de referido o sin parent consistente.
//
// Devuelve ErrEmailDuplicado si el email ya existe, ErrCodigoRefInvalido
// si `req.Ref` no corresponde a ningún nodo del árbol.
func (r *Repository) Registrar(req models.RegisterRequest) (*models.RegisterResult, error) {
	if r.db == nil {
		return nil, errors.New("base de datos no disponible")
	}

	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("error iniciando transacción: %w", err)
	}
	defer tx.Rollback()

	// 1) INSERT usuario — bcrypt cost 12 (alineado con el seed del 005).
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		return nil, fmt.Errorf("error hasheando contraseña: %w", err)
	}

	var uid int
	err = tx.QueryRow(
		`INSERT INTO usuarios (email, password_hash, rol, nombre, activo)
		 VALUES ($1, $2, 'cliente', $3, true)
		 RETURNING id`,
		req.Email, string(hash), req.Nombre,
	).Scan(&uid)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, models.ErrEmailDuplicado
		}
		return nil, fmt.Errorf("error insertando usuario: %w", err)
	}

	// 2) Generar código de referido único. La probabilidad de colisión es
	// ~10^-7 con 32^6 códigos, pero igualmente reintentamos si el INSERT
	// falla por unique_violation.
	var codigo string
	const maxIntentos = 5
	for intento := 0; intento < maxIntentos; intento++ {
		codigo, err = domain.GenerarCodigoRef()
		if err != nil {
			return nil, fmt.Errorf("error generando código de referido: %w", err)
		}

		var rid int
		err = tx.QueryRow(
			`INSERT INTO referidos (codigo, parent_id, usuario_id)
			 VALUES ($1, NULL, $2)
			 RETURNING id`,
			codigo, uid,
		).Scan(&rid)
		if err == nil {
			break
		}
		if !isUniqueViolation(err) {
			return nil, fmt.Errorf("error creando nodo de referido: %w", err)
		}
		// colisión de código — generar otro y reintentar
		err = nil
	}
	if err != nil {
		return nil, fmt.Errorf("no se pudo generar un código único tras %d intentos: %w", maxIntentos, err)
	}

	// 3) Vincular el back-pointer usuarios.referido_id SIEMPRE al nodo recién
	// creado. Sin esto, ActualizarComisiones no podría caminar el árbol del
	// MLM porque parte de `SELECT referido_id FROM usuarios WHERE email = $1`.
	var ridNuevo int
	if err = tx.QueryRow(`SELECT id FROM referidos WHERE usuario_id = $1`, uid).Scan(&ridNuevo); err != nil {
		return nil, fmt.Errorf("error localizando nodo recién creado: %w", err)
	}
	if _, err = tx.Exec(
		`UPDATE usuarios SET referido_id = $1 WHERE id = $2`,
		ridNuevo, uid,
	); err != nil {
		return nil, fmt.Errorf("error vinculando usuario a su nodo: %w", err)
	}

	// 4) Si trae código de referente válido, enlazar al árbol. La columna
	// `usuarios.referido_id` apunta al id del nodo (referidos.id) del
	// referente. El nodo nuevo recibe parent_id = id_del_referente.
	if req.Ref != "" {
		var parentID int
		err = tx.QueryRow(
			`SELECT id FROM referidos WHERE codigo = $1`,
			strings.ToUpper(req.Ref),
		).Scan(&parentID)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, models.ErrCodigoRefInvalido
		}
		if err != nil {
			return nil, fmt.Errorf("error buscando referente: %w", err)
		}

		if _, err = tx.Exec(
			`UPDATE referidos SET parent_id = $1 WHERE id = $2`,
			parentID, ridNuevo,
		); err != nil {
			return nil, fmt.Errorf("error vinculando nodo al parent: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("error en commit: %w", err)
	}

	return &models.RegisterResult{
		UsuarioID: uid,
		Email:     req.Email,
		CodigoRef: codigo,
	}, nil
}

// GetUsuarioByCodigoRef busca un nodo por su código. Devuelve
// ErrCodigoRefInvalido si no existe (404 implícito en el controller).
func (r *Repository) GetUsuarioByCodigoRef(codigo string) (*models.Referido, error) {
	if r.db == nil {
		return nil, errors.New("base de datos no disponible")
	}

	var ref models.Referido
	err := r.db.QueryRow(
		`SELECT id, codigo, parent_id, usuario_id
		   FROM referidos WHERE codigo = $1`,
		strings.ToUpper(codigo),
	).Scan(&ref.ID, &ref.Codigo, &ref.ParentID, &ref.UsuarioID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, models.ErrCodigoRefInvalido
	}
	if err != nil {
		return nil, fmt.Errorf("error consultando código de referido: %w", err)
	}
	return &ref, nil
}

// ─────────────────────────── Catálogo / Red ───────────────────────────
//
// La red no se ha migrado todavía (queda como slice en memoria para no
// romper F3). El catálogo ya vive en PostgreSQL.

// GetAllProductos devuelve todos los productos activo=true ordenados
// por nombre. Es el método que usa el catálogo público; el admin usa
// GetProductosAdmin (que no filtra por activo y soporta paginación).
func (r *Repository) GetAllProductos() ([]models.Producto, error) {
	if r.db == nil {
		return nil, errors.New("base de datos no disponible")
	}
	rows, err := r.db.Query(
		`SELECT id, nombre, precio, categoria, stock, img, activo
		   FROM productos
		  WHERE activo = true
		  ORDER BY nombre`,
	)
	if err != nil {
		return nil, fmt.Errorf("error consultando productos: %w", err)
	}
	defer rows.Close()

	out := []models.Producto{}
	for rows.Next() {
		var p models.Producto
		if err := rows.Scan(&p.ID, &p.Nombre, &p.Precio, &p.Categoria, &p.Stock, &p.Img, &p.Activo); err != nil {
			return nil, fmt.Errorf("error leyendo fila de producto: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// GetProductoByID devuelve el producto sea activo o no (404 si no
// existe la fila). El controller público ya filtra activo si quiere,
// pero exponerlo permite que admin consulte inactivos.
func (r *Repository) GetProductoByID(id int) (*models.Producto, error) {
	if r.db == nil {
		return nil, errors.New("base de datos no disponible")
	}
	var p models.Producto
	err := r.db.QueryRow(
		`SELECT id, nombre, precio, categoria, stock, img, activo
		   FROM productos WHERE id = $1`,
		id,
	).Scan(&p.ID, &p.Nombre, &p.Precio, &p.Categoria, &p.Stock, &p.Img, &p.Activo)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, models.ErrProductoNoEncontrado
	}
	if err != nil {
		return nil, fmt.Errorf("error consultando producto %d: %w", id, err)
	}
	return &p, nil
}

// GetProductosAdmin implementa GET /api/admin/productos con búsqueda
// opcional y paginación. El total se obtiene vía COUNT(*) OVER() para
// evitar una segunda consulta. Normaliza page/limit a defaults seguros
// (page=1, limit=20, limit cap=100) — el controller ya valida pero lo
// hacemos aquí también como defensa en profundidad.
//
// Query-params soportados:
//
//	q       — substring a buscar (ILIKE sobre nombre y categoria)
//	activo  — true/false (nil = todos)
//	page    — 1-based; default 1
//	limit   — 1..100; default 20
func (r *Repository) GetProductosAdmin(f interfaces.ProductoFiltro) ([]models.Producto, int, error) {
	if r.db == nil {
		return nil, 0, errors.New("base de datos no disponible")
	}

	if f.Page <= 0 {
		f.Page = 1
	}
	if f.Limit <= 0 {
		f.Limit = 20
	}
	if f.Limit > 100 {
		f.Limit = 100
	}

	conds := []string{}
	args := []interface{}{}
	if q := strings.TrimSpace(f.Q); q != "" {
		args = append(args, "%"+q+"%")
		conds = append(conds, fmt.Sprintf("(nombre ILIKE $%d OR categoria ILIKE $%d)", len(args), len(args)))
	}
	if f.Activo != nil {
		args = append(args, *f.Activo)
		conds = append(conds, fmt.Sprintf("activo = $%d", len(args)))
	}

	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}

	args = append(args, f.Limit, (f.Page-1)*f.Limit)
	query := fmt.Sprintf(
		`SELECT id, nombre, precio, categoria, stock, img, activo,
		        COUNT(*) OVER() AS total
		   FROM productos
		   %s
		  ORDER BY nombre
		  LIMIT $%d OFFSET $%d`,
		where, len(args)-1, len(args),
	)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("error consultando productos admin: %w", err)
	}
	defer rows.Close()

	out := []models.Producto{}
	var total int
	for rows.Next() {
		var p models.Producto
		if err := rows.Scan(&p.ID, &p.Nombre, &p.Precio, &p.Categoria, &p.Stock, &p.Img, &p.Activo, &total); err != nil {
			return nil, 0, fmt.Errorf("error leyendo fila admin: %w", err)
		}
		out = append(out, p)
	}
	return out, total, rows.Err()
}

// CreateProducto inserta un producto nuevo y devuelve la fila completa
// con id asignado y activo=true por defecto. El stock negativo o el
// precio negativo se rechazan en el CHECK de la tabla (precio/stock>=0).
func (r *Repository) CreateProducto(d dto.ProductoCreate) (*models.Producto, error) {
	if r.db == nil {
		return nil, errors.New("base de datos no disponible")
	}

	var p models.Producto
	err := r.db.QueryRow(
		`INSERT INTO productos (nombre, precio, categoria, stock, img, activo)
		 VALUES ($1, $2, $3, $4, $5, true)
		 RETURNING id, nombre, precio, categoria, stock, img, activo`,
		d.Nombre, d.Precio, d.Categoria, d.Stock, d.Img,
	).Scan(&p.ID, &p.Nombre, &p.Precio, &p.Categoria, &p.Stock, &p.Img, &p.Activo)
	if err != nil {
		return nil, fmt.Errorf("error insertando producto: %w", err)
	}
	return &p, nil
}

// UpdateProducto aplica un PUT parcial con COALESCE: sólo los campos
// enviados se actualizan. Stock y Activo son *punteros* en el DTO, lo
// que permite distinguir "0/false explícito" de "no enviado". Para
// los strings, vacío == no enviado (la validación del DTO rechaza
// nombre de 1 char o vacío antes de llegar aquí).
func (r *Repository) UpdateProducto(id int, d dto.ProductoUpdate) (*models.Producto, error) {
	if r.db == nil {
		return nil, errors.New("base de datos no disponible")
	}

	// Construir NullTyers: nil/no-enviado -> SQL NULL -> COALESCE conserva
	// el valor actual; valor enviado -> COALESCE lo escribe.
	var (
		nombre    sql.NullString
		precio    sql.NullFloat64
		categoria sql.NullString
		stock     sql.NullInt64
		img       sql.NullString
		activo    sql.NullBool
	)
	if d.Nombre != "" {
		nombre = sql.NullString{String: d.Nombre, Valid: true}
	}
	if d.Precio != 0 {
		precio = sql.NullFloat64{Float64: d.Precio, Valid: true}
	}
	if d.Categoria != "" {
		categoria = sql.NullString{String: d.Categoria, Valid: true}
	}
	if d.Stock != nil {
		stock = sql.NullInt64{Int64: int64(*d.Stock), Valid: true}
	}
	if d.Img != "" {
		img = sql.NullString{String: d.Img, Valid: true}
	}
	if d.Activo != nil {
		activo = sql.NullBool{Bool: *d.Activo, Valid: true}
	}

	var p models.Producto
	err := r.db.QueryRow(
		`UPDATE productos SET
		    nombre    = COALESCE($1, nombre),
		    precio    = COALESCE($2, precio),
		    categoria = COALESCE($3, categoria),
		    stock     = COALESCE($4, stock),
		    img       = COALESCE($5, img),
		    activo    = COALESCE($6, activo)
		 WHERE id = $7
		 RETURNING id, nombre, precio, categoria, stock, img, activo`,
		nombre, precio, categoria, stock, img, activo, id,
	).Scan(&p.ID, &p.Nombre, &p.Precio, &p.Categoria, &p.Stock, &p.Img, &p.Activo)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, models.ErrProductoNoEncontrado
	}
	if err != nil {
		return nil, fmt.Errorf("error actualizando producto %d: %w", id, err)
	}
	return &p, nil
}

// SoftDeleteProducto marca activo=false sin eliminar la fila (preserva
// FK desde pedido_items y mantiene histórico de auditoría). Si el
// producto ya estaba inactivo, la operación es idempotente (no error).
// Si no existe, devuelve ErrProductoNoEncontrado.
func (r *Repository) SoftDeleteProducto(id int) error {
	if r.db == nil {
		return errors.New("base de datos no disponible")
	}

	res, err := r.db.Exec(
		`UPDATE productos SET activo = false
		 WHERE id = $1 AND activo = true`,
		id,
	)
	if err != nil {
		return fmt.Errorf("error desactivando producto %d: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n > 0 {
		return nil // desactivado en este momento
	}

	// 0 filas afectadas: o no existe, o ya estaba inactivo.
	var exists bool
	if err := r.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM productos WHERE id = $1)`, id).Scan(&exists); err != nil {
		return fmt.Errorf("error verificando existencia: %w", err)
	}
	if !exists {
		return models.ErrProductoNoEncontrado
	}
	// ya estaba inactivo: idempotente
	return nil
}

func (r *Repository) GetNetwork() []models.Referido {
	return r.red
}

// ─────────────────────────── Checkout (real DB desde S1) ───────────────────────────

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

// ─────────────────────────── Comisiones (F1 — upline walk real) ───────────────────────────

// ActualizarComisiones recorre el árbol de referidos hasta 3 niveles
// hacia arriba del usuario que compró y registra una comisión por cada
// nodo referente, con los porcentajes 10% / 5% / 2%.
//
// Notas sobre el modelo:
//
//   - pedido_id queda NULL por ahora: el checkout actual no propaga el id
//     de pedido al callback asíncrono. En F4 se cambiará la firma para
//     recibirlo y enlazar la comisión a su pedido origen.
//   - La operación se hace best-effort: si cualquier nivel falla (por
//     ejemplo FK constraint al cerrar la conexión), se loguea y se
//     abandona sin afectar la respuesta HTTP al cliente.
func (r *Repository) ActualizarComisiones(total float64, emailUsuario string) {
	if r.db == nil {
		log.Printf("[Goroutine] DB no disponible, comisiones no actualizadas")
		return
	}

	// 1) Obtener el nodo de referido del usuario.
	var refID sql.NullInt64
	err := r.db.QueryRow(
		`SELECT referido_id FROM usuarios WHERE email = $1`,
		emailUsuario,
	).Scan(&refID)
	if err != nil || !refID.Valid {
		log.Printf("[Comision] usuario %s sin referido, no hay upline", emailUsuario)
		return
	}

	porcentajes := []float64{0.10, 0.05, 0.02}
	current := int(refID.Int64)
	for nivel, pct := range porcentajes {
		// Subir un nivel: el "parent" del nodo actual es el nodo del referente.
		var parentID sql.NullInt64
		err := r.db.QueryRow(
			`SELECT parent_id FROM referidos WHERE id = $1`,
			current,
		).Scan(&parentID)
		if err != nil || !parentID.Valid {
			// Llegamos a la raíz o el nodo no tiene parent: fin del upline.
			return
		}
		current = int(parentID.Int64)

		comision := total * pct
		// pedido_id NULL intencionalmente en F1 (ver doc-comment arriba).
		_, err = r.db.Exec(
			`INSERT INTO comisiones
			   (nivel, monto, origen_email, pedido_id, referido_id, porcentaje, estado)
			 VALUES ($1, $2, $3, NULL, $4, $5, 'pendiente')`,
			nivel+1, comision, emailUsuario, current, pct*100,
		)
		if err != nil {
			log.Printf("[Comision] error nivel %d: %v", nivel+1, err)
			return
		}
	}
	log.Printf("[Comision] upline walk completado para %s (total=%.2f)", emailUsuario, total)
}

// isUniqueViolation devuelve true si err es una violación de unicidad de
// Postgres (SQLSTATE 23505). Se usa para distinguir email duplicado o
// colisión de código de los demás errores de INSERT.
func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "23505"
	}
	return false
}