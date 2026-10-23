# Deployment Guide — MultiCatálogo Practica07

**Última actualización:** 2026-10-23  
**Versión:** 1.0  
**Alcance:** Desarrollo local, staging, producción

---

## 📋 Tabla de Contenidos

1. [Requisitos Previos](#requisitos-previos)
2. [Arquitectura](#arquitectura)
3. [Desarrollo Local](#desarrollo-local)
4. [Variables de Entorno](#variables-de-entorno)
5. [Despliegue Producción](#despliegue-producción)
6. [Docker & Containers](#docker--containers)
7. [CI/CD Pipeline](#cicd-pipeline)
8. [Base de Datos](#base-de-datos)
9. [Troubleshooting](#troubleshooting)

---

## Requisitos Previos

### Sistema Operativo
- **Linux** (recomendado: Ubuntu 20.04+)
- **macOS** (Monterey+)
- **Windows** (WSL2 + Docker Desktop)

### Software Requerido

| Componente | Versión | Propósito |
|-----------|---------|----------|
| **Node.js** | ^20.19 ó ≥22.12 | Frontend (Vite 8 exige ese mínimo) |
| **npm** | 9+ | Package manager JS |
| **pnpm** | 9+ | Alternativa; ver nota de lockfiles |
| **Docker** | 20.10+ | Contenedores DB + Redis + backend |
| **Docker Compose** | 2.0+ | Orquestación local |
| **Git** | 2.30+ | Control de versiones |
| **PostgreSQL CLI** | 14+ | Gestión base de datos (opcional) |

### Verificar Instalaciones
```bash
node --version       # v20.19+ o v22.12+
npm --version        # 9.0.0+
go version           # go1.26+
docker --version     # Docker 20.10+
docker compose version   # Compose v2 (plugin, sin guion)
```

---

## Arquitectura

```
┌─────────────────────────────────────────────────────────────┐
│                  CLIENTE (Navegador)                        │
│                  http://localhost:5173                      │
└──────────────────────┬──────────────────────────────────────┘
                       │ HTTP/HTTPS
                       ▼
┌─────────────────────────────────────────────────────────────┐
│  FRONTEND (React SPA)                                       │
│  Vite + TypeScript + Tailwind + Context API                │
│  Puertos: 5173 (dev), :3000 (prod)                         │
│  Archivo: frontend/                                         │
└──────────────────────┬──────────────────────────────────────┘
                       │ REST API
                       ▼
┌─────────────────────────────────────────────────────────────┐
│  BACKEND (Go/Fiber)                                         │
│  RESTful API + Middleware (Auth, RateLimit, CORS)          │
│  Puertos: 3000 (dev), :3000 (prod)                         │
│  Archivo: backend/                                          │
└──────┬─────────────────────────────────┬────────────────────┘
       │ SQL                             │ Cache
       ▼                                 ▼
┌─────────────────────┐      ┌──────────────────────┐
│   PostgreSQL 16     │      │   Redis (caché)      │
│   Puerto: 5433      │      │   Puerto: 6379       │
│   Volumen: pgdata   │      │   Volatile (OK)      │
└─────────────────────┘      └──────────────────────┘
```

---

## Desarrollo Local

### 1. Clonar Repositorio

```bash
cd /mnt/GAMING/UNI/01\ -\ FRAMEWORK\ PROGRAMACIÓN\ WEB/Programaci-n-Web
git clone <repo-url>  # O usar tu repo local
cd Talleres/Practica07
```

### 2. Levantar Base de Datos + Redis

```bash
# Desde la raíz de Practica07
docker compose up -d

# Espera a que postgres esté healthy (~5-10 s)
docker compose ps
# STATUS debe ser "healthy" para postgres
```

> **Usa `docker compose`, no `docker-compose`.** Compose v2 es un plugin del
> daemon y se invoca sin guion. El comando con guion es la v1, que ya no se
> instala por defecto y da comportamientos distintos con `profiles`.

Atajo equivalente del Makefile: `make db-up` (levanta sólo la infra).

**Verificar conectividad:**
```bash
# PostgreSQL
docker compose exec postgres psql -U admin -d multicatalogo -c "SELECT 1;"
# Output: 1

# Redis
docker compose exec redis redis-cli PING
# Output: PONG

# Backend (sólo si lo levantaste)
curl http://localhost:3000/api/health
```

### 3. Aplicar Migraciones

`database/migrations/` está montado en `/docker-entrypoint-initdb.d`, así que
Postgres ejecuta los `.sql` **en orden alfabético, sólo cuando el volumen se
crea por primera vez**.

> `docker compose restart postgres` **NO** re-aplica las migraciones. El
> entrypoint de initdb sólo corre con un volumen vacío. Para reaplicar sobre un
> volumen existente, usa los atajos del Makefile.

```bash
# Aplicar sobre un volumen existente
make db-migrate      # psql -f de cada .sql, en orden
make db-status       # tablas + conteos
make db-reset        # ⚠ borra el volumen y re-aplica desde cero

# Verificar
docker compose exec postgres psql -U admin -d multicatalogo \
  -c "SELECT COUNT(*) FROM usuarios;"
# Output: 3 (seed data)
```

Migraciones actuales:

| Archivo | Qué añade |
|---|---|
| `001_schema_base.sql` | `pedidos`, `pedido_items`, `comisiones` + extensión `pgcrypto` |
| `002_backoffice.sql` | `usuarios`, `productos`, `referidos`, `liquidaciones` + índices, FKs, CHECKs |
| `003_seed.sql` | Datos de prueba: 3 usuarios, 3 nodos de red, 4 productos |
| `004_direccion_entrega.sql` | `pedidos.direccion`, `.ciudad`, `.telefono` + CHECKs de longitud |

Todas son **idempotentes** (`CREATE TABLE IF NOT EXISTS`, `ADD COLUMN IF NOT
EXISTS`, guardas `DO $$` sobre `information_schema`).

### 4. Backend (Go/Fiber)

```bash
cd backend

# Crear .env
cp .env.example .env

# Editar .env (cambiar valores si es necesario)
# JWT_SECRET debe ser >= 32 bytes
# Generar secreto: openssl rand -base64 32

# Descargar dependencias
go mod download

# Verificar
go vet ./...
make build        # go build ./...

# Tests
make test              # go test ./...
make test-race         # detector de carreras
make coverage          # cobertura + coverage.html

# Tests de integración: requieren PostgreSQL arriba y una base SEPARADA.
make db-test           # crea multicatalogo_test + aplica migraciones (idempotente)
make test-backend      # db-test + go test ./... con TEST_DATABASE_URL

# ⚠ Los tests de integración hacen DELETE sobre pedidos, comisiones,
#   pedido_items, liquidaciones y usuarios. Nunca deben correr contra
#   `multicatalogo` (la base de desarrollo con los datos de `make db-demo`).
#   Por eso dsnTest() cae por defecto en `multicatalogo_test`.
#   Si no hay DB, los tests hacen t.Skip y la suite queda "verde" sin haber
#   probado SQL: para saber que sí corrieron, revisa que el resumen diga
#   `ok` y no haya líneas `--- SKIP`.

# Ejecutar (lee .env)
make run             # go run .
# O directamente:
go run .

# Output esperado:
# [boot] APP_ENV=dev puerto=3000
```

**Endpoint de prueba:**
```bash
curl http://localhost:3000/api/health
# {"status":"ok"}
```

### 5. Frontend (React/Vite)

```bash
# En otra terminal
cd frontend

# Crear .env
cp .env.example .env

# Instalar dependencias
pnpm install          # = `make install` (--frozen-lockfile)

# Desarrollo (hot reload)
pnpm run dev
# ➜  Local:   http://localhost:5173/

# Verificar
make lint            # eslint
make typecheck       # tsc -b
make test             # Vitest (69 tests)
make test-coverage

# Build
make build           # genera dist/ (+ sw.js del plugin PWA)
make preview         # sirve dist/ en :4173

> **Prueba la PWA contra `preview`, no contra `dev`.** En desarrollo el
> frontend (:5173) y el backend (:3000) son distinto origen, y el service
> worker sólo cachea respuestas cross-origin si el backend manda CORS. En
> `preview` ambos comparten origen.
```

### 6. Acceder a la Aplicación

```
Frontend:  http://localhost:5173
Backend:   http://localhost:3000
PostgreSQL: localhost:5433
Redis:      localhost:6379
```

> No hay Adminer en este compose. Para ver la base con UI:
> `psql "postgres://admin:secret@localhost:5433/multicatalogo"`,
> o `dbeaver` / `pgAdmin` apuntando a `localhost:5433`.

**Usuario de prueba:**
- Email: `admin@upse.edu.ec`
- Password: `12345678`
- Rol: `admin`

---

## Variables de Entorno

### Backend (.env)

**Requeridas:**
```env
# Puerto HTTP
PORT=3000

# Entorno (production, dev)
APP_ENV=dev

# JWT Secret (mínimo 32 bytes, máximo 64 recomendado)
JWT_SECRET=pon-aqui-un-secreto-de-al-menos-32-bytes-please-please

# CORS Origins (CSV, sin espacios)
ALLOWED_ORIGINS=http://localhost:5173,http://localhost:5174

# Base de datos
DATABASE_URL=postgres://admin:secret@localhost:5433/multicatalogo?sslmode=disable

# Redis
REDIS_URL=localhost:6379
```

**Opcionales:**
```env
# Timezone (default: UTC)
TZ=America/Guayaquil

# Log level (debug, info, warn, error)
LOG_LEVEL=info
```

**Generar JWT_SECRET seguro:**
```bash
openssl rand -base64 32
# Copiar output a .env
```

### Frontend (.env)

**Requeridas:**
```env
# Backend API URL
VITE_API_URL=http://localhost:3000

# Entorno (development, production)
VITE_ENV=development
```

---

## Despliegue Producción

### Opción 1: Railway (Recomendado para Unidad 2)

**Ventaja:** Gratuito para estudiantes, fácil setup, incluye PostgreSQL.

#### Paso 1: Crear Proyecto en Railway
```bash
# Instalar CLI
npm i -g @railway/cli

# Login
railway login

# Crear proyecto
railway init
```

#### Paso 2: Crear Servicios
```bash
# PostgreSQL (desde Dashboard de Railway)
# 1. New → Database → PostgreSQL
# 2. Anotar DATABASE_URL

# Redis (desde Dashboard)
# 1. New → Add from Template → Redis
# 2. Anotar REDIS_URL

# Backend (Fiber)
# 1. New → GitHub Repo → Practica07
# 2. Environment: Go 1.26
# 3. Build: cd backend && go build -o app
# 4. Start: ./app
# 5. Anotar PORT=3000

# Frontend (React)
# 1. New → GitHub Repo → Practica07
# 2. Environment: Node 22   (Vite 8 exige ^20.19 ó >=22.12)
# 3. Build: cd frontend && pnpm install && pnpm run build
# 4. Start: npx serve -s dist -l 3000   (o hosting estático: Vercel/Netlify)
# (O usar static hosting: Vercel/Netlify en paralelo)
```

#### Paso 3: Variables de Entorno
```bash
# En Dashboard de Railway, para Backend:
PORT=3000
APP_ENV=production
JWT_SECRET=<generar con openssl rand -base64 32>
DATABASE_URL=<copiar de PostgreSQL service>
REDIS_URL=<copiar de Redis service>
ALLOWED_ORIGINS=https://tu-frontend-dominio.com
```

#### Paso 4: Deploy
```bash
# Commit y push a main
git push origin main

# Railway auto-deploya desde GitHub
# Monitorear en Dashboard → Deployments
```

### Opción 2: Vercel (Frontend) + Render (Backend + DB)

#### Frontend en Vercel
```bash
# 1. Conectar GitHub repo
# 2. Framework: Vite
# 3. Build: pnpm run build
# 4. Output: dist
# 5. Environment Variables:
#    VITE_API_URL=https://backend-render.onrender.com
```

#### Backend en Render
```bash
# 1. Conectar GitHub
# 2. Environment: Go
# 3. Build: cd backend && go build -o app
# 4. Start: ./app
# 5. Environment vars (igual a Railway)
```

#### PostgreSQL en Render
```bash
# 1. New → PostgreSQL
# 2. Copiar DATABASE_URL a Backend service
```

### Opción 3: Docker Compose en Server (VPS)

#### En tu servidor VPS (DigitalOcean, Linode, Hetzner)

**Requisito:** Docker + Docker Compose instalados

```bash
# 1. Clonar repo
git clone <repo-url> /app
cd /app

# 2. Crear .env producción
cat > .env.prod << 'EOF'
# Backend
PORT=3000
APP_ENV=production
JWT_SECRET=$(openssl rand -base64 32)
DATABASE_URL=postgres://admin:$(openssl rand -base64 16)@postgres:5432/multicatalogo?sslmode=disable
REDIS_URL=redis:6379
ALLOWED_ORIGINS=https://tu-dominio.com

# Frontend
VITE_API_URL=https://tu-dominio.com/api
EOF

# 3. Actualizar docker-compose.yml para prod
# (cambiar puertos, volumenes persistentes, etc.)

# 4. Levantar servicios
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d

# 5. Verificar
docker compose ps
curl http://localhost:3000/api/health
```

---

## Docker & Containers

### Local Development (docker-compose.yml)

El archivo declara **tres** servicios, todos sin `profiles` — `up` los levanta
todos:

| Servicio | Imagen | Puerto | Notas |
|---|---|---|---|
| `postgres` | `postgres:16-alpine` | `5433→5432` | healthcheck con `pg_isready`; volumen `multicatalogo_pgdata` |
| `redis` | `redis:alpine` | `6379` | sin volumen (es caché, se puede perder) |
| `backend` | build desde `./backend` | `3000→3000` | `depends_on` postgres *healthy* + redis; `restart: on-failure` |

```bash
# Sólo la infra (postgres + redis)
docker compose up -d postgres redis

# Todo, incluido el backend
docker compose up -d
make db-up          # atajo: sólo infra

# Estado y logs
docker compose ps
docker compose logs -f postgres
docker compose logs -f backend

# Detener
docker compose down

# Atajo equivalente
make down / make ps / make logs
```

#### ⚠️ `down -v` borra tus datos

```bash
docker compose down -v        # ← elimina el volumen multicatalogo_pgdata
```

Con `-v` se lleva la base de datos entera: usuarios, productos, pedidos. Sólo
úsalo si quieres empezar de cero, y entonces aplica las migraciones de nuevo:

```bash
docker compose down -v && docker compose up -d   # vuelve a initdb
make db-status
```

#### `JWT_SECRET` es obligatorio

El compose lo declara así:

```yaml
JWT_SECRET: ${JWT_SECRET:?JWT_SECRET requerido (≥32 bytes)...}
```

La sintaxis `:?` hace que `docker compose up` **falle de inmediato** si la
variable no está definida. Está en `.env`; si falta:

```bash
echo "JWT_SECRET=$(openssl rand -base64 32)" >> .env
```

#### Nota sobre lockfiles

#### Gestor de paquetes: pnpm

El lockfile canónico es **`pnpm-lock.yaml`**. Ya no existe `package-lock.json`
en el frontend, y no debe volver: con ambos presentes el siguiente
`npm install` regenera el lock desde cero e instala versiones que nadie
probó.

```
pnpm install --frozen-lockfile   # es lo que hace `make install`
```

`--frozen-lockfile` falla si el lock no cuadra con `package.json`, que es lo
deseable en CI. El `Makefile` del frontend usa `NPM ?= pnpm` como variable, así
que todo target (`make lint`, `make test`, `make build`, `make audit`) corre con
pnpm.

**Vulnerabilidades transitivas.** Los overrides viven en
`frontend/pnpm-workspace.yaml`, no en el campo `pnpm` de `package.json`: pnpm 11
ignora ese campo en silencio (solo avisa por stderr). Hoy fuerza
`source-map-js@1.2.2`, que llega transitiva por `@tailwindcss/node` y por
`css-tree → jsdom` sin versión parcheada propia. `make audit` falla el build si
vuelve a aparecer cualquier advisory high o critical.

Consecuencia práctica: una dependencia transitiva no es visible para el bundler.
Por eso `workbox-window` está declarada como devDependency **explícita** en
`package.json` aunque `vite-plugin-pwa` ya la traiga:

```
[vite]: Rolldown failed to resolve import "workbox-window"
```

Usa **pnpm** de forma consistente. Mezclar gestores reescribe el árbol y rompe
`node_modules`.

### Producción (Multi-Stage Build)

**Backend Dockerfile** — el real, en `backend/Dockerfile`:
```dockerfile
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o server .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/.env.example ./.env.example
COPY --from=builder /app/server .
EXPOSE 3000
CMD ["./server"]
```

> El binario se llama **`server`**, no `app`. Es lo que espera el `CMD`, y lo
> que produce `make build`. Un `CMD ["./app"]` arrancaría un binario inexistente
> y el contenedor saldría con exit 1.

**Build y push:**
```bash
docker build -t tu-registry/multicatalogo-backend:1.0 -f backend/Dockerfile backend/
docker push tu-registry/multicatalogo-backend:1.0
```

**Frontend Dockerfile** — *no existe en el repositorio*. Si lo añades:
```dockerfile
FROM node:22-alpine AS builder   # Vite 8 exige Node ^20.19 ó >=22.12

# pnpm no viene en la imagen de node: hay que habilitarlo.
RUN corepack enable && corepack prepare pnpm@11.24.0 --activate

WORKDIR /app
# pnpm-workspace.yaml NO es opcional: lleva los `overrides` de seguridad.
# Copiarlo sin él deja la imagen con las versiones sin parchear.
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile

COPY . .
RUN pnpm run build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/nginx.conf

EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
```

---

## CI/CD Pipeline

### GitHub Actions (CI)

**Archivo:** `.github/workflows/ci.yml` (ya existe)

Ejecuta automáticamente en cada push:
- Lint (frontend + backend)
- Tests (backend)
- Build (frontend + backend)

**Ver resultados:** GitHub → Actions

### Desplegar Automáticamente (CD)

**Opción: Railway auto-deploy desde GitHub**
```yaml
# En railway.json (root)
{
  "build": {
    "builder": "dockerfile"
  }
}
```

Railway detecta cambios en main → auto-deploya

---

## Base de Datos

### Backups

**Local:**
```bash
# Dump de BD
docker compose exec -T postgres pg_dump -U admin multicatalogo > backup-$(date +%Y%m%d).sql

# Restaurar
docker compose exec -T postgres psql -U admin multicatalogo < backup-20261023.sql
```

> La `-T` desactiva la TTY. Sin ella, `pg_dump` falla con
> `the input device is not a TTY` al redirigir la salida a un archivo.

**Producción:**
Railway y Render tienen backups automáticos. Ver en Dashboards.

### Migraciones

Migraciones son **idempotentes** (aplican sin error si ya existen).

```bash
make db-migrate     # aplica todas en orden alfabético
make db-status      # tablas + conteos
```

O sobre el volumen existente, a mano:

```bash
docker compose exec -T postgres psql -U admin -d multicatalogo \
  -f /docker-entrypoint-initdb.d/004_direccion_entrega.sql
```

### Inspección de la base

```bash
# PSQL dentro del contenedor
docker compose exec postgres psql -U admin -d multicatalogo

# Desde el host, con la app instalada
psql "postgres://admin:secret@localhost:5433/multicatalogo"

# Atajo
make db-psql
```

**Cliente gráfico:** DBeaver o pgApp → host `localhost`, puerto **5433**, base
`multicatalogo`, usuario `admin`, contraseña `secret`.

---

## Troubleshooting

### Frontend

#### "Cannot find module '@/...'"
**Solución:** Verificar tsconfig.json
```json
{
  "compilerOptions": {
    "baseUrl": "./src",
    "paths": {
      "@/*": ["./*"]
    }
  }
}
```

#### "VITE_API_URL no definida"
```bash
# Crear .env
VITE_API_URL=http://localhost:3000
```

#### Port 5173 ya en uso
```bash
pnpm run dev -- --port 5174
```

### Backend

#### "database connection refused"
```bash
# Verificar PostgreSQL
docker compose ps postgres
# Si stopped, reiniciar:
docker compose up -d postgres

# Esperar healthcheck
sleep 10
```

#### "redis: connection refused"
```bash
# Verificar Redis
docker compose up -d redis

# Test
docker compose exec redis redis-cli PING
```

#### "JWT_SECRET no configurado"
```bash
# Generar
openssl rand -base64 32

# Copiar a .env
JWT_SECRET=<output>
```

#### Port 3000 ya en uso
```bash
PORT=3001 go run .
```

### Database

#### "no such table"
```bash
# ¿Es el primer arranque? Las migraciones se aplican solas al crear el volumen.
docker compose exec postgres psql -U admin -d multicatalogo -c "\dt"

# ¿Volumen ya existente? Reaplicar a mano.
make db-migrate

# ¿Empezar de cero? (⚠ borra los datos)
make db-reset
```

#### "relation pedidos does not exist" pero la tabla está
El backend corre en Docker y espera `postgres:5432`; desde el host el puerto es
**5433**. Si cambiaste una de las dos rutas, los dos extremos apuntan a sitios
distintos.

#### "too many connections"
```bash
# Ver conexiones
docker compose exec postgres psql -U admin -c "SELECT count(*) FROM pg_stat_activity;"

# Matar conexiones (cuidado)
docker compose down
docker compose up -d postgres
```

### Docker

#### "port already in use"
```bash
# Cambiar puerto en docker-compose.yml
# "5433:5432" → "5434:5432"

docker compose up -d
```

#### "no space left on device"
```bash
# Limpiar volumes no usados
docker volume prune

# O resetear todo
docker compose down -v
docker compose up -d
```

---

## Checklist de Despliegue

### Pre-Despliegue
- [ ] `git status` limpio (sin cambios sin commitear)
- [ ] `make typecheck` sin errores (frontend)
- [ ] `make build` sin errores (frontend)
- [ ] `make -C backend vet` sin errores (backend)
- [ ] `make -C backend build` sin errores (backend)
- [ ] `make test` pasa — **69 tests** (frontend)
- [ ] `make db-test` antes de `make test-backend` (si no, los tests de
      integración hacen `t.Skip` y pasan sin probar SQL)
- [ ] `make test-backend` pasa — **295 tests, 0 skip** (backend)
- [ ] `make -C backend coverage` — cobertura >70% en `controllers` y `domain`
- [ ] `make -C backend test-race` sin carreras de datos
- [ ] `make test-backend` es idempotente: correrlo 2 veces seguidas da el
      mismo resultado (los residuos entre corridas se limpian solos)
- [ ] `.env` configurado con valores de producción
- [ ] `JWT_SECRET` generado con openssl (>= 32 bytes)
- [ ] `DATABASE_URL` apunta a BD producción
- [ ] `REDIS_URL` apunta a Redis producción
- [ ] `ALLOWED_ORIGINS` contiene frontend URL

### Despliegue
- [ ] Backup de BD actual
- [ ] Aplicar migraciones (si hay nuevas)
- [ ] Deploy backend
- [ ] Deploy frontend
- [ ] Verificar `/api/health` retorna 200
- [ ] Test login funciona
- [ ] Test checkout funciona
- [ ] Verificar logs sin errores

### Post-Despliegue
- [ ] Monitorear logs (24h primeras)
- [ ] Verificar BD creciendo (usuarios, pedidos)
- [ ] Verificar Redis cacheando
- [ ] Configurar alerts (si disponible)
- [ ] Documentar URL producción

---

## Monitoreo en Producción

### Logs
```bash
# Railway
railway logs

# Render
# Ver en Dashboard → Logs

# Docker (VPS)
docker compose logs -f backend
```

### Métricas (opcional)
```bash
# Ver uso de recursos
docker stats
```

### Alertas (opcional)
- Railway: Integración con Slack
- Render: Webhooks personalizados
- VPS: Configurar con Prometheus/Grafana

---

## URLs Útiles

| Servicio | Desarrollo | Producción |
|----------|-----------|-----------|
| Frontend (Vite) | http://localhost:5173 | — |
| Frontend (preview) | http://localhost:4173 | https://tu-dominio.com |
| Backend | http://localhost:3000 | https://tu-dominio.com |
| PostgreSQL | `localhost:5433` | Managed (Railway/Render) |
| Redis | `localhost:6379` | Managed (Railway/Render) |

> El puerto de PostgreSQL es **5433**, no 5432: el compose lo mapea así para no
> chocar con un Postgres del sistema. `DATABASE_URL` dentro de Docker apunta a
> `postgres:5432` (la red interna); desde el host, a `localhost:5433`.

---

## Preguntas Frecuentes

**P: ¿Cuál es el costo?**  
A: Railway/Render gratuito para estudiantes. Después de graduation, paga según uso.

**P: ¿Puedo usar Heroku?**  
A: Heroku ya no tiene plan free. Usa Railway/Render en su lugar.

**P: ¿Necesito HTTPS?**  
A: Recomendado en producción. Railway/Render lo incluyen automáticamente.

**P: ¿Cómo escalarlo?**  
A: Futuro (U3+). Por ahora, single instance es suficiente.

**P: ¿Dónde guardo archivos (imágenes)?**  
A: `img` campo es URL. Usar CDN (Cloudinary, AWS S3) en futuro.

---

**Versión:** 1.0  
**Última actualización:** 2026-10-23  
**Responsable:** Framework Programación Web — UPSE

