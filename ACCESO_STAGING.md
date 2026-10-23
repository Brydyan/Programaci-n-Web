# Acceso a Plataforma Staging - MultiCatálogo

## 🌐 URL Pública

```
https://staging-andy.tailc8cd10.ts.net/
```

**Nota:** Requiere **Tailscale** instalado y conectado al tailnet de la UPSE.

---

## 👥 Credenciales de Usuarios

### 1. Administrador
- **Email:** `admin@upse.edu.ec`
- **Contraseña:** `12345678`
- **Rol:** Admin
- **Acceso:** Panel de administración, gestión de productos, reportes, auditoría de comisiones

### 2. Cliente Demo
- **Email:** `cliente@upse.edu.ec`
- **Contraseña:** `12345678`
- **Rol:** Cliente
- **Acceso:** Catálogo de productos, compras, red de referidos, comisiones

---

## 🚀 Primeros Pasos

1. **Instala Tailscale** en tu dispositivo (Mac, Windows, Linux, iOS, Android)
2. **Conéctate** al tailnet de la UPSE
3. **Accede** a la URL: `https://staging-andy.tailc8cd10.ts.net/`
4. **Inicia sesión** con cualquiera de las credenciales anteriores

---

## 🏗️ Arquitectura

| Componente | URL | Puerto |
|-----------|-----|--------|
| Frontend | https://staging-andy.tailc8cd10.ts.net/ | 5173 |
| Backend API | https://staging-andy.tailc8cd10.ts.net/:3000 | 3000 |
| Health Check | https://staging-andy.tailc8cd10.ts.net/:3000/api/health | - |

---

## 🔄 Despliegue Automático

- **Rama:** `develop`
- **Trigger:** Push a `origin/develop`
- **Servidor:** VM 109 (staging-andy)
- **Script:** `/opt/multicatalogo/deploy.sh`

Los cambios se despliegan automáticamente a través de GitHub Actions.

---

## 📝 Notas de Desarrollo

- **Base de datos:** PostgreSQL 16 (puerto 5433 local)
- **Caché:** Redis (puerto 6379 local)
- **Contraseña seed:** La contraseña `12345678` es **solo para desarrollo**
- **Archivo seed:** `Talleres/Practica07/database/migrations/003_seed.sql`

---

## ⚙️ Troubleshooting

### No puedo acceder desde mi dispositivo
1. Verifica que **Tailscale esté activo** en tu dispositivo
2. Verifica que estés en el **mismo tailnet** que la VM
3. Intenta con `ping staging-andy.tailc8cd10.ts.net` desde terminal

### El backend responde 404
1. Verifica que los contenedores están corriendo: `docker compose ps`
2. Revisa logs: `docker compose logs backend | tail -20`

### El frontend no carga las rutas (SPA)
1. Verifica que `serve` está corriendo: `docker compose logs frontend`
2. Reconstruye: `docker compose down -v && docker compose --profile staging up -d --build`

---

**Último actualizado:** 2026-10-23  
**Ambiente:** Staging (Tailscale Funnel)  
**Responsable:** Andy Alejandro
