# Optional Improvements — Estado Actual

**Fecha:** 2026-10-23  
**Estado General:** ✅ I1 Completado | ❌ I2-I4 No aplican (auditados contra código real)

---

## Resumen Ejecutivo

De las 4 mejoras opcionales propuestas, **I1 (Frontend Tests) está 100% completado**. Las otras 3 no aplican porque:

- **I2** (Race Condition): El código actual usa transacción SQL atómica — no hay carrera
- **I3** (Dashboard Charts): Recharts añadiría 500 KB sin beneficio funcional (ya hay CSS bars)
- **I4** (Paginación): API ya está estandarizada en `{data, page, limit, total}`

### Métricas

| Item | Estado | Esfuerzo Usado | Resultado |
|------|--------|--------|-----------|
| **I1: Frontend Tests** | ✅ Completado | 4-6h | 69 tests ✓ · 54.6% coverage · 6 archivos |
| **I2: Race Condition** | ❌ No aplica | 0h | SQL es atómico · No hay carrera |
| **I3: Charts** | ⚠️ Decisión | 0h | Deliberadamente excluido · Código CSS existente |
| **I4: Paginación** | ❌ No aplica | 0h | Formato ya consistente en API |

---

## I1 — Frontend Tests ✅ COMPLETADO

**Tests Implementados:** 6 archivos, 69 tests, 54.6% cobertura

### Desglose por componente

| Archivo | Tests | Qué valida |
|---------|-------|-----------|
| `src/services/http.test.ts` | 17 | Token Bearer, ApiError, 204 sin body, FormData, body no pre-serializado |
| `src/components/Checkout.test.tsx` | 11 | Campos obligatorios, payload, ausencia inteligente de email, bloqueo en vuelo |
| `src/context/CartContext.test.tsx` | 10 | Acumulación de cantidad, totales, eliminación selectiva, persistencia |
| `src/context/AuthContext.test.tsx` | 11 | Persistencia de sesión, restore desde localStorage, logout, rol manipulado |
| `src/components/reportes/format.test.ts` | 13 | toFixed correcto, separadores, fechas inválidas, barras ancho (anchoBarra) |
| `src/hooks/useReportes.test.ts` | 7 | Guard de carreras en reload, deps correctas, stale closure |

### Bugs encontrados y corregidos

**Bug 1:** `anchoBarra(10, NaN)` devolvía `NaN`
- Línea: `src/components/reportes/format.ts`
- Causa: `NaN <= 0` es `false` en JavaScript
- Síntoma: `style="width: NaN%"` sin error de consola, barra desaparece silenciosamente
- Corregido: Guard actualizado para `isNaN(maximo)`

**Bug 2:** `<label>` sin `htmlFor` en Checkout
- Línea: `src/components/Checkout.tsx` (5 campos)
- Síntoma: Lector de pantalla no anuncia campos; click en label no enfoca input
- Corregido: Todos `<label htmlFor>` asociados a sus inputs

### Tecnología usada

**Vitest en lugar de Jest** — Decisión correcta:
- Jest + ts-jest crearían pipeline duplicada paralela a Vite
- Vitest reutiliza `vite.config.ts` → single source of truth
- Resolución de módulos consistente con build final

### Verificación

```bash
cd frontend
npm test              # ✅ 69 passed, 6 files
npm run test:coverage # ✅ 54.58% statements (threshold: 50%)
npm run build         # ✅ Sin errores TypeScript
```

---

## I2 — Race Condition ❌ NO APLICA

**Análisis:** La presunta carrera no existe.

### Código inspeccionado

Función: `backend/repository/repository.go:1154-1168`
```go
// ActualizarVentasUpline — SQL atómico, no bucle en app
func (r *PostgresRepository) ActualizarVentasUpline(ctx context.Context, uid int, total decimal.Decimal) error {
    upline, err := r.getCadenaUpline(ctx, uid, 3)
    if err != nil { return err }
    
    // Una sola sentencia SQL (no bucle):
    _, err = r.db.ExecContext(ctx,
        `WITH RECURSIVE upline AS (...) 
         UPDATE referidos SET ventas = ventas + $2 WHERE id IN (SELECT id FROM upline)`,
        total, uid,
    )
    return err
}
```

### Por qué NO hay carrera

`SET ventas = ventas + $2` ejecutado por PostgreSQL es **atómico**:
- PostgreSQL bloquea la fila durante UPDATE (row-level lock)
- Dos checkouts concurrentes se **serializan** sobre ese lock automáticamente
- No hay *lost update* posible

**Regla:** Un lost update exige:
1. Leer (`r1 = SELECT`)
2. Decidir (`r1 + 50`)
3. Escribir (`UPDATE ... WHERE`)

ejecutado desde la **aplicación** en 3 pasos. `x = x + 1` ejecutado por BD es atómico por definición.

### Lo que SÍ es cierto (pero no es carrera)

`ActualizarVentasUpline` corre en goroutine **post-COMMIT**:
```
COMMIT
  ↓
goroutine procesarComisiones() inicia
  ↓ (desfase si proceso muere aquí)
UPDATE referidos.ventas
```

Si el proceso muere entre ambas líneas, `referidos.ventas` queda desfasado. Pero:
- No corrompe dinero: `comisiones` tabla (source of truth) sí está dentro de tx
- No es una carrera: es una ventana de pérdida post-COMMIT
- Arreglo (si aplica): mover UPDATE **dentro** de tx del checkout

**Conclusión:** No implementar. Código es seguro.

---

## I3 — Dashboard Charts ⚠️ DECISIÓN, NO GAP

**Propuesta inicial:** Integrar recharts para gráficos en TopProductos / TopReferidos

**Decisión:** Deliberadamente excluido. Razones:

### 1. TopProductos y TopReferidos YA tienen barras visuales

Código actual:
```tsx
// src/components/reportes/TopProductos.tsx
{data.map(p => (
  <div className="flex items-center gap-4">
    <span>{p.nombre}</span>
    <div className="bg-blue-100 h-6" 
         style={{ width: `${(p.unidades / max) * 100}%` }} />
    <span>{p.unidades}</span>
  </div>
))}
```

Ya renderiza ranking como barras CSS. Recharts es **redundante**.

### 2. Costo sin beneficio

- Recharts: ~500 KB de JS
- Para: dos listas ordenadas (ya mostradas)
- Ganancia: visual marginal (barras proporcionales ya existen)

### 3. El código propuesto rompería el dashboard

Errores si se ejecutara:
```
| Propuesta en I3 | Realidad en código |
|---|---|
| dataKey="total_vendido" | Campo está en ReporteResumen, NO en TopProducto → vacío |
| dataKey="cantidad_ordenes" | Campo NO existe en ningún tipo → undefined |
| import { listTopProductos } | Función real: getTopProductos() |
| @/components/TopProductos | Viven en src/components/reportes/ |
| @/ alias | NO existe en tsconfig.app.json |
```

**Conclusión:** No implementar. Decisión correcta: mantener solución ligera.

---

## I4 — Paginación Response Standardization ❌ NO APLICA

**Propuesta:** Estandarizar respuesta paginada a `{data, page, limit, total}`

**Análisis:** Ya está estandarizada.

### Endpoints verificados

```bash
# Comisiones
curl -H "Authorization: Bearer TOKEN" \
  http://localhost:3000/api/comisiones/mias?page=1&limit=10
# ✅ Respuesta: {data: [...], page: 1, limit: 10, total: 42, totales: {...}}

# Liquidaciones
curl -H "Authorization: Bearer TOKEN" \
  http://localhost:3000/api/admin/liquidaciones?page=1&limit=10
# ✅ Respuesta: {data: [...], page: 1, limit: 10, total: 15}

# Productos
GET /api/admin/productos?page=1&limit=10
# ✅ Respuesta: {data: [...], page: 1, limit: 10, total: 4}
```

### Única inconsistencia deliberada

Los **4 endpoints de reportes** devuelven array directo (sin paginación):
```
GET /api/reportes/resumen → {...} (singular, no array)
GET /api/reportes/top-productos → [{...}, ...] (array, no paginado)
GET /api/reportes/top-referidos → [{...}, ...] (array, no paginado)
GET /api/reportes/comisiones-pendientes → [{...}, ...] (array, no paginado)
```

**Por qué:** No tienen paginación en el contrato OpenAPI (solo `?limit`, sin `page`)

Forzarles `{data, page, limit, total}` obligaría frontend a desempaquetar `page` y `total` que no existen. Decisión correcta.

**Conclusión:** No hay inconsistencia que arreglar. Lo que falta es documentación clara (ya en OpenAPI).

---

## Infraestructura — Nota importante

**Lockfiles:** El proyecto tiene `package-lock.json` + `pnpm-lock.yaml`

Esto significa `node_modules` es store de pnpm (resolución estricta, sin hoisting).

**Impacto anterior:** `workbox-window` no era visible → build fallaba  
**Solución:** Agregada como devDependency explícita en `package.json`

**Recomendación futura:** Elegir **uno** como canónico:
- Opción A: npm (simplificar, un lockfile)
- Opción B: pnpm (mantener ambos, workflow flexible)

Actualmente ambos están versionados → próximo `npm install` respeta el árbol.

---

## Verificación Final

```bash
cd frontend

# Tests
npm test                # ✅ 69 passed (6 files)
npm run test:coverage  # ✅ 54.58% statements

# Build
npm run build          # ✅ Sin errores
npm run preview        # ✅ PWA preview en 4173

# Backend
cd ../backend
go build ./...         # ✅ Sin errores
go test ./...          # ✅ Todos pasan
go test -race ./...    # ✅ Sin data races
```

---

## Conclusión

| Mejora | Implementación | Esfuerzo | Razón |
|--------|---|---|---|
| **I1: Frontend Tests** | ✅ Hecho | 4-6h usadas | 69 tests, 54.6% cov, 2 bugs encontrados |
| **I2: Race Condition** | ❌ No | 0h | No existe en código (SQL atómico) |
| **I3: Dashboard Charts** | ❌ No | 0h | Recharts añadiría 500 KB (ya hay CSS bars) |
| **I4: Paginación** | ❌ No | 0h | API ya está estandarizada |

**Proyecto completamente funcional. No hay gaps críticos restantes.**

---

**Responsable:** Auditoría técnica  
**Versión:** 2.0 — Estado contra código real  
**Próximo paso:** Despliegue en producción (ver `docs/DEPLOYMENT.md`)
