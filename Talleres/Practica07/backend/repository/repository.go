package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"

	"golang.org/x/crypto/bcrypt"

	"multicatalogo-backend/domain"
	"multicatalogo-backend/interfaces"
	"multicatalogo-backend/internal/dto"
	"multicatalogo-backend/models"
)

// Repository implementa interfaces.DBRepository.
//
// La red multinivel ya no vive en memoria: tanto la vista global como el
// subárbol personal se reconstruyen desde la tabla `referidos` con una CTE
// recursiva. El slice `red` que quedaba como mock fue eliminado en Sprint 1.
type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
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

// GetReferidoIDByUsuario devuelve el nodo de red del usuario, o nil si
// todavía no tiene (se registró sin código de referido).
func (r *Repository) GetReferidoIDByUsuario(usuarioID int) (*int, error) {
	if r.db == nil {
		return nil, errors.New("base de datos no disponible")
	}
	var refID sql.NullInt64
	err := r.db.QueryRow(
		`SELECT referido_id FROM usuarios WHERE id = $1`, usuarioID,
	).Scan(&refID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, models.ErrUsuarioNoEncontrado
	}
	if err != nil {
		return nil, fmt.Errorf("error consultando el referido del usuario %d: %w", usuarioID, err)
	}
	if !refID.Valid {
		return nil, nil
	}
	v := int(refID.Int64)
	return &v, nil
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

// ─────────────────────────── Red multinivel (Sprint 1) ───────────────────────
//
// Ambas vistas devuelven la lista PLANA de nodos; el ensamblado a árbol lo
// hace domain.ConstruirArbol (puro, testeable sin I/O).

// maxNivelesRed acota la profundidad de la CTE. El schema no impide que
// parent_id forme un lazo; sin este tope, un ciclo convertiría la consulta
// recursiva en un bucle infinito (UNION ALL no deduplica porque la columna
// `nivel` cambia en cada vuelta).
const maxNivelesRed = 20

// columnasRed es la proyección compartida por las dos consultas.
const columnasRed = `COALESCE(u.nombre, r.codigo) AS nombre`

// GetRedGlobal devuelve TODOS los nodos del árbol. Es la vista de
// administración (GET /api/red): la usa el admin para ver la red completa.
// El nivel de cada nodo lo recalcula domain.ConstruirArbol, no la consulta.
func (r *Repository) GetRedGlobal() ([]domain.NodoPlano, error) {
	if r.db == nil {
		return nil, errors.New("base de datos no disponible")
	}
	return r.consultarRed(
		`SELECT r.id, r.codigo, r.parent_id, r.usuario_id, r.ventas, ` + columnasRed + `
		   FROM referidos r
		   LEFT JOIN usuarios u ON u.id = r.usuario_id
		  ORDER BY r.id`,
	)
}

// GetRedPropia devuelve el subárbol que cuelga del nodo del usuario indicado
// (el `referido_id` de `usuarios`). Es lo que responde GET /api/red/mia: cada
// usuario ve SU red, no la del seed completo.
//
// El ancla es referidos.usuario_id = $1, de modo que el resultado empieza en el
// propio nodo y `nivel` es la distancia relativa al usuario.
//
// Si el usuario todavía no tiene nodo (referido_id NULL) devuelve un slice
// vacío y error nil: no es un fallo, es "todavía no tengo red".
// El ancla es `usuarios.referido_id` (la FK declarada en 002_backoffice), no
// la columna inversa referidos.usuario_id: el vínculo autoritativo va de la
// persona a su nodo, y anclar por el otro lado exigiría que ese campo
// estuviera retroalimentado en todas las filas.
func (r *Repository) GetRedPropia(usuarioID int) ([]domain.NodoPlano, error) {
	if r.db == nil {
		return nil, errors.New("base de datos no disponible")
	}
	return r.consultarRed(
		`WITH RECURSIVE arbol AS (
		     SELECT r.id, r.codigo, r.parent_id, r.usuario_id, r.ventas,
		            COALESCE(u.nombre, r.codigo) AS nombre,
		            0 AS nivel
		       FROM referidos r
		       LEFT JOIN usuarios u ON u.id = r.usuario_id
		      WHERE r.id = (SELECT referido_id FROM usuarios WHERE id = $1)
		     UNION ALL
		     SELECT r.id, r.codigo, r.parent_id, r.usuario_id, r.ventas,
		            COALESCE(u.nombre, r.codigo) AS nombre,
		            a.nivel + 1
		       FROM referidos r
		       LEFT JOIN usuarios u ON u.id = r.usuario_id
		       JOIN arbol a ON r.parent_id = a.id
		      WHERE a.nivel < $2
		 )
		 SELECT id, codigo, parent_id, usuario_id, ventas, nombre
		   FROM arbol
		  ORDER BY nivel, id`,
		usuarioID, maxNivelesRed,
	)
}

// consultarRed ejecuta la consulta y proyecta las filas a domain.NodoPlano.
// `ventas` es NUMERIC en Postgres: se lee como string y se convierte para no
// arrastrar el redondeo de NUMERIC a float en la capa de scan.
func (r *Repository) consultarRed(query string, args ...interface{}) ([]domain.NodoPlano, error) {
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("error consultando la red: %w", err)
	}
	defer rows.Close()

	nodos := make([]domain.NodoPlano, 0)
	for rows.Next() {
		var (
			n      domain.NodoPlano
			ventas string
		)
		if err := rows.Scan(&n.ID, &n.Codigo, &n.ParentID, &n.UsuarioID, &ventas, &n.Nombre); err != nil {
			return nil, fmt.Errorf("error leyendo nodo de la red: %w", err)
		}
		v, convErr := strconv.ParseFloat(ventas, 64)
		if convErr != nil {
			return nil, fmt.Errorf("ventas inválidas en el nodo %d: %w", n.ID, convErr)
		}
		n.Ventas = v
		nodos = append(nodos, n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error recorriendo la red: %w", err)
	}
	return nodos, nil
}

// ─────────────────────────── Pedidos (F3) ───────────────────────────

// CreatePedidoConUsuario ejecuta una transacción ACID que:
//  0. Bloquea cada producto con SELECT ... FOR UPDATE y lee su PRECIO REAL.
//  1. INSERT en `pedidos` con email_usuario + usuario_id + total +
//     metodo_pago + los datos de envío. El estado toma el default del schema
//     ('pendiente').
//  2. INSERT en `pedido_items` por cada producto del carrito, copiando el
//     nombre y el precio de la fila bloqueada.
//  3. UPDATE de stock condicionado a stock >= cantidad (si no hay stock
//     suficiente devuelve ErrProductoNoEncontrado — ROLLBACK).
//  4. INSERT del ledger de comisiones en la MISMA transacción.
//  5. Relee la cabecera + items y devuelve *PedidoDetalle para que el
//     controller lo retorne al cliente.
//
// usuarioID y req.EmailUsuario vienen del JWT (sub), nunca del body.
//
// Sobre el total: se calcula con los precios LEÍDOS DE LA BASE en el paso 0,
// no con los del body. El precio que manda el cliente es un dato no confiable:
// sin este paso, enviar `{"precio": 0.01}` en una línea del carrito compraría
// el producto por un céntimo y el ledger repartiría comisiones sobre esa base.
// El paso 0 toma además el lock de las filas, así que el descuento de stock
// del paso 3 no puede competir con otra compra concurrente.
func (r *Repository) CreatePedidoConUsuario(req dto.CheckoutRequest, usuarioID int) (*models.PedidoDetalle, error) {
	if r.db == nil {
		return nil, errors.New("base de datos no disponible")
	}
	if len(req.Items) == 0 {
		return nil, errors.New("carrito vacío")
	}

	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("error iniciando transacción: %w", err)
	}
	defer tx.Rollback()

	// Paso 0: precio real + lock de cada producto del carrito.
	//
	// El lock va aquí y no en el UPDATE del paso 3 porque hace falta el precio
	// ANTES de insertar la cabecera (para el total), y el orden de los pasos
	// tiene que ser el mismo dentro y fuera de la función.
	precios := make(map[int]float64, len(req.Items))
	nombres := make(map[int]string, len(req.Items))
	var total float64
	for _, item := range req.Items {
		var precio float64
		var nombre string
		err := tx.QueryRow(
			`SELECT nombre, precio FROM productos
			  WHERE id = $1 AND activo = true
			  FOR UPDATE`,
			item.ID,
		).Scan(&nombre, &precio)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, fmt.Errorf("%w: producto %d inexistente o inactivo",
					models.ErrProductoNoEncontrado, item.ID)
			}
			return nil, fmt.Errorf("error bloqueando producto %d: %w", item.ID, err)
		}
		precios[item.ID] = precio
		nombres[item.ID] = nombre
		total += precio * float64(item.Cantidad)
	}

	// Paso 1: INSERT cabecera (identidad + envío). El estado lo pone el
	// default 'pendiente' del schema.
	var pedidoID int
	err = tx.QueryRow(
		`INSERT INTO pedidos
		    (email_usuario, usuario_id, total, metodo_pago, direccion, ciudad, telefono)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id`,
		req.EmailUsuario, usuarioID, total, req.MetodoPago,
		req.Direccion, req.Ciudad, req.Telefono,
	).Scan(&pedidoID)
	if err != nil {
		return nil, fmt.Errorf("error insertando pedido: %w", err)
	}

	// Paso 2: INSERT items. Nombre y precio son la foto histórica del
	// catálogo en el momento de la compra, tomados de la fila bloqueada.
	for _, item := range req.Items {
		_, err = tx.Exec(
			`INSERT INTO pedido_items (pedido_id, producto_id, nombre, precio, cantidad)
			 VALUES ($1, $2, $3, $4, $5)`,
			pedidoID, item.ID, nombres[item.ID], precios[item.ID], item.Cantidad,
		)
		if err != nil {
			return nil, fmt.Errorf("error insertando item %d: %w", item.ID, err)
		}
	}

	// Paso 3: descontar stock. Las filas ya están bloqueadas por el paso 0,
	// así que la condición stock >= cantidad no puede perder la carrera contra
	// otra compra. Si no cumple, la actualización afecta 0 filas y se aborta
	// con ErrProductoNoEncontrado (el cliente ve un 409 STOCK_INSUFICIENTE).
	for _, item := range req.Items {
		res, err := tx.Exec(
			`UPDATE productos
			    SET stock = stock - $1
			  WHERE id = $2
			    AND stock >= $1
			    AND activo = true`,
			item.Cantidad, item.ID,
		)
		if err != nil {
			return nil, fmt.Errorf("error descontando stock producto %d: %w", item.ID, err)
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			// No distinguimos entre "no existe" e "insuficiente" desde
			// el controller — el cliente ve el mismo 409 STOCK_INSUFICIENTE.
			return nil, fmt.Errorf("%w: producto %d sin stock suficiente",
				models.ErrProductoNoEncontrado, item.ID)
		}
	}

	// Paso 4: ledger de comisiones DENTRO de la misma transacción.
	//
	// Es el punto que cambió en Sprint 2. Antes la goroutine post-COMMIT
	// insertaba en `comisiones`: si ese INSERT fallaba, el pedido quedaba
	// vendido y la comisión sin existir, sin ninguna forma de saberlo. Ahora
	// el ledger es parte de la unidad transaccional — o entra con el pedido,
	// o no entra ninguno de los dos.
	//
	// Lo que sí queda fuera es el contador `referidos.ventas`, que es una
	// lectura rápida de conveniencia (ver ActualizarVentasUpline).
	if err := insertarComisionesTx(tx, pedidoID, usuarioID, req.EmailUsuario, total); err != nil {
		return nil, err
	}

	// Paso 5: releer detalle (cabecera + items) para devolverlo al FE.
	detalle, err := selectPedidoDetalleTx(tx, pedidoID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("error en commit: %w", err)
	}
	log.Printf("[DB] Pedido #%d creado (usuario_id=%d, total=%.2f)", pedidoID, usuarioID, total)
	return detalle, nil
}

// selectPedidoDetalleTx lee cabecera + items dentro de la misma tx.
// Se usa por CreatePedidoConUsuario (tx ya abierta) y por
// CambiarEstadoPedido para devolver el detalle final.
func selectPedidoDetalleTx(tx *sql.Tx, pedidoID int) (*models.PedidoDetalle, error) {
	var d models.PedidoDetalle
	var usuarioID sql.NullInt64
	var createdAt time.Time
	err := tx.QueryRow(
		`SELECT id, usuario_id, email_usuario, total, metodo_pago, estado, created_at
		   FROM pedidos WHERE id = $1`,
		pedidoID,
	).Scan(&d.ID, &usuarioID, &d.EmailUsuario, &d.Total, &d.MetodoPago, &d.Estado, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, models.ErrPedidoNoEncontrado
	}
	if err != nil {
		return nil, fmt.Errorf("error leyendo pedido %d: %w", pedidoID, err)
	}
	if usuarioID.Valid {
		v := int(usuarioID.Int64)
		d.UsuarioID = &v
	}
	d.CreatedAt = createdAt.UTC().Format(time.RFC3339)

	rows, err := tx.Query(
		`SELECT id, producto_id, nombre, precio, cantidad
		   FROM pedido_items
		  WHERE pedido_id = $1
		  ORDER BY id`,
		pedidoID,
	)
	if err != nil {
		return nil, fmt.Errorf("error leyendo items del pedido %d: %w", pedidoID, err)
	}
	defer rows.Close()
	for rows.Next() {
		var it models.PedidoItem
		if err := rows.Scan(&it.ID, &it.ProductoID, &it.Nombre, &it.Precio, &it.Cantidad); err != nil {
			return nil, fmt.Errorf("error escaneando item: %w", err)
		}
		d.Items = append(d.Items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &d, nil
}

// normalizarPagina aplica los defaults y el tope de los listados paginados:
// página 1-based, 20 filas por defecto, 100 como máximo.
//
// Centralizarlo evita que cada endpoint interprete `?page=0` o `?limit=9999`
// a su manera — que es como un cliente termina pidiendo la tabla entera.
func normalizarPagina(page, limit int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}

// GetMisPedidos devuelve los pedidos del usuario paginados.
// Orden: más recientes primero (created_at DESC).
func (r *Repository) GetMisPedidos(usuarioID int, page, limit int) ([]models.PedidoResumen, int, error) {
	if r.db == nil {
		return nil, 0, errors.New("base de datos no disponible")
	}
	page, limit = normalizarPagina(page, limit)

	rows, err := r.db.Query(
		`SELECT id, usuario_id, total, estado, created_at,
		        COUNT(*) OVER() AS total
		   FROM pedidos
		  WHERE usuario_id = $1
		  ORDER BY created_at DESC
		  LIMIT $2 OFFSET $3`,
		usuarioID, limit, (page-1)*limit,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("error consultando pedidos del usuario %d: %w", usuarioID, err)
	}
	defer rows.Close()

	out := []models.PedidoResumen{}
	var total int
	for rows.Next() {
		var p models.PedidoResumen
		var uid sql.NullInt64
		var createdAt time.Time
		if err := rows.Scan(&p.ID, &uid, &p.Total, &p.Estado, &createdAt, &total); err != nil {
			return nil, 0, fmt.Errorf("error leyendo pedido: %w", err)
		}
		if uid.Valid {
			v := int(uid.Int64)
			p.UsuarioID = &v
		}
		p.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		out = append(out, p)
	}
	return out, total, rows.Err()
}

// GetPedidosAdmin lista pedidos con filtro opcional de estado. Si
// filtro.Estado == "" se devuelven todos (sin WHERE extra).
func (r *Repository) GetPedidosAdmin(f interfaces.PedidoFiltro) ([]models.PedidoResumen, int, error) {
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
	if e := strings.TrimSpace(f.Estado); e != "" {
		args = append(args, e)
		conds = append(conds, fmt.Sprintf("estado = $%d", len(args)))
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}

	args = append(args, f.Limit, (f.Page-1)*f.Limit)
	q := fmt.Sprintf(
		`SELECT id, usuario_id, total, estado, created_at,
		        COUNT(*) OVER() AS total
		   FROM pedidos
		   %s
		  ORDER BY created_at DESC
		  LIMIT $%d OFFSET $%d`,
		where, len(args)-1, len(args),
	)
	rows, err := r.db.Query(q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("error consultando pedidos admin: %w", err)
	}
	defer rows.Close()

	out := []models.PedidoResumen{}
	var total int
	for rows.Next() {
		var p models.PedidoResumen
		var uid sql.NullInt64
		var createdAt time.Time
		if err := rows.Scan(&p.ID, &uid, &p.Total, &p.Estado, &createdAt, &total); err != nil {
			return nil, 0, fmt.Errorf("error leyendo pedido admin: %w", err)
		}
		if uid.Valid {
			v := int(uid.Int64)
			p.UsuarioID = &v
		}
		p.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		out = append(out, p)
	}
	return out, total, rows.Err()
}

// GetPedidoDetalle devuelve el detalle de un pedido (cabecera + items).
// Devuelve ErrPedidoNoEncontrado si no existe. La autorización (dueño
// vs admin) la hace el controller — el repository es agnóstico a la
// sesión.
func (r *Repository) GetPedidoDetalle(id int) (*models.PedidoDetalle, error) {
	if r.db == nil {
		return nil, errors.New("base de datos no disponible")
	}

	var d models.PedidoDetalle
	var usuarioID sql.NullInt64
	var createdAt time.Time
	err := r.db.QueryRow(
		`SELECT id, usuario_id, email_usuario, total, metodo_pago, estado, created_at
		   FROM pedidos WHERE id = $1`,
		id,
	).Scan(&d.ID, &usuarioID, &d.EmailUsuario, &d.Total, &d.MetodoPago, &d.Estado, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, models.ErrPedidoNoEncontrado
	}
	if err != nil {
		return nil, fmt.Errorf("error consultando pedido %d: %w", id, err)
	}
	if usuarioID.Valid {
		v := int(usuarioID.Int64)
		d.UsuarioID = &v
	}
	d.CreatedAt = createdAt.UTC().Format(time.RFC3339)

	rows, err := r.db.Query(
		`SELECT id, producto_id, nombre, precio, cantidad
		   FROM pedido_items
		  WHERE pedido_id = $1
		  ORDER BY id`,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("error leyendo items del pedido %d: %w", id, err)
	}
	defer rows.Close()
	for rows.Next() {
		var it models.PedidoItem
		if err := rows.Scan(&it.ID, &it.ProductoID, &it.Nombre, &it.Precio, &it.Cantidad); err != nil {
			return nil, fmt.Errorf("error escaneando item: %w", err)
		}
		d.Items = append(d.Items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &d, nil
}

// CambiarEstadoPedido ejecuta una transacción que:
//  1. SELECT … FOR UPDATE del pedido (lock pesimista para evitar que
//     dos admins transicionen a la vez a estados incompatibles).
//  2. Valida la transición con domain.EsTransicionValida. Si es ilegal,
//     devuelve domain.ErrTransicionInvalida.
//  3. UPDATE pedidos SET estado = $nuevo.
//  4. Si nuevo == cancelado:
//     - Por cada pedido_item, devuelve stock: UPDATE productos SET
//     stock = stock + item.cantidad WHERE id = item.producto_id.
//     - UPDATE comisiones SET estado='revertida' WHERE pedido_id=$id
//     AND estado IN ('pendiente','aprobada'). Las comisiones ya
//     'pagadas' NO se tocan (decisión de negocio: una vez pagadas al
//     MLM, revertir requiere un flujo de clawback fuera de F3).
//
// Devuelve models.ErrPedidoNoEncontrado si el id no existe.
func (r *Repository) CambiarEstadoPedido(id int, nuevoEstado domain.EstadoPedido) error {
	if r.db == nil {
		return errors.New("base de datos no disponible")
	}

	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("error iniciando transacción: %w", err)
	}
	defer tx.Rollback()

	// 1) SELECT … FOR UPDATE del pedido.
	var estadoActual string
	err = tx.QueryRow(
		`SELECT estado FROM pedidos WHERE id = $1 FOR UPDATE`,
		id,
	).Scan(&estadoActual)
	if errors.Is(err, sql.ErrNoRows) {
		return models.ErrPedidoNoEncontrado
	}
	if err != nil {
		return fmt.Errorf("error leyendo pedido %d: %w", id, err)
	}

	// 2) Validar FSM.
	if !domain.EsTransicionValida(domain.EstadoPedido(estadoActual), nuevoEstado) {
		return fmt.Errorf("%w: %s → %s",
			domain.ErrTransicionInvalida, estadoActual, nuevoEstado)
	}

	// 3) UPDATE cabecera.
	if _, err := tx.Exec(
		`UPDATE pedidos SET estado = $1 WHERE id = $2`,
		string(nuevoEstado), id,
	); err != nil {
		return fmt.Errorf("error actualizando estado del pedido %d: %w", id, err)
	}

	// 4) Side-effects al cancelar.
	if nuevoEstado == domain.EstadoCancelado {
		// 4a) Devolver stock de cada item al producto.
		if _, err := tx.Exec(
			`UPDATE productos p
			    SET stock = p.stock + pi.cantidad
			   FROM pedido_items pi
			  WHERE pi.pedido_id = $1
			    AND pi.producto_id = p.id`,
			id,
		); err != nil {
			return fmt.Errorf("error devolviendo stock al cancelar pedido %d: %w", id, err)
		}
		// 4b) Revertir comisiones pendientes/aprobadas. Pagadas no se tocan:
		// el dinero ya salió y la reversión es un asunto contable aparte.
		if _, err := tx.Exec(
			`UPDATE comisiones
			    SET estado = 'revertida'
			  WHERE pedido_id = $1
			    AND estado IN ('pendiente','aprobada')`,
			id,
		); err != nil {
			return fmt.Errorf("error revirtiendo comisiones del pedido %d: %w", id, err)
		}
		// 4c) Descontar el contador `ventas` de los mismos nodos. El
		// checkout locredited dentro de su propia transacción; sin este
		// paso el árbol seguiría mostrando ventas de una compra anulada y
		// el cache quedaría mintiendo.
		if _, err := tx.Exec(
			`WITH cm AS (
			     SELECT referido_id, SUM(monto) AS total
			       FROM comisiones
			  WHERE pedido_id = $1
			  GROUP BY referido_id
			 )
			 UPDATE referidos r
			    SET ventas = r.ventas - cm.total
			   FROM cm
			  WHERE r.id = cm.referido_id`,
			id,
		); err != nil {
			return fmt.Errorf("error revirtiendo ventas del pedido %d: %w", id, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error en commit: %w", err)
	}
	log.Printf("[DB] Pedido #%d → %s", id, nuevoEstado)
	return nil
}

// ─────────────────────────── Ledger de comisiones ───────────────────────────

// cadenaUplineTx devuelve los ids de `referidos` que corresponden a los
// niveles 1..N por encima del nodo del usuario, en orden de cercanía.
//
// El ancla es el PADRE del nodo del usuario, no el nodo en sí: el nivel 1 de
// una comisión es el referente, y el comprador nunca se acredita a sí mismo.
//
// Se usa una CTE recursiva en lugar de un bucle de consultas porque devuelve
// una sola ida a la base. El tope de profundidad protege contra ciclos de
// parent_id, igual que en la construcción del árbol.
func cadenaUplineTx(tx *sql.Tx, usuarioID int, niveles int) ([]int, error) {
	rows, err := tx.Query(
		`WITH RECURSIVE upline AS (
		     SELECT r.id, r.parent_id, 1 AS nivel
		       FROM usuarios u
		       JOIN referidos propio ON propio.id = u.referido_id
		       JOIN referidos r      ON r.id = propio.parent_id
		      WHERE u.id = $1
		     UNION ALL
		     SELECT r.id, r.parent_id, up.nivel + 1
		       FROM referidos r
		       JOIN upline up ON r.id = up.parent_id
		      WHERE up.nivel < $2
		 )
		 SELECT id FROM upline ORDER BY nivel`,
		usuarioID, niveles,
	)
	if err != nil {
		return nil, fmt.Errorf("error consultando la cadena de upline: %w", err)
	}
	defer rows.Close()

	cadena := make([]int, 0, niveles)
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("error leyendo la cadena de upline: %w", err)
		}
		cadena = append(cadena, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error recorriendo la cadena de upline: %w", err)
	}
	return cadena, nil
}

// insertarComisionesTx escribe el ledger del pedido dentro de la transacción
// abierta. Si algo falla, el llamador hace ROLLBACK y el pedido tampoco existe.
func insertarComisionesTx(tx *sql.Tx, pedidoID, usuarioID int, email string, total float64) error {
	cadena, err := cadenaUplineTx(tx, usuarioID, domain.MaxNivelesComision)
	if err != nil {
		return err
	}
	// Sin upline no hay comisión: es un resultado válido (compra de raíz).
	comisiones := domain.RepartirComisiones(total, cadena, domain.TasasComision)
	for _, c := range comisiones {
		if _, err := tx.Exec(
			`INSERT INTO comisiones
			   (nivel, monto, origen_email, pedido_id, referido_id, porcentaje, estado)
			 VALUES ($1, $2, $3, $4, $5, $6, 'pendiente')`,
			c.Nivel, c.Monto, email, pedidoID, c.ReferidoID, c.Porcentaje,
		); err != nil {
			return fmt.Errorf("error registrando la comisión de nivel %d: %w", c.Nivel, err)
		}
	}
	log.Printf("[DB] Pedido #%d → %d comisiones en el ledger (total %.2f)",
		pedidoID, len(comisiones), domain.TotalComisiones(comisiones))
	return nil
}

// ActualizarVentasUpline suma el total del pedido al contador `ventas` de
// cada nodo del upline.
//
// Es un cache de lectura, NO la fuente de verdad del dinero: el ledger
// (`comisiones`) lo es. Por eso va fuera de la transacción y en goroutine —
// si falla, la venta sigue siendo válida y sólo el contador queda atraso.
// Losing una actualización de un contador es un problema distinto (y mucho
// menor) que perder una fila del ledger.
func (r *Repository) ActualizarVentasUpline(emailUsuario string, total float64) error {
	if r.db == nil {
		return errors.New("base de datos no disponible")
	}

	var refID sql.NullInt64
	err := r.db.QueryRow(
		`SELECT referido_id FROM usuarios WHERE email = $1`, emailUsuario,
	).Scan(&refID)
	if err != nil || !refID.Valid {
		// El usuario no tiene nodo (compra desde la raíz): no hay a quién
		// acreditarle nada. No es un error.
		return nil
	}

	// Un solo UPDATE que sube los 3 niveles del upline de una vez.
	// El ancla es el padre del nodo del comprador: el vendedor de sí mismo
	// no existe. Ojo: la CTE debe proyectar parent_id en el ancla, porque el
	// término recursivo lee `up.parent_id` — sin esa columna Postgres
	// responde "column does not exist" en tiempo de ejecución.
	_, err = r.db.Exec(
		`WITH RECURSIVE upline AS (
		     SELECT r.id, r.parent_id, 1 AS nivel
		       FROM referidos propio
		       JOIN referidos r ON r.id = propio.parent_id
		      WHERE propio.id = $1
		     UNION ALL
		     SELECT r.id, r.parent_id, up.nivel + 1
		       FROM referidos r JOIN upline up ON r.id = up.parent_id
		      WHERE up.nivel < $3
		 )
		 UPDATE referidos SET ventas = ventas + $2
		  WHERE id IN (SELECT id FROM upline)`,
		refID.Int64, total, domain.MaxNivelesComision,
	)
	if err != nil {
		return fmt.Errorf("error actualizando ventas del upline: %w", err)
	}
	return nil
}

// GetComisiones devuelve las filas del ledger que casan con el filtro,
// paginadas, junto con el total de filas que cumplen el filtro (no el de la
// página). ReferidoID nil = todos los distribuidores (uso admin).
func (r *Repository) GetComisiones(filtro interfaces.ComisionFiltro) ([]models.Comision, int, error) {
	if r.db == nil {
		return nil, 0, errors.New("base de datos no disponible")
	}
	page, limit := normalizarPagina(filtro.Page, filtro.Limit)

	where := "WHERE 1=1"
	args := []interface{}{}
	if filtro.ReferidoID != nil {
		args = append(args, *filtro.ReferidoID)
		where += fmt.Sprintf(" AND referido_id = $%d", len(args))
	}
	if filtro.Estado != "" {
		args = append(args, filtro.Estado)
		where += fmt.Sprintf(" AND estado = $%d", len(args))
	}

	var total int
	if err := r.db.QueryRow(
		`SELECT count(*) FROM comisiones `+where, args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("error contando comisiones: %w", err)
	}

	args = append(args, limit, (page-1)*limit)
	rows, err := r.db.Query(
		`SELECT id, pedido_id, referido_id, nivel, porcentaje, monto, estado, created_at
		   FROM comisiones `+where+fmt.Sprintf(
			` ORDER BY created_at DESC, id DESC LIMIT $%d OFFSET $%d`,
			len(args)-1, len(args)),
		args...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("error consultando comisiones: %w", err)
	}
	defer rows.Close()

	comisiones := make([]models.Comision, 0)
	for rows.Next() {
		var c models.Comision
		var pedidoID, referidoID sql.NullInt64
		var porcentaje, monto string
		var creado time.Time
		if err := rows.Scan(&c.ID, &pedidoID, &referidoID, &c.Nivel,
			&porcentaje, &monto, &c.Estado, &creado); err != nil {
			return nil, 0, fmt.Errorf("error escaneando comisión: %w", err)
		}
		// NUMERIC llega como texto desde lib/pq: se convierte con ParseFloat
		// en vez de dejar que database/sql caste y trunque.
		if c.Porcentaje, err = strconv.ParseFloat(porcentaje, 64); err != nil {
			return nil, 0, fmt.Errorf("porcentaje inválido en la comisión %d: %w", c.ID, err)
		}
		if c.Monto, err = strconv.ParseFloat(monto, 64); err != nil {
			return nil, 0, fmt.Errorf("monto inválido en la comisión %d: %w", c.ID, err)
		}
		if pedidoID.Valid {
			v := int(pedidoID.Int64)
			c.PedidoID = &v
		}
		if referidoID.Valid {
			v := int(referidoID.Int64)
			c.ReferidoID = &v
		}
		c.CreadoEn = creado.UTC().Format(time.RFC3339)
		comisiones = append(comisiones, c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error recorriendo comisiones: %w", err)
	}
	return comisiones, total, nil
}

// GetTotalesComisiones agrupa el ledger por estado. Si referidoID no es nil,
// sólo totaliza ese distribuidor.
//
// Es la consulta que responde "¿cuánto le debo a esta persona?" y la que
// usará POST /api/admin/liquidaciones para decidir si hay algo que pagar.
func (r *Repository) GetTotalesComisiones(referidoID *int) ([]models.ComisionTotal, error) {
	if r.db == nil {
		return nil, errors.New("base de datos no disponible")
	}
	query := `SELECT estado, COALESCE(SUM(monto), 0), count(*)
	            FROM comisiones`
	args := []interface{}{}
	if referidoID != nil {
		query += ` WHERE referido_id = $1`
		args = append(args, *referidoID)
	}
	query += ` GROUP BY estado ORDER BY estado`

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("error totalizando comisiones: %w", err)
	}
	defer rows.Close()

	totales := make([]models.ComisionTotal, 0, 4)
	for rows.Next() {
		var t models.ComisionTotal
		var monto string
		if err := rows.Scan(&t.Estado, &monto, &t.Cantidad); err != nil {
			return nil, fmt.Errorf("error escaneando total: %w", err)
		}
		v, convErr := strconv.ParseFloat(monto, 64)
		if convErr != nil {
			return nil, fmt.Errorf("total inválido para %q: %w", t.Estado, convErr)
		}
		t.Total = v
		totales = append(totales, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error recorriendo totales: %w", err)
	}
	return totales, nil
}

// ─────────────────────────── Liquidaciones ───────────────────────────────────

// Liquidar paga todas las comisiones 'pendiente' de un distribuidor en una
// única transacción.
//
// Todo el dance es por concurrencia: dos admins que pulsan "liquidar" a la vez
// no pueden pagar dos veces lo mismo. El bloqueo SELECT … FOR UPDATE sobre
// las comisiones pendientes serializa a los dos; el segundo ve el conjunto
// vacío y recibe ErrNadaQueLiquidar.
//
// El orden importa: se bloquean y suman las comisiones PRIMERO, se inserta la
// liquidación con ese total, y sólo después se marcan las comisiones como
// pagadas enlazándolas a esa liquidación. Si algo falla, el ROLLBACK deja las
// comisiones como estaban.
func (r *Repository) Liquidar(referidoID int) (*models.Liquidacion, error) {
	if r.db == nil {
		return nil, errors.New("base de datos no disponible")
	}

	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("error iniciando transacción de liquidación: %w", err)
	}
	defer tx.Rollback()

	// 1) Bloquear las comisiones pendientes del distribuidor.
	//    El FOR UPDATE es lo que impide la doble liquidación.
	rows, err := tx.Query(
		`SELECT id, COALESCE(monto, 0)
		   FROM comisiones
		  WHERE referido_id = $1 AND estado = 'pendiente'
		  ORDER BY id
		  FOR UPDATE`,
		referidoID,
	)
	if err != nil {
		return nil, fmt.Errorf("error bloqueando comisiones del referido %d: %w", referidoID, err)
	}
	var (
		ids   []int
		total float64
	)
	for rows.Next() {
		var (
			id    int
			monto string
		)
		if err := rows.Scan(&id, &monto); err != nil {
			rows.Close()
			return nil, fmt.Errorf("error leyendo comisiones a liquidar: %w", err)
		}
		v, convErr := strconv.ParseFloat(monto, 64)
		if convErr != nil {
			rows.Close()
			return nil, fmt.Errorf("monto inválido en la comisión %d: %w", id, convErr)
		}
		ids = append(ids, id)
		total += v
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error recorriendo comisiones a liquidar: %w", err)
	}

	if len(ids) == 0 {
		return nil, models.ErrNadaQueLiquidar
	}

	// 2) Registrar el pago.
	var liq models.Liquidacion
	var creado, pagada time.Time
	err = tx.QueryRow(
		`INSERT INTO liquidaciones (referido_id, total, periodo, estado, pagada_en)
		 VALUES ($1, $2, $3, 'pagada', NOW())
		 RETURNING id, referido_id, total, periodo, estado, creado_en, pagada_en`,
		referidoID, total, domain.PeriodoActual(),
	).Scan(&liq.ID, &liq.ReferidoID, &liq.Total, &liq.Periodo, &liq.Estado, &creado, &pagada)
	if err != nil {
		return nil, fmt.Errorf("error registrando la liquidación: %w", err)
	}
	liq.CreadoEn = creado.UTC().Format(time.RFC3339)
	pagadaStr := pagada.UTC().Format(time.RFC3339)
	liq.PagadaEn = &pagadaStr

	// 3) Marcar las comisiones como pagadas y enlazarlas al pago.
	if _, err := tx.Exec(
		`UPDATE comisiones
		    SET estado = 'pagada', liquidacion_id = $1
		  WHERE id = ANY($2)`,
		liq.ID, pq.Array(ids),
	); err != nil {
		return nil, fmt.Errorf("error marcando comisiones como pagadas: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("error en commit de la liquidación: %w", err)
	}
	log.Printf("[DB] Liquidación #%d: referido=%d total=%.2f (%d comisiones)",
		liq.ID, referidoID, liq.Total, len(ids))
	return &liq, nil
}

// GetLiquidaciones devuelve el historial de pagos, paginado.
func (r *Repository) GetLiquidaciones(page, limit int) ([]models.Liquidacion, int, error) {
	return r.consultarLiquidaciones(nil, page, limit)
}

// GetLiquidacionesDe devuelve el historial de un distribuidor concreto.
func (r *Repository) GetLiquidacionesDe(referidoID, page, limit int) ([]models.Liquidacion, int, error) {
	return r.consultarLiquidaciones(&referidoID, page, limit)
}

func (r *Repository) consultarLiquidaciones(referidoID *int, page, limit int) ([]models.Liquidacion, int, error) {
	if r.db == nil {
		return nil, 0, errors.New("base de datos no disponible")
	}
	page, limit = normalizarPagina(page, limit)

	where := ""
	args := []interface{}{}
	if referidoID != nil {
		args = append(args, *referidoID)
		where = "WHERE referido_id = $1"
	}

	var total int
	if err := r.db.QueryRow(
		`SELECT count(*) FROM liquidaciones `+where, args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("error contando liquidaciones: %w", err)
	}

	args = append(args, limit, (page-1)*limit)
	rows, err := r.db.Query(
		`SELECT id, referido_id, total, COALESCE(periodo,''), estado, creado_en, pagada_en
		   FROM liquidaciones `+where+fmt.Sprintf(
			` ORDER BY id DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args)),
		args...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("error consultando liquidaciones: %w", err)
	}
	defer rows.Close()

	liquidaciones := make([]models.Liquidacion, 0)
	for rows.Next() {
		var l models.Liquidacion
		var totalStr string
		var pagada sql.NullTime
		if err := rows.Scan(&l.ID, &l.ReferidoID, &totalStr, &l.Periodo,
			&l.Estado, &l.CreadoEn, &pagada); err != nil {
			return nil, 0, fmt.Errorf("error escaneando liquidación: %w", err)
		}
		v, convErr := strconv.ParseFloat(totalStr, 64)
		if convErr != nil {
			return nil, 0, fmt.Errorf("total inválido en la liquidación %d: %w", l.ID, convErr)
		}
		l.Total = v
		if pagada.Valid {
			s := pagada.Time.UTC().Format(time.RFC3339)
			l.PagadaEn = &s
		}
		liquidaciones = append(liquidaciones, l)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error recorriendo liquidaciones: %w", err)
	}
	return liquidaciones, total, nil
}

// GetDeuda devuelve cuánto se le debe a cada distribuidor: la suma de sus
// comisiones 'pendiente'. Es la consulta que alimenta la pantalla de
// liquidación y responde "¿a quién tengo que pagarle?".
//
// Se hace con JOIN a referidos/usuarios para que la pantalla muestre nombre
// y código, no ids sueltos.
func (r *Repository) GetDeuda() ([]models.DeudaDistribuidor, error) {
	if r.db == nil {
		return nil, errors.New("base de datos no disponible")
	}
	rows, err := r.db.Query(
		`SELECT r.id,
		        r.codigo,
		        COALESCE(u.nombre, r.codigo) AS nombre,
		        COALESCE(SUM(c.monto), 0) AS total,
		        count(c.id) AS comisiones,
		        MAX(l.pagada_en)
		   FROM referidos r
		   JOIN comisiones c ON c.referido_id = r.id AND c.estado = 'pendiente'
		   LEFT JOIN usuarios u ON u.id = r.usuario_id
		   LEFT JOIN LATERAL (
		        SELECT MAX(pagada_en) AS pagada_en
		          FROM liquidaciones
		         WHERE referido_id = r.id
		   ) l ON true
		  GROUP BY r.id, r.codigo, u.nombre
		  ORDER BY total DESC`)
	if err != nil {
		return nil, fmt.Errorf("error consultando la deuda: %w", err)
	}
	defer rows.Close()

	deudas := make([]models.DeudaDistribuidor, 0)
	for rows.Next() {
		var d models.DeudaDistribuidor
		var totalStr string
		var ultima sql.NullTime
		if err := rows.Scan(&d.ReferidoID, &d.Codigo, &d.Nombre, &totalStr,
			&d.Comisiones, &ultima); err != nil {
			return nil, fmt.Errorf("error escaneando la deuda: %w", err)
		}
		v, convErr := strconv.ParseFloat(totalStr, 64)
		if convErr != nil {
			return nil, fmt.Errorf("total inválido para el referido %d: %w", d.ReferidoID, convErr)
		}
		d.Total = v
		if ultima.Valid {
			d.UltimaPago = ultima.Time.UTC().Format(time.RFC3339)
		}
		deudas = append(deudas, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error recorriendo la deuda: %w", err)
	}
	return deudas, nil
}

// ─────────────────────────── Reportes ───────────────────────────────────────
//
// Todas las consultas de esta sección excluyen los pedidos `cancelado`.
//
// La razón es de dinero, no de gusto: al cancelar, CambiarEstadoPedido
// devuelve el stock y revierte las comisiones del pedido. Contarlo como
// venta infla la facturación y las comisiones devengadas a la vez.

// appendRangoFecha añade el recorte temporal a los argumentos y devuelve un
// fragmento "AND ..." parametrizado con los placeholders correctos.
//
// Se reutiliza entre `pedidos` y `comisiones` porque el dashboard siempre
// recorta las dos por el mismo periodo: si sólo se recortara una, el ticket
// medio y las comisiones dejarían de cuadrar entre sí.
//
// Cada tabla usa su propio slice de argumentos porque son dos consultas
// independientes: los placeholders empiezan en $1 en ambas.
func appendRangoFecha(
	args []interface{}, desde, hasta *time.Time,
) (string, []interface{}) {
	frag := ""
	if desde != nil {
		args = append(args, *desde)
		frag += fmt.Sprintf(" AND created_at >= $%d", len(args))
	}
	if hasta != nil {
		args = append(args, *hasta)
		frag += fmt.Sprintf(" AND created_at < $%d", len(args))
	}
	return frag, args
}

// GetReporteResumen devuelve los KPIs del periodo.
//
// `ticket_promedio` se calcula en Go, no en SQL, con guarda de división por
// cero: un periodo sin pedidos da 0, no NaN ni un 500 por división en cero.
func (r *Repository) GetReporteResumen(filtro interfaces.ReporteFiltro) (*models.ReporteResumen, error) {
	if r.db == nil {
		return nil, errors.New("base de datos no disponible")
	}

	resumen := &models.ReporteResumen{}

	// Ventas y pedidos: una sola pasada sobre `pedidos`.
	// El SUM devuelve NUMERIC, que lib/pq entrega como texto: se convierte
	// con ParseFloat en vez de dejar que database/sql caste y trunque.
	argsPedidos := []interface{}{}
	rangoPedidos, argsPedidos := appendRangoFecha(argsPedidos, filtro.Desde, filtro.Hasta)

	var ventasStr string
	if err := r.db.QueryRow(
		`SELECT COALESCE(SUM(total), 0), count(*)
		   FROM pedidos
		  WHERE estado <> 'cancelado'`+rangoPedidos,
		argsPedidos...,
	).Scan(&ventasStr, &resumen.NumPedidos); err != nil {
		return nil, fmt.Errorf("error agregando ventas: %w", err)
	}
	ventas, err := strconv.ParseFloat(ventasStr, 64)
	if err != nil {
		return nil, fmt.Errorf("total vendido inválido %q: %w", ventasStr, err)
	}
	resumen.TotalVendido = ventas

	// Comisiones devengadas: consulta APARTE, nunca un JOIN con `pedidos`.
	// Unir las dos tablas en la misma fila multiplicaría `total` por el número
	// de comisiones del pedido (hasta 3, uno por nivel del upline) y
	// falsearía la facturación por ese factor.
	//
	// Se excluye 'revertida': una comisión revertida nunca se devengó.
	// 'pendiente', 'aprobada' y 'pagada' sí cuentan — devengar no es pagar.
	argsCom := []interface{}{}
	rangoCom, argsCom := appendRangoFecha(argsCom, filtro.Desde, filtro.Hasta)

	var comStr string
	if err := r.db.QueryRow(
		`SELECT COALESCE(SUM(monto), 0)
		   FROM comisiones
		  WHERE estado <> 'revertida'`+rangoCom,
		argsCom...,
	).Scan(&comStr); err != nil {
		return nil, fmt.Errorf("error agregando comisiones devengadas: %w", err)
	}
	devengadas, err := strconv.ParseFloat(comStr, 64)
	if err != nil {
		return nil, fmt.Errorf("comisiones devengadas inválidas %q: %w", comStr, err)
	}
	resumen.ComisionesDevengadas = devengadas

	if resumen.NumPedidos > 0 {
		resumen.TicketPromedio = resumen.TotalVendido / float64(resumen.NumPedidos)
	}

	return resumen, nil
}

// GetTopProductos rankea por ingresos.
//
// Se agrupa por `pedido_items.producto_id` y no por el JOIN con `productos`:
// el nombre en pedido_items es la foto histórica del precio y del nombre en
// el momento de la compra, que es lo que un reporte de ventas debe mostrar.
func (r *Repository) GetTopProductos(limit int) ([]models.TopProducto, error) {
	if r.db == nil {
		return nil, errors.New("base de datos no disponible")
	}
	limit = normalizarLimitReporte(limit)

	rows, err := r.db.Query(
		`SELECT pi.producto_id,
		        MIN(pi.nombre),
		        SUM(pi.cantidad),
		        SUM(pi.cantidad * pi.precio)
		   FROM pedido_items pi
		   JOIN pedidos p ON p.id = pi.pedido_id
		  WHERE p.estado <> 'cancelado'
		  GROUP BY pi.producto_id
		  ORDER BY SUM(pi.cantidad * pi.precio) DESC, pi.producto_id
		  LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("error consultando top de productos: %w", err)
	}
	defer rows.Close()

	top := make([]models.TopProducto, 0, limit)
	for rows.Next() {
		var t models.TopProducto
		var ingresosStr string
		if err := rows.Scan(&t.ProductoID, &t.Nombre, &t.Unidades, &ingresosStr); err != nil {
			return nil, fmt.Errorf("error escaneando top de productos: %w", err)
		}
		v, convErr := strconv.ParseFloat(ingresosStr, 64)
		if convErr != nil {
			return nil, fmt.Errorf("ingresos inválidos para el producto %d: %w", t.ProductoID, convErr)
		}
		t.Ingresos = v
		top = append(top, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error recorriendo el top de productos: %w", err)
	}
	return top, nil
}

// GetTopReferidos rankea distribuidores por ventas acumuladas.
//
// Ventas sale del contador `referidos.ventas` (rápido, best-effort) y
// Comisiones del ledger real (lento, exacto). Que difieran es correcto: el
// contador se actualiza en una goroutine tras el commit del checkout, así que
// puede ir un instante por detrás del ledger. Para auditoría hay que usar
// /api/admin/comisiones, no este ranking.
func (r *Repository) GetTopReferidos(limit int) ([]models.TopReferido, error) {
	if r.db == nil {
		return nil, errors.New("base de datos no disponible")
	}
	limit = normalizarLimitReporte(limit)

	rows, err := r.db.Query(
		`SELECT r.id,
		        r.codigo,
		        COALESCE(u.nombre, r.codigo),
		        r.ventas,
		        COALESCE(SUM(c.monto) FILTER (WHERE c.estado <> 'revertida'), 0)
		   FROM referidos r
		   LEFT JOIN usuarios u ON u.id = r.usuario_id
		   LEFT JOIN comisiones c ON c.referido_id = r.id
		  GROUP BY r.id, r.codigo, u.nombre, r.ventas
		  ORDER BY r.ventas DESC, r.id
		  LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("error consultando top de referidos: %w", err)
	}
	defer rows.Close()

	top := make([]models.TopReferido, 0, limit)
	for rows.Next() {
		var t models.TopReferido
		var ventasStr, comStr string
		if err := rows.Scan(&t.ReferidoID, &t.Codigo, &t.Nombre, &ventasStr, &comStr); err != nil {
			return nil, fmt.Errorf("error escaneando top de referidos: %w", err)
		}
		if t.Ventas, err = strconv.ParseFloat(ventasStr, 64); err != nil {
			return nil, fmt.Errorf("ventas inválidas para el referido %d: %w", t.ReferidoID, err)
		}
		if t.Comisiones, err = strconv.ParseFloat(comStr, 64); err != nil {
			return nil, fmt.Errorf("comisiones inválidas para el referido %d: %w", t.ReferidoID, err)
		}
		top = append(top, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error recorriendo el top de referidos: %w", err)
	}
	return top, nil
}

// GetComisionesPendientes devuelve el ledger que aún no se ha liquidado.
//
// Es la lista de trabajo del admin antes de pulsar "liquidar": lo que hay
// pendiente y por qué. No se limita por referido a propósito — la pantalla es
// global y el admin puede liquidar a cualquiera (POST
// /api/admin/liquidaciones).
func (r *Repository) GetComisionesPendientes(limit int) ([]models.Comision, error) {
	if r.db == nil {
		return nil, errors.New("base de datos no disponible")
	}
	limit = normalizarLimitReporte(limit)

	rows, err := r.db.Query(
		`SELECT id, pedido_id, referido_id, nivel, porcentaje, monto, estado, created_at
		   FROM comisiones
		  WHERE estado = 'pendiente'
		  ORDER BY created_at DESC, id DESC
		  LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("error consultando comisiones pendientes: %w", err)
	}
	defer rows.Close()

	comisiones := make([]models.Comision, 0, limit)
	for rows.Next() {
		var c models.Comision
		var pedidoID, referidoID sql.NullInt64
		var porcentaje, monto string
		var creado time.Time
		if err := rows.Scan(&c.ID, &pedidoID, &referidoID, &c.Nivel,
			&porcentaje, &monto, &c.Estado, &creado); err != nil {
			return nil, fmt.Errorf("error escaneando comisión pendiente: %w", err)
		}
		if c.Porcentaje, err = strconv.ParseFloat(porcentaje, 64); err != nil {
			return nil, fmt.Errorf("porcentaje inválido en la comisión %d: %w", c.ID, err)
		}
		if c.Monto, err = strconv.ParseFloat(monto, 64); err != nil {
			return nil, fmt.Errorf("monto inválido en la comisión %d: %w", c.ID, err)
		}
		if pedidoID.Valid {
			v := int(pedidoID.Int64)
			c.PedidoID = &v
		}
		if referidoID.Valid {
			v := int(referidoID.Int64)
			c.ReferidoID = &v
		}
		c.CreadoEn = creado.UTC().Format(time.RFC3339)
		comisiones = append(comisiones, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error recorriendo comisiones pendientes: %w", err)
	}
	return comisiones, nil
}

// normalizarLimitReporte acota el `?limit=` de los reportes.
//
// Default 10 y tope 50, los valores que fija openapi.yaml para
// /api/reportes/top-productos y /api/reportes/top-referidos. Sin tope, un
// `?limit=999999` convertiría un endpoint de dashboard en un volcado de tabla.
func normalizarLimitReporte(limit int) int {
	if limit <= 0 {
		return 10
	}
	if limit > 50 {
		return 50
	}
	return limit
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
