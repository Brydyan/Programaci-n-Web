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
| **Node.js** | 18+ | Frontend (React + Vite) |
| **npm** | 9+ | Package manager JS |
| **Go** | 1.21+ | Backend (Fiber) |
| **Docker** | 20.10+ | Contenedores DB + Redis |
| **Docker Compose** | 2.0+ | Orquestación local |
| **Git** | 2.30+ | Control de versiones |
| **PostgreSQL CLI** | 14+ | Gestión base de datos (opcional) |

### Verificar Instalaciones
```bash
node --version        # v18.0.0+
npm --version        # 9.0.0+
go version          # go1.21+
docker --version    # Docker 20.10+
docker-compose --version  # Docker Compose 2.0+
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
# Desde raíz de Practica07
docker-compose up -d postgres redis

# Esperar a que estén healthy (~5-10 segundos)
docker-compose ps
# STATUS debe ser "healthy" para ambos
```

**Verificar conectividad:**
```bash
# PostgreSQL
docker-compose exec postgres psql -U admin -d multicatalogo -c "SELECT 1;"
# Output: 1

# Redis
docker-compose exec redis redis-cli PING
# Output: PONG
```

### 3. Aplicar Migraciones

```bash
# Opción A: Automático (primera vez)
# Las migraciones se aplican automáticamente al iniciar postgres
docker-compose restart postgres
sleep 5

# Opción B: Manual
docker-compose exec postgres psql -U admin -d multicatalogo < database/migrations/001_schema_base.sql
docker-compose exec postgres psql -U admin -d multicatalogo < database/migrations/002_backoffice.sql
docker-compose exec postgres psql -U admin -d multicatalogo < database/migrations/003_seed.sql

# Verificar
docker-compose exec postgres psql -U admin -d multicatalogo -c "SELECT COUNT(*) FROM usuarios;"
# Output: 3 (seed data)
```

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
go mod tidy

# Compilar
go build -o app .

# Ejecutar
./app
# O directamente:
go run .

# Output esperado:
# [11:42:32] Fiber v2.x.x
# [11:42:32] 🚀  http://localhost:3000
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
npm install

# Desarrollo (hot reload)
npm run dev
# Output: VITE v4.x.x ready in XXX ms
# ➜  Local:   http://localhost:5173/

# O build para producción
npm run build  # Genera dist/
npm run preview  # Servir build localmente
```

### 6. Acceder a la Aplicación

```
Frontend:  http://localhost:5173
Backend:   http://localhost:3000
Adminer:   http://localhost:8080  (manage DB visually)
```

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
# 2. Environment: Go 1.21
# 3. Build: cd backend && go build -o app
# 4. Start: ./app
# 5. Anotar PORT=3000

# Frontend (React)
# 1. New → GitHub Repo → Practica07
# 2. Environment: Node 18
# 3. Build: cd frontend && npm install && npm run build
# 4. Start: npx serve -s dist -l 3000
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
# 3. Build: npm run build
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
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d

# 5. Verificar
docker-compose ps
curl http://localhost:3000/api/health
```

---

## Docker & Containers

### Local Development (docker-compose.yml)

El archivo ya existe. Levanta:
- **postgres:16** (puerto 5433)
- **redis:7** (puerto 6379)
- **adminer** (puerto 8080, opcional)

```bash
# Ver servicios
docker-compose ps

# Ver logs
docker-compose logs -f postgres  # Logs PostgreSQL
docker-compose logs -f redis     # Logs Redis

# Detener
docker-compose down

# Resetear (eliminar volúmenes)
docker-compose down -v
```

### Producción (Multi-Stage Build)

**Backend Dockerfile:**
```dockerfile
# build.Dockerfile (backend/)
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.* .
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o app .

FROM alpine:3.18
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/app .

EXPOSE 3000
CMD ["./app"]
```

**Build y push:**
```bash
docker build -t tu-registry/multicatalogo-backend:1.0 -f backend/Dockerfile backend/
docker push tu-registry/multicatalogo-backend:1.0
```

**Frontend Dockerfile:**
```dockerfile
# Dockerfile (frontend/)
FROM node:18-alpine AS builder

WORKDIR /app
COPY package*.json .
RUN npm install

COPY . .
RUN npm run build

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
docker-compose exec postgres pg_dump -U admin multicatalogo > backup-$(date +%Y%m%d).sql

# Restaurar
docker-compose exec postgres psql -U admin multicatalogo < backup-20261023.sql
```

**Producción:**
Railway y Render tienen backups automáticos. Ver en Dashboards.

### Migraciones

Migraciones son **idempotentes** (aplican sin error si ya existen).

```bash
# Aplicar manualmente (si es necesario)
docker-compose exec postgres psql -U admin multicatalogo -f /migrations/001_schema_base.sql
```

### Monitorear

**Adminer (UI web para BD):**
```
http://localhost:8080
Server: postgres
User: admin
Password: secret
Database: multicatalogo
```

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
npm run dev -- --port 5174
```

### Backend

#### "database connection refused"
```bash
# Verificar PostgreSQL
docker-compose ps postgres
# Si stopped, reiniciar:
docker-compose up -d postgres

# Esperar healthcheck
sleep 10
```

#### "redis: connection refused"
```bash
# Verificar Redis
docker-compose up -d redis

# Test
docker-compose exec redis redis-cli PING
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
# Aplicar migraciones
docker-compose exec postgres psql -U admin multicatalogo < database/migrations/001_schema_base.sql
```

#### "too many connections"
```bash
# Ver conexiones
docker-compose exec postgres psql -U admin -c "SELECT count(*) FROM pg_stat_activity;"

# Matar conexiones (cuidado)
docker-compose down
docker-compose up -d postgres
```

### Docker

#### "port already in use"
```bash
# Cambiar puerto en docker-compose.yml
# "5433:5432" → "5434:5432"

docker-compose up -d
```

#### "no space left on device"
```bash
# Limpiar volumes no usados
docker volume prune

# O resetear todo
docker-compose down -v
docker-compose up -d
```

---

## Checklist de Despliegue

### Pre-Despliegue
- [ ] `git status` limpio (sin cambios sin commitear)
- [ ] `npm run build` sin errores (frontend)
- [ ] `go build ./...` sin errores (backend)
- [ ] `npm run test` pasa (si existe)
- [ ] `go test ./...` pasa (si existe)
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
docker-compose logs -f backend
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
| Frontend | http://localhost:5173 | https://tu-dominio.com |
| Backend | http://localhost:3000 | https://tu-dominio.com/api |
| PostgreSQL | localhost:5433 | Managed (Railway/Render) |
| Redis | localhost:6379 | Managed (Railway/Render) |
| Adminer | http://localhost:8080 | N/A (local only) |

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

