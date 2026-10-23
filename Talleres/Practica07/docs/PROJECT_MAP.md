# PROJECT MAP — framework-web-2026-02

> Mapa arquitectónico del workspace. Leer ESTE archivo antes de explorar código.
> Proyecto didáctico "MultiCatálogo": tienda de cosméticos con red de ventas
> multinivel (MLM). Asignatura: Framework Programación Web (UPSE, 2026-02).

## Visión general

```
React SPA (:5173) ──fetch──► Go/Fiber API (:3000, env PORT) ──pgx────► PostgreSQL 16 (:5434, Docker)
                                    │
                                    └──go-redis──► Redis (:6379, Docker) — caché de lecturas
```

## Documentos raíz (meta, no código)

| Archivo | Contenido |
|---|---|
| `PROJECT_MAP.md` | Este archivo. Índice de navegación del proyecto. |
| `REQUIREMENTS_CHECKLIST.md` | Estado del proyecto en **fases por Unidad/Tema del sílabo** (U1 SPA → U2 Go/Fiber → U3 FastAPI SaaS → U4 despliegue) + roadmap post-sílabo. Principios: seguridad por defecto, DI, funciones puras de dominio, mobile-first. |
| `Silabo_Framework_Programacion_Web.md` | Sílabo oficial: 4 unidades (Frontend/Backend/Full-stack/Despliegue). |
| `Planificacion_Unidad1_Tema5.md` | Planificación del tema activo (interfaces full-screen + flujos MLM). |
| `Guia_Practica_Tema5_leccion.md` | Guía paso a paso (14 pasos) de la Práctica 03. |
| `.windsurfrules` | Reglas del agente IA (ahorro de tokens, navegación, seguimiento). |

## Unidad1_Frontend/Proyecto_base — SPA

Stack: React 19 + TypeScript + Vite 8 + Tailwind CSS 4 + React Router 7.
Comandos: `npm run dev` (:5173) | `npm run build` | `npm run lint`.

| Ruta (`src/`) | Responsabilidad |
|---|---|
| `App.tsx` | Router, guardas `ProtectedRoute`/`AdminRoute`, `CartBoundary` (carrito por usuario). |
| `main.tsx` | Bootstrap de React. |
| `index.css` | Tailwind v4 (import único). |
| `context/AuthContext.tsx` | Sesión: `{email, rol}` admin\|cliente. Sin persistencia. |
| `context/CartContext.tsx` | Carrito por usuario persistido en localStorage (`multicatalogo_carrito_<email>`). |
| `data/productos.ts` | Solo tipos `Producto` + lista `categorias` (los datos vienen de la API). |
| `data/red.ts` | Tipos `Referido` + funciones puras: comisiones (10/5/2 %), totales, nivel. |
| `services/productosService.ts` | `getProductos()`, `getProductoById()` → `GET /api/productos[/:id]`. |
| `services/redService.ts` | `getRed()` → `GET /api/red`. |
| `components/Login.tsx` | Login; lee `APIError.message` del backend; redirige por rol. |
| `components/Storefront.tsx` | `/tienda` — hero full-screen, categorías, destacados, plan MLM. |
| `components/Catalogo.tsx` | `/catalogo` — búsqueda + filtro por `?categoria=` (useMemo, useSearchParams). |
| `components/DetalleProducto.tsx` | `/producto/:id` — detalle + galería/lightbox. |
| `components/Carrito.tsx` | `/carrito` — ítems, cantidades, total. |
| `components/Checkout.tsx` | `/checkout` — formulario → `POST /api/checkout` real → resumen; muestra `APIError`. |
| `components/Confirmacion.tsx` | `/confirmacion` — pedido confirmado, limpia carrito. |
| `components/MiRed.tsx` | `/mi-red` (admin) — árbol multinivel + comisiones. |
| `components/Dashboard.tsx` | `/` (admin) — KPIs derivados de la red. |
| `components/Layout.tsx` `Navbar.tsx` `Sidebar.tsx` | Shell: sidebar colapsable, navbar con carrito/avatar/logout. |

## Unidad2_Backend/multicatalogo-backend — API REST

Stack: Go 1.26 + Fiber v2 + pgx v5 (pool) + godotenv + PostgreSQL 16 + Redis (go-redis v9).
Comandos: `docker compose up -d` | `go run .` (:3000, sobreescribible con `PORT`) | `go build ./...`.
Capas: `routes` → `controllers` → `repository` → `config.DB`.

| Ruta | Responsabilidad |
|---|---|
| `main.go` | Entry point: `Connect()`+`ConnectRedis()`, ensambla DI (`NewPostgresRepository`→`NewRedisCache`→`NewControllers`), CORS (5173-5175), `app.Listen(PORT‖":3000")`. |
| `config/db.go` | Pool pgx; carga `.env`; `Connect()`/`Close()`. |
| `config/redis.go` | Cliente go-redis global `RedisClient`; `ConnectRedis()`/`CloseRedis()`; si Redis cae, la API sigue sin caché. |
| `models/models.go` | `LoginRequest`, `Usuario`, `Producto`(+stock), `Referido`, `CheckoutRequest`/`ItemPedido`, `APIError{status,message,details}`. |
| `routes/routes.go` | `SetupRoutes(app, *Controllers)` — registra endpoints con métodos de structs inyectados. |
| `controllers/controllers.go` | Agregador `Controllers` + `NewControllers(repo, cache)` — punto de ensamblado de la DI. |
| `controllers/authController.go` | `AuthController{repo}` → `POST /api/login` — valida body + campos vacíos (400), 401 con APIError. |
| `controllers/checkoutController.go` | `CheckoutController{repo,cache}` → `POST /api/checkout`: transacción ACID, goroutine comisiones, invalidación caché. |
| `controllers/prodController.go` | `GET /api/productos` (Cache-Aside `productos:all`, TTL 5 min), `GET /api/productos/:id` (400 si id no numérico, 404). |
| `controllers/comisionesController.go` | `GET /api/comisiones/mias` (identidad del token) + `GET /api/admin/comisiones` (auditoría con filtro). Valida `?estado=` contra el CHECK del schema. |
| `controllers/networkController.go` | `GET /api/red` (admin) + `GET /api/red/mia` (token): árbol multinivel desde recursive CTE, Cache-Aside con **versión de clave** (`red:ver`). |
| `domain/comisiones.go` | `TasasComision` (10/5/2 %), `RepartirComisiones(total, cadena, tasas)` pura + `TotalComisiones`. El reparto vive en dominio, no en SQL. |
| `domain/red.go` | `NodoPlano` (fila plana) + `ConstruirArbol` puro (ensambla la jerarquía, corta ciclos, acota profundidad) + `ContarNodos`/`TotalVentas`. Sin I/O, testeado en `red_test.go`. |
| `controllers/healthController.go` | `GET /api/health` — `{"status":"ok"}`. |
| `repository/repository.go` | Interfaz `DBRepository` + `PostgresRepository{db}`: SQL puro; árbol `referidos` por `parent_id`; `CrearPedido` transaccional (`FOR UPDATE`); `CalcularComisiones` (upline 3 niveles). |
| `repository/cache.go` | Interfaz `CacheRepository` + `RedisCache{client}` (Get/Set/Del nil-safe). |
| `database/migrations/001_schema_base.sql` | DDL base: `pedidos`, `pedido_items`, `comisiones` + extensión `pgcrypto`. Sin seed. |
| `database/migrations/002_backoffice.sql` | `usuarios`, `productos`, `referidos`, `liquidaciones` + columnas/FKs/índices del back-office. Idempotente. |
| `database/migrations/003_seed.sql` | Datos de desarrollo: 2 cuentas (`12345678`), 3 nodos de red, 4 productos. Idempotente. |
| `docker-compose.yml` | Postgres 16 + volumen `multicatalogo_pgdata` + monta `database/migrations/` en `/docker-entrypoint-initdb.d` (se aplican en orden alfabético **sólo en el primer arranque** del volumen); `redis:alpine` (:6379) con healthcheck; servicio `backend`. |
| `.env` (gitignored) / `.env.example` | `DB_HOST/PORT/USER/PASSWORD/NAME` + `REDIS_HOST/PORT`. Puerto host **5434** (ver nota abajo). |

## Datos y contratos clave

- Errores API: siempre `{"status":N,"message":"...","details"?}` (struct `APIError`).
- Login OK: `{"token","email","rol"}`. Cuentas seed: `admin@upse.edu.ec` / `cliente@upse.edu.ec` — password **`12345678`** (ver `003_seed.sql`).
- Checkout: `POST /api/checkout` `{nombre,email,direccion,ciudad,telefono,metodoPago,items:[{producto_id,cantidad}],total}` → `201 {"pedido_id","total","message"}`; el total se recalcula en servidor y el stock se descuenta en la misma transacción.
- Árbol de referidos: raíz id 1 "TU" (`referidos.codigo`), 3 nodos semilla (TU, ADMIN, LUIS), comisiones por nivel.
- **Deuda conocida:** no existe vista de liquidaciones ni de reportes (Sprints 3 y 5). `referidos.ventas` es un contador best-effort: si la goroutine falla, el ledger sigue siendo correcto y el contador queda atrasado hasta la siguiente venta.

## Notas operativas

- Puerto DB en **5433**: es el puerto que publica este compose en el host (Postgres corre en 5432 *dentro* del contenedor). 5432 lo ocupa el Postgres local del sistema. Si el contenedor `postgres-funcional` de otro proyecto tomara 5433, cambiar `ports` en `docker-compose.yml` y `DATABASE_URL` en `backend/.env` + `Makefile`.
- Puerto API configurable con `PORT` (defecto 3000): `backend-haskell` (otro proyecto) ocupa 3000; para probar usar `PORT=3001 go run .`.
- **Migraciones:** se aplican solas sólo en el primer arranque del volumen. Para un entorno limpio usar `make db-reset` (destructivo) o `make db-migrate` sobre un volumen existente. `make db-status` muestra tablas y conteos.
- Caché Cache-Aside: `productos:all` (TTL 5 min) y `red:tree:v{N}:{all|u<id>}` (TTL 10 min), donde `N` es el contador `red:ver`. Subir `red:ver` invalida toda la red de golpe sin enumerar claves — hace falta porque un checkout acredita hasta 3 nodos del upline y un alta cuelga un nodo del padre. Si Redis no responde, la API consulta PostgreSQL directamente.
- Comisiones MLM: el ledger se escribe **dentro de la transacción del checkout** (`insertarComisionesTx`), con el reparto calculado por `domain.RepartirComisiones`. Sólo queda fuera el contador `referidos.ventas`, que es cache de lectura y se actualiza en goroutine tras el COMMIT. El ancla del upline es el *padre* del nodo del comprador: el nivel 1 es el referente, nunca uno mismo.
- Passwords en texto plano (temporal didáctico); token JWT ficticio; endpoints sin auth real.
- Pendiente de commit: todo `Unidad2_Backend/`, `Proyecto_base/` y los `.md` raíz.
