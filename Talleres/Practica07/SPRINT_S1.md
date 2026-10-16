# SPRINT S1 — Fundaciones · MultiCatálogo

> Sprint base del back-office (post-análisis). Pre-requisito de F1–F7.
> Cierra las grietas detectadas en `ANALISIS_FUNCIONAL_U2.md` y deja el
> terreno parejo para implementar registro, CRUD admin, pedidos y red real.

## Entregables

| ID | Pieza | Path |
|----|------|------|
| A1 | Contrato OpenAPI compartido BE↔FE | `openapi.yaml` |
| A2 | Migración SQL idempotente (F0 ampliado) | `backend/database/005_fundaciones.sql` |
| A3 | Hardening backend (middleware + JWT + validator + env) | `backend/middleware/`, `backend/internal/dto/` |
| A4 | Hardening frontend (apiFetch + tipos + AdminRoute + rol) | `frontend/src/services/http.ts`, `frontend/src/types/api.ts`, `frontend/src/components/AdminRoute.tsx` |
| A5 | CI mínimo (Makefiles + workflow) | `backend/Makefile`, `frontend/Makefile`, `.github/workflows/ci.yml` |

## Cambios incompatibles (migrar antes de prod)

1. **`/api/login` ahora devuelve `{token, email, rol}`** (antes: `{token, email}`).
   Frontend ya tipado en `types/api.ts`; default `rol='cliente'` si el BE viejo no lo envía.
2. **`/api/checkout` ya NO acepta `email` en el body como identidad**. Lo toma del
   JWT (`sub`). Si el body trae email, se ignora (no se persiste ni atribuye comisiones).
   Esta validación se cablea en F1 cuando el endpoint use `RequireAuth`.
3. **Endpoints `/api/admin/*` y `/api/red/mia` requieren `Authorization: Bearer`**.
   No rompen hoy (no existen) pero al crearlos en F1+ exigirán token.

## Cómo correr local

```bash
# 1) Levantar Postgres + Redis
docker compose up -d postgres redis

# 2) Aplicar migraciones (init.sql corre automático al iniciar postgres la 1ª vez;
#    para reaplicar 005 usar el archivo de migración):
psql "postgres://admin:secret@localhost:5433/multicatalogo" \
     -f backend/database/005_fundaciones.sql

# 3) Backend (requiere .env — usar backend/.env.example)
cd backend
cp .env.example .env       # editar JWT_SECRET con 32+ bytes
make tidy && make run

# 4) Frontend
cd ../frontend
cp .env.example .env
make dev
```

## Verificación S1

```bash
# Backend
cd backend
make vet && make build && make test && make vuln

# Frontend
cd ../frontend
make lint && make typecheck && make build && make audit
```

## Variables de entorno requeridas (backend)

| Variable | Requerido | Ejemplo | Notas |
|----------|-----------|---------|-------|
| `JWT_SECRET` | sí (≥32 bytes) | `change-me-in-production-32+bytes` | Firma HS256 |
| `ALLOWED_ORIGINS` | sí | `http://localhost:5173` | CSV; sin espacios |
| `PORT` | no (defecto 3000) | `3001` | Evitar colisión con otros proyectos |
| `DATABASE_URL` | sí | `postgres://admin:secret@localhost:5433/multicatalogo?sslmode=disable` | Docker usa :5432 interno, host :5433 |
| `REDIS_URL` | sí | `localhost:6379` | Si cae, API sigue sin caché |
| `APP_ENV` | no (defecto production) | `dev` | Activa `details` en errores |

## Decisiones de seguridad aplicadas

| # | Decisión | Justificación |
|---|----------|--------------|
| S1 | Rate-limit 10/min en `/api/login` y `/api/register` | Mitigar password spraying / creación masiva |
| S2 | `JWT_SECRET` validado en startup (≥32 bytes) | Evitar firma débil por error |
| S3 | CORS desde env, **no** `*` | Cerrar superficie CSRF/abuso |
| S4 | Headers seguridad (CSP, X-Frame-Options, etc.) | Defensa en profundidad XSS/clickjacking |
| S5 | HSTS solo en `APP_ENV=production` | Dev con HTTP no debe romper HSTS |
| S6 | Detalles de error solo en `APP_ENV=dev` | Evitar info disclosure en prod |
| S7 | `RequireAuth` y `RequireAdmin` listos pero **no** aplicados a rutas existentes | Backwards-compatible; se cablean en F1-F7 |
| S8 | Validator central (DTOs) | Evita drift entre controladores |
| S9 | apiFetch único + `ApiError` tipado | Contrato FE↔BE alineado con OpenAPI |
| S10 | `AdminRoute` para `/mi-red` | Solo admins ven la red global |

## Pendiente para F1 (Sprint S2)

- Endpoint real `POST /api/register` con bcrypt + transacción usuarios+referidos.
- Migrar `authController.Login` a consultar DB (en lugar de mock).
- `GET /api/checkout` con `RequireAuth` + identidad del token.
- `POST /api/register` con rate-limit ya cableado.
- Tests de integración con testcontainers (pg + redis).

## Riesgos residuales (registrados)

- Comisiones siguen fire-and-forget (sigue en roadmap → Outbox).
- Token en localStorage (XSS residual). Roadmap: cookie httpOnly + CSRF double-submit.
- `direccion/telefono` sin cifrar. Roadmap: cifrado app o pgcrypto.