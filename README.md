# Framework Programación Web

Repositorio de estudio para la materia **Framework Programación Web** del 7º semestre de la carrera de Ingeniería en Sistemas.

**Institución**: Universidad Estatal Península de Santa Elena (UPSE)

---

## 📁 Estructura del Repositorio

```
Programaci-n-Web/
├── Talleres/                    # Ejercicios prácticos semanales
│   ├── Practica02_React_Bases/  # Fundamentos de React
│   ├── Practica03_React_Bases/  # React avanzado
│   └── Practica04_frontend/     # Frontend con React + Vite
│       └── src/
│           ├── components/      # Componentes React
│           ├── context/         # Context API (Auth, Cart, Sidebar)
│           └── services/        # Servicios API
│   └── Practica04-backend/      # Backend con Go + Fiber
│       ├── controllers/         # Lógica de negocio
│       ├── models/              # Estructuras de datos
│       └── routes/              # Endpoints REST
├── Tareas/                      # Actividades individuales
├── Lecciones/                   # Material teórico
└── README.md                    # Este archivo
```

---

## 🛠️ Tecnologías

### Frontend
- **React 18** con TypeScript
- **Vite** (bundler)
- **React Router** (navegación)
- **Tailwind CSS** (estilos)
- **Context API** (gestión de estado)

### Backend
- **Go** (lenguaje)
- **Fiber v2** (framework web)
- **CORS** (middleware)

---

## 🚀 Inicio Rápido

### Backend
```bash
cd Talleres/Practica04-backend
go run main.go
# Escucha en http://localhost:3000
```

### Frontend
```bash
cd Talleres/Practica04_frontend
npm install
npm run dev
# Abre http://localhost:5173
```

---

## 📋 Tareas Completadas

### Practica04 - Conexión Frontend-Backend ✓
- [x] Crear servicio API (fetch wrapper)
- [x] Implementar autenticación con token JWT
- [x] Conectar Login a POST `/api/login`
- [x] Conectar Catálogo a GET `/api/productos`
- [x] Guardar token en localStorage
- [x] Configurar CORS en backend

**Credenciales de prueba:**
- Email: `admin@upse.edu.ec`
- Contraseña: `123456`

---

## 📚 Aprendizajes Clave

1. **Integración Frontend-Backend**: Comunicación entre React y Go mediante API REST
2. **Autenticación**: Uso de tokens JWT y almacenamiento en localStorage
3. **CORS**: Configuración segura de solicitudes cross-origin
4. **Context API**: Gestión global de estado (autenticación, carrito)
5. **Manejo de async/await**: Operaciones asincrónicas en React

---

## ⚙️ Configuración

### Variables de Entorno (Frontend)
En `src/services/api.ts` está configurada la URL del backend:
```typescript
const API_URL = "http://localhost:3000";
```

Para producción, cambiar a la URL del servidor.

---

## 📝 Notas Importantes

- El backend requiere Go 1.18+
- El frontend requiere Node.js 16+
- Los tokens se guardan en `localStorage` (no seguro para producción)
- El CORS está configurado para `localhost:5173` (desarrollo)

---

## 📧 Autor

**Estudiante**: Andy Alejandro  
**Carrera**: Ingeniería en Sistemas  
**Semestre**: 7º  
**Universidad**: UPSE

---

*Último actualizado: Septiembre 2026*
