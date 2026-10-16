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
| `controllers/redController.go` | `GET /api/red[?email=]` — árbol multinivel; Cache-Aside `red:tree:{email}`, TTL 10 min. |
| `controllers/healthController.go` | `GET /api/health` — `{"status":"ok"}`. |
| `repository/repository.go` | Interfaz `DBRepository` + `PostgresRepository{db}`: SQL puro; árbol `referidos` por `parent_id`; `CrearPedido` transaccional (`FOR UPDATE`); `CalcularComisiones` (upline 3 niveles). |
| `repository/cache.go` | Interfaz `CacheRepository` + `RedisCache{client}` (Get/Set/Del nil-safe). |
| `database/init.sql` | DDL + seed: `usuarios`, `productos` (`galeria TEXT[]`, `stock`), `referidos` (`parent_id` FK), `pedidos`, `pedido_items`. |
| `database/002_checkout.sql` | Migración checkout para volúmenes existentes (`ALTER stock` + tablas pedidos). |
| `docker-compose.yml` | Postgres 16 + volumen `pgdata` + autoejecuta `init.sql`; servicio `redis:alpine` (:6379) con healthcheck. |
| `.env` (gitignored) / `.env.example` | `DB_HOST/PORT/USER/PASSWORD/NAME` + `REDIS_HOST/PORT`. Puerto host **5434** (ver nota abajo). |

## Datos y contratos clave

- Errores API: siempre `{"status":N,"message":"...","details"?}` (struct `APIError`).
- Login OK: `{"token","email","rol"}`. Cuentas: `admin@upse.edu.ec`/`cliente@upse.edu.ec` — `123456`.
- Checkout: `POST /api/checkout` `{nombre,email,direccion,ciudad,telefono,metodoPago,items:[{producto_id,cantidad}],total}` → `201 {"pedido_id","total","message"}`; el total se recalcula en servidor y el stock se descuenta en la misma transacción.
- Árbol de referidos: raíz id 0 "Tú", 7 nodos, comisiones por nivel (front: `data/red.ts`).
- Comisión esperada de la red semilla: **$319.50** total, 7 referidos.

## Notas operativas

- Puerto DB en **5434**: `postgres.exe` local ocupa 5432 y el contenedor `postgres-funcional` (otro proyecto) ocupa 5433.
- Puerto API configurable con `PORT` (defecto 3000): `backend-haskell` (otro proyecto) ocupa 3000; para probar usar `PORT=3001 go run .`.
- Caché Cache-Aside: `productos:all` (TTL 5 min) y `red:tree:{email}` (TTL 10 min); si Redis no responde, la API consulta PostgreSQL directamente. El checkout invalida ambas claves.
- Comisiones MLM: goroutine tras COMMIT suma `total` a `ventas` del upline del usuario (máx. 3 niveles por `parent_id`). Con el seed actual, `cliente` (id 2 → Luis Poveda) solo acredita a "Tú".
- Passwords en texto plano (temporal didáctico); token JWT ficticio; endpoints sin auth real.
- Pendiente de commit: todo `Unidad2_Backend/`, `Proyecto_base/` y los `.md` raíz.
