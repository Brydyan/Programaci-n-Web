<!-- Context directory index — Navigate all documentation -->

# .context/ Directory Index

**Última actualización:** 2026-10-23  
**Estado:** Organizado en 3 categorías

---

## 🗂️ Estructura

```
.context/
├── INDEX.md                          (Este archivo)
├── README.md                         (Este directorio — guía rápida)
│
├── implementado/                     (✅ YA HECHO)
│   ├── ANALISIS_FUNCIONAL_U2.md     (Especificación funcional)
│   ├── AUDIT_IMPLEMENTATION.md      (Estado actual U2)
│   ├── IMPLEMENTATION_GAPS_4ITEMS.md (4 gaps ya implementados)
│   ├── PLAN_IMPLEMENTACION_CORREGIDO.md (Bug report histórico)
│   ├── ROUTES_AUDIT.md              (Auditoría 24 endpoints)
│   ├── SPRINT_S1.md                 (Decisiones seguridad)
│   ├── STATUS_DOCUMENTS.md          (Matriz de aplicación)
│   ├── VERIFICATION_CHECKLIST.md    (Para verificar)
│   └── README.md                    (Guía de navegación)
│
├── porImplementar/                  (⚠️ FUTURO)
│   ├── KNOWN_GAPS_AND_LIMITATIONS.md (Todos los gaps + mejoras)
│   ├── OPTIONAL_IMPROVEMENTS_5-8H.md (Propuesta inicial — 4 mejoras detalladas)
│   ├── OPTIONAL_IMPROVEMENTS_STATUS.md (✅ Estado actual contra código real)
│   └── PWA_PLAN.md                  (Roadmap PWA, auditado y corregido)
│
└── deprecado/                       (❌ NO USAR)
    ├── _DEPRECATED_BUILDER_GUIDE.md.bak.bak (Viejo)
    └── TECHNICAL_MAP.md             (Contiene errores)
```

---

## 📚 Cómo Navegar

### "Soy nuevo en el proyecto, ¿dónde empiezo?"

1. **Leer especificación:** `implementado/ANALISIS_FUNCIONAL_U2.md` (15 min)
2. **Ver estado actual:** `implementado/AUDIT_IMPLEMENTATION.md` (10 min)
3. **Entender rutas:** `implementado/ROUTES_AUDIT.md` (5 min)

**Total:** 30 minutos para context completo.

---

### "¿Qué falta por implementar?"

→ **`porImplementar/KNOWN_GAPS_AND_LIMITATIONS.md`** (LEER PRIMERO)

Contiene:
- 5 gaps parciales documentados
- Referencias a dónde están las instrucciones
- Matriz de impacto/esfuerzo
- Recomendación de qué hacer ahora vs futuro

**Estado actual de mejoras opcionales:**

→ **`porImplementar/OPTIONAL_IMPROVEMENTS_STATUS.md`** (VERIFICACIÓN CONTRA CÓDIGO REAL)

Resultado:
- ✅ I1: Frontend Tests — COMPLETADO (69 tests, 54.6% coverage)
- ❌ I2: Race Condition — NO APLICA (SQL es atómico)
- ⚠️ I3: Dashboard Charts — DECISIÓN (excluido deliberadamente)
- ❌ I4: Paginación — NO APLICA (API ya estandarizada)

Si necesitas **instrucciones detalladas originales:**

- **Mejoras propuesta inicial:** `porImplementar/OPTIONAL_IMPROVEMENTS_5-8H.md`
- **PWA post-U2:** `porImplementar/PWA_PLAN.md`

---

### "¿Cómo verifico que algo está hecho?"

→ `implementado/VERIFICATION_CHECKLIST.md`

O para los 4 gaps específicos:

→ `implementado/IMPLEMENTATION_GAPS_4ITEMS.md` (sección final: resultado)

---

### "¿Cuáles son las decisiones de seguridad?"

→ `implementado/SPRINT_S1.md`

Contiene:
- Middleware aplicado
- JWT implementation
- Rate limiting
- Variables de entorno

---

### "¿Cuál fue el proceso que se siguió?"

→ `implementado/STATUS_DOCUMENTS.md`

Matriz que muestra:
- Qué documentos existen
- Qué está aplicado vs pendiente
- Qué tiene errores (TECHNICAL_MAP.md)

---

## ✅ Implementado (Ya Aplicado al Código)

| Archivo | Propósito | Usar para |
|---------|----------|-----------|
| **ANALISIS_FUNCIONAL_U2.md** | Especificación funcional original | Entender requisitos |
| **AUDIT_IMPLEMENTATION.md** | Estado real del código (95% completo) | Verificar qué existe |
| **IMPLEMENTATION_GAPS_4ITEMS.md** | 4 gaps ya implementados | Histórico/referencia |
| **PLAN_IMPLEMENTACION_CORREGIDO.md** | Bug report de documentos viejos | Entender qué salió mal |
| **ROUTES_AUDIT.md** | Auditoría completa de 24 endpoints | Verificar seguridad |
| **SPRINT_S1.md** | Decisiones seguridad + middleware | Configuración producción |
| **STATUS_DOCUMENTS.md** | Matriz de qué documentos qué hacen | Meta-documentación |
| **VERIFICATION_CHECKLIST.md** | Pasos para verificar cada task | Testing manual |
| **IMPLEMENTATION_GAPS_4ITEMS.md** | 4 gaps + veredicto de ejecución | Verificar antes de implementar |

**Estos archivos son referenciales** — ya no cambian.

---

## ⚠️ Por Implementar (Futuro)

| Archivo | Propósito | Esfuerzo | Timeline |
|---------|----------|----------|----------|
| **KNOWN_GAPS_AND_LIMITATIONS.md** | Todos los gaps documentados | — | Ya existe |
| **OPTIONAL_IMPROVEMENTS_5-8H.md** | Instrucciones iniciales (4 mejoras detalladas) | 5-8h | Referencia histórica |
| **OPTIONAL_IMPROVEMENTS_STATUS.md** | ✅ Estado actual contra código real | — | VERIFICADO |
| **PWA_PLAN.md** | Roadmap PWA (Fases A, B, C ya completadas) | — | Post-U2 |

**Para pasar al builder:** Lee `KNOWN_GAPS_AND_LIMITATIONS.md` + `OPTIONAL_IMPROVEMENTS_STATUS.md`.

---

## ❌ Deprecado (NO USAR)

| Archivo | Por qué | Alternativa |
|---------|---------|-------------|
| **TECHNICAL_MAP.md** | Contiene errores de arquitectura | Lee AUDIT_IMPLEMENTATION.md |
| **_DEPRECATED_BUILDER_GUIDE.md.bak.bak** | Viejo, fue reemplazado | Lee OPTIONAL_IMPROVEMENTS_5-8H.md |

---

## 🎯 Flujos Comunes

### Flujo 1: Nuevo Developer Onboarding
```
1. Leer: ANALISIS_FUNCIONAL_U2.md (especificación)
2. Verificar: AUDIT_IMPLEMENTATION.md (estado actual)
3. Revisar: ROUTES_AUDIT.md (endpoints + seguridad)
4. Consultar: SPRINT_S1.md (configuración)
→ Tiempo: 30 minutos
```

### Flujo 2: Verificar Estado de Mejoras Opcionales
```
1. Leer: OPTIONAL_IMPROVEMENTS_STATUS.md (auditado contra código real)
2. Resultado: I1 ✅ hecho, I2-I4 ❌ no aplican
3. Si necesitas instrucciones originales: OPTIONAL_IMPROVEMENTS_5-8H.md
4. Próximo paso: Despliegue en producción
→ Tiempo: 5 minutos
```

### Flujo 3: PWA Post-U2 (10 horas)
```
1. Leer: PWA_PLAN.md (roadmap)
2. Implementar: PWA-1 → PWA-2 → ... → PWA-5
3. Verificar: Lighthouse audit
→ Tiempo: 10 horas
```

### Flujo 4: Despliegue Producción
```
1. Leer: ../docs/DEPLOYMENT.md (guía completa)
2. Elegir plataforma: Railway, Render, o VPS
3. Configurar variables de entorno: Ver SPRINT_S1.md
4. Deploy
→ Tiempo: 10-30 minutos (según plataforma)
```

---

## 📊 Checklist de Lectura

### Mínimo (Para cerrar U2)
- [ ] ANALISIS_FUNCIONAL_U2.md (especificación)
- [ ] AUDIT_IMPLEMENTATION.md (estado actual)
- [ ] ROUTES_AUDIT.md (endpoints)

### Recomendado (Para producción)
- [ ] KNOWN_GAPS_AND_LIMITATIONS.md (gaps + mejoras)
- [ ] OPTIONAL_IMPROVEMENTS_STATUS.md (estado verificado contra código)
- [ ] SPRINT_S1.md (seguridad)
- [ ] ../docs/DEPLOYMENT.md (cómo desplegar)

### Completo (Para dominar todo)
- [ ] Todos los archivos en `implementado/`
- [ ] KNOWN_GAPS_AND_LIMITATIONS.md
- [ ] OPTIONAL_IMPROVEMENTS_5-8H.md (si implementas mejoras)
- [ ] PWA_PLAN.md (si haces PWA post-U2)

---

## 🚀 Para el Builder Agent

Si pasas trabajo al builder:

1. **Leer:** `porImplementar/KNOWN_GAPS_AND_LIMITATIONS.md`
2. **Instrucciones:** `porImplementar/OPTIONAL_IMPROVEMENTS_5-8H.md`
3. **Verificar:** `implementado/VERIFICATION_CHECKLIST.md`
4. **Commit:** Uno por item (6 commits total)

---

## 📞 Duda Rápida?

| Pregunta | Respuesta rápida |
|----------|-----------------|
| ¿Está hecho X? | Ver AUDIT_IMPLEMENTATION.md |
| ¿Cuál es el especificación? | ANALISIS_FUNCIONAL_U2.md |
| ¿Cómo se configura seguridad? | SPRINT_S1.md |
| ¿Qué rutas existen? | ROUTES_AUDIT.md |
| ¿Qué falta implementar? | KNOWN_GAPS_AND_LIMITATIONS.md |
| ¿Qué mejoras opcionales quedan? | OPTIONAL_IMPROVEMENTS_STATUS.md (I1 ✅, I2-I4 ❌) |
| ¿Cómo verifico cambios? | VERIFICATION_CHECKLIST.md |
| ¿Cómo despliego? | ../docs/DEPLOYMENT.md |

---

**Versión:** 1.0  
**Última actualización:** 2026-10-23  
**Responsable:** Framework Programación Web — UPSE

