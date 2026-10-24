-- ═════════════════════════════════════════════════════════════════════════════
-- demo_data.sql · MultiCatálogo — Datos de demostración
--
-- 003_seed.sql deja el catálogo mínimo (4 productos) y CERO pedidos, por lo que
-- el dashboard, los reportes, la red y el ledger salen vacíos: no hay forma de
-- comprobar que la UI se pinta contra datos reales.
--
-- Este script genera un dataset verosímil: red de 3 niveles, catálogo amplio y
-- ~90 pedidos repartidos en los últimos 120 días, con su ledger de comisiones
-- calculado con la MISMA regla que usa el backend (domain.TasasComision:
-- 10/5/2 %, máximo 3 niveles — backend/domain/comisiones.go).
--
-- ⚠️ Por qué SQL y no la API:
-- `pedidos.created_at` tiene DEFAULT NOW(), así que por API todos los pedidos
-- salen con la misma fecha y los reportes por periodo no se pueden ver. Para
-- histórico hay que escribir la columna directamente.
--
-- IDEMPOTENTE: borra lo que generó en una corrida previa (marcado por el
-- sufijo @demo.ec y los nombres con prefijo 'Demo ') y lo vuelve a crear.
-- Puedes correrlo las veces que quieras.
--
-- Uso:
--   make db-demo            # desde la raíz de Practica07
--   docker compose exec -T postgres psql -U admin -d multicatalogo \
--     < database/seeds/demo_data.sql
-- ═════════════════════════════════════════════════════════════════════════════

BEGIN;

-- ─────────────────────────────────────────────────────────────────────────────
-- 0. Limpieza de la corrida anterior
-- ─────────────────────────────────────────────────────────────────────────────
DELETE FROM comisiones      WHERE origen_email LIKE '%@demo.ec';
DELETE FROM pedido_items    WHERE pedido_id IN (
    SELECT id FROM pedidos WHERE email_usuario LIKE '%@demo.ec');
DELETE FROM pedidos         WHERE email_usuario LIKE '%@demo.ec';
-- Sin DELETE FROM referidos aquí, a propósito: referidos_usuario_id_fkey es
-- ON DELETE CASCADE, así que el DELETE de usuarios de la línea siguiente ya
-- arrastra sus referidos.
--
-- La versión anterior hacía `DELETE FROM referidos WHERE usuario_id IN (SELECT
-- id FROM usuarios WHERE email LIKE '%@demo.ec')` DESPUÉS de borrar los
-- usuarios. Ese subquery ya no encontraba nada —los acababa de borrar— así que
-- la sentencia afectaba cero filas y era código muerto. Peor: aparentaba
-- limpiar la red cuando el trabajo real lo hacía el CASCADE. Si algún día el
-- CASCADE cambiara a RESTRICT, el seed empezaría a dejar referidos huérfanos
-- sin avisar.
--
-- liquidaciones_referido_id_fkey es RESTRICT, y comisiones_liquidacion_id_fkey
-- NO ACTION: por eso las liquidaciones se borran explícitamente y ANTES que las
-- comisiones que las referencian. Sin esta línea, liquidar a un distribuidor
-- demo y volver a correr el seed dejaba pagos acumulados y el dataset ya no
-- era reproducible.
DELETE FROM liquidaciones  WHERE referido_id IN (
    SELECT id FROM referidos WHERE codigo LIKE 'MKT-%' OR codigo LIKE 'SOP-%');
DELETE FROM usuarios        WHERE email LIKE '%@demo.ec';
DELETE FROM productos       WHERE nombre LIKE 'Demo %';

-- Reinicia las secuencias para que la corrida N produzca EXACTAMENTE los
-- mismos ids que la corrida N-1. Sin esto el dataset cambia en cada
-- ejecución (las ventas se derivan de p.id), y no se pueden comparar dos
-- capturas antes/después ni verificar que el generador es determinista.
SELECT setval('pedidos_id_seq', 1, false);
SELECT setval('pedido_items_id_seq', 1, false);
SELECT setval('comisiones_id_seq', 1, false);

-- ─────────────────────────────────────────────────────────────────────────────
-- 1. Red de tres niveles
--
-- Los nodos 1-3 son los de 003_seed.sql. Se añaden nodos con usuario en los
-- niveles 2 y 3 para que existan cadenas de 2 y de 3 niveles, y así ver las
-- tasas 10 % y 5 % en acción, además del 2 %.
--
--   1 RAIZ          (sin usuario, es el distribuidor raíz)
--   ├── 2 ADMIN     admin@upse.edu.ec     (nivel 1)
--   │   └── 10 MKT-ANA   ana@demo.ec      (nivel 2)
--   │       ├── 30 SOP-SOF  sofia@demo.ec (nivel 3)
--   │       └── 31 SOP-DIE  diego@demo.ec
--   └── 3 LUIS      cliente@upse.edu.ec
--       └── 11 MKT-LUIS   luis@demo.ec
--           └── 32 SOP-ANA2  andrea@demo.ec
--
-- Cuando compra quien está en el nodo 30, la cadena es 10 → 2 → 1: tres
-- niveles, 10/5/2. Cuando compra quien está en el nodo 11, es 2 → 1: dos
-- niveles, 10/5.
-- ─────────────────────────────────────────────────────────────────────────────
INSERT INTO usuarios (email, password_hash, rol, nombre) VALUES
    ('ana@demo.ec',     crypt('12345678', gen_salt('bf', 12)), 'cliente', 'Ana Demo'),
    ('sofia@demo.ec',   crypt('12345678', gen_salt('bf', 12)), 'cliente', 'Sofía Demo'),
    ('diego@demo.ec',   crypt('12345678', gen_salt('bf', 12)), 'cliente', 'Diego Demo'),
    ('luis@demo.ec',    crypt('12345678', gen_salt('bf', 12)), 'cliente', 'Luis Demo'),
    ('andrea@demo.ec',  crypt('12345678', gen_salt('bf', 12)), 'cliente', 'Andrea Demo')
ON CONFLICT (email) DO NOTHING;

INSERT INTO referidos (id, codigo, parent_id, usuario_id)
SELECT v.id, v.codigo, v.parent_id, u.id
  FROM (VALUES
      (10, 'MKT-ANA',    2,  'ana@demo.ec'),
      (11, 'MKT-LUIS',   3,  'luis@demo.ec'),
      (30, 'SOP-SOF',   10,  'sofia@demo.ec'),
      (31, 'SOP-DIE',   10,  'diego@demo.ec'),
      (32, 'SOP-AND',   11,  'andrea@demo.ec')
  ) AS v(id, codigo, parent_id, email)
  LEFT JOIN usuarios u ON u.email = v.email
 WHERE NOT EXISTS (SELECT 1 FROM referidos r WHERE r.id = v.id);

SELECT setval('referidos_id_seq', GREATEST((SELECT MAX(id) FROM referidos), 1));

UPDATE usuarios u SET referido_id = 10 WHERE u.email = 'ana@demo.ec';
UPDATE usuarios u SET referido_id = 11 WHERE u.email = 'luis@demo.ec';
UPDATE usuarios u SET referido_id = 30 WHERE u.email = 'sofia@demo.ec';
UPDATE usuarios u SET referido_id = 31 WHERE u.email = 'diego@demo.ec';
UPDATE usuarios u SET referido_id = 32 WHERE u.email = 'andrea@demo.ec';

-- ─────────────────────────────────────────────────────────────────────────────
-- 2. Catálogo ampliado
--
-- Los 4 de 003_seed.sql se quedan; estos 12 entran con stock alto (400)
-- porque el generador vende en total más de lo que el stock original
-- aguantaría. Las existencias se ajustan al final para que cuadren con las
-- ventas realmente generadas.
-- ─────────────────────────────────────────────────────────────────────────────
INSERT INTO productos (nombre, precio, categoria, stock, img)
SELECT * FROM (VALUES
    ('Demo Serum Hydratante',      68.90,  'rostro',      400, 'https://picsum.photos/seed/serum2/150'),
    ('Demo Cleanser Gel',           24.50,  'rostro',      400, 'https://picsum.photos/seed/cleanser/150'),
    ('Demo Protector Solar SPF50',  42.00,  'rostro',      400, 'https://picsum.photos/seed/spf/150'),
    ('Demo Tonico Facial',         36.75,  'rostro',      400, 'https://picsum.photos/seed/tonico2/150'),
    ('Demo Exfoliante AHA',        55.30,  'tratamiento', 400, 'https://picsum.photos/seed/exfoliante/150'),
    ('Demo Gel Contorno Ojos',      29.99,  'tratamiento', 400, 'https://picsum.photos/seed/gel/150'),
    ('Demo Mascarilla de Arcilla',  31.40,  'tratamiento', 400, 'https://picsum.photos/seed/arcilla/150'),
    ('Demo Aceite Facial Nocturno', 47.80,  'tratamiento', 400, 'https://picsum.photos/seed/aceite/150'),
    ('Demo Shampoo Anticaspa',      26.90,  'cabello',     400, 'https://picsum.photos/seed/shampoo2/150'),
    ('Demo Acondicionador Keratina',33.60,  'cabello',     400, 'https://picsum.photos/seed/acond/150'),
    ('Demo Serum Capilar',          39.20,  'cabello',     400, 'https://picsum.photos/seed/serumcap/150'),
    ('Demo Labial Hidratante',      18.45,  'labios',      400, 'https://picsum.photos/seed/labial/150')
) AS v(nombre, precio, categoria, stock, img)
WHERE NOT EXISTS (SELECT 1 FROM productos p WHERE p.nombre = v.nombre);

-- ─────────────────────────────────────────────────────────────────────────────
-- 3. Pedidos
--
-- 90 pedidos repartidos en los últimos 120 días. El reparto de estados
-- imita uno real: la mayoría entregado, una cola viva de pendientes y
-- algunos cancelados.
--
-- `w` es un peso fijo por fila (NO aleatorio) para que el resultado sea
-- reproducible: la misma semilla da siempre las mismas cifras. Con
-- random() los números cambiarían en cada corrida y no se podrían comparar
-- capturas antes/después.
-- ─────────────────────────────────────────────────────────────────────────────
INSERT INTO pedidos
    (email_usuario, usuario_id, total, metodo_pago, estado, created_at,
     direccion, ciudad, telefono)
SELECT
    u.email,
    u.id,
    0,                                    -- se recalcula desde los items
    (ARRAY['tarjeta','efectivo','transferencia'])[1 + (g % 3)],
    CASE
        WHEN g % 20 IN (0, 1)      THEN 'cancelado'
        WHEN g % 20 IN (2, 3, 4)   THEN 'pendiente'
        WHEN g % 20 IN (5, 6)      THEN 'preparando'
        WHEN g % 20 IN (7, 8, 9)   THEN 'enviado'
        ELSE                            'entregado'
    END,
    NOW() - ((g * 13) || ' hours')::interval,
    (ARRAY[
        'Av. 10 de Agosto 100', 'Av. República del Salvador 455',
        'Av. Amazonas N34-120',    'Calle Los Ríos 780'
    ])[1 + (g % 4)],
    (ARRAY['Quito', 'Guayaquil', 'Cuenca', 'Sangolquí'])[1 + (g % 4)],
    '09' || lpad((g % 10)::text, 1, '0') || lpad((1000000 + g * 7919)::text, 7, '0')
  FROM generate_series(1, 90) AS g
  -- Un comprador por pedido, rotando entre los cinco. Determinista: la
  -- corrida N siempre da la misma distribución, sin random().
  JOIN usuarios u
    ON u.email = (ARRAY['ana@demo.ec','sofia@demo.ec','diego@demo.ec',
                         'luis@demo.ec','andrea@demo.ec'])[1 + (g % 5)];

-- Cada pedido lleva 1-3 líneas de productos ya descontados de stock.
INSERT INTO pedido_items (pedido_id, producto_id, nombre, precio, cantidad)
SELECT
    p.id,
    pr.id,
    pr.nombre,
    pr.precio,
    1 + ((p.id + i) % 3)
  FROM pedidos p
  CROSS JOIN generate_series(1, 1 + (p.id % 3)) AS i
  JOIN productos pr ON pr.id = 1 + ((p.id * 3 + i * 7) % (SELECT COUNT(*) FROM productos))
 WHERE p.email_usuario LIKE '%@demo.ec';

-- El total sale de los items, nunca del body del cliente (misma regla que
-- CreatePedidoConUsuario).
UPDATE pedidos p
   SET total = sub.total
  FROM (SELECT pedido_id, SUM(precio * cantidad) AS total
          FROM pedido_items GROUP BY pedido_id) AS sub
 WHERE sub.pedido_id = p.id;

-- Stock restante = 400 menos lo vendido por el generador.
UPDATE productos pr
   SET stock = GREATEST(0, 400 - COALESCE((
       SELECT SUM(pi.cantidad)
         FROM pedido_items pi
         JOIN pedidos p ON p.id = pi.pedido_id
        WHERE pi.producto_id = pr.id
          AND p.estado <> 'cancelado'), 0))
 WHERE pr.nombre LIKE 'Demo %';

-- ─────────────────────────────────────────────────────────────────────────────
-- 4. Ledger de comisiones — MISMA regla que backend/domain/comisiones.go
--
-- Tasas: 10 % nivel 1, 5 % nivel 2, 2 % nivel 3. Se corta en cuanto se acaba
-- la cadena o se agotan las tasas.
--
-- `cancelado` NO genera comisiones: al cancelar el backend revierte las que
-- hubiera dejado pendientes. Para que el historial sea coherente, los pedidos
-- cancelados quedan con una fila 'revertida' (10 %) que representa lo que se
-- anuló, y así MiRed y /api/reportes/comisiones-pendientes tienen algo que
-- mostrar de ambos lados.
-- ─────────────────────────────────────────────────────────────────────────────
INSERT INTO comisiones (pedido_id, referido_id, nivel, porcentaje, monto, estado,
                        origen_email, created_at)
-- La lista de columnas es obligatoria: sin ella Postgres deduce los nombres
-- de la rama ancla (id, total, ...), donde `p.id` y `r.id` colisionarían en
-- `id` y `up.pedido_id` no existiría.
WITH RECURSIVE upline
    (pedido_id, total, estado, email_usuario, created_at,
     referido_id, nivel, parent_id) AS (
    -- Ancla: el padre del nodo del comprador es su nivel 1.
    -- parent_id se proyecta porque el término recursivo lo lee.
    SELECT p.id, p.total, p.estado, p.email_usuario, p.created_at,
           r.id, 1, r.parent_id
      FROM pedidos p
      JOIN usuarios   comprador ON comprador.id = p.usuario_id
      JOIN referidos  propio   ON propio.id = comprador.referido_id
      JOIN referidos  r        ON r.id = propio.parent_id
     WHERE p.email_usuario LIKE '%@demo.ec'

    UNION ALL

    -- Recursivo: sube por los padres hasta el tope de niveles.
    SELECT up.pedido_id, up.total, up.estado, up.email_usuario, up.created_at,
           r.id, up.nivel + 1, r.parent_id
      FROM upline up
      JOIN referidos r ON r.id = up.parent_id
     WHERE up.nivel < 3
)
SELECT
    pedido_id,
    referido_id,
    nivel,
    ROUND((ARRAY[10.00, 5.00, 2.00])[nivel], 2)          AS porcentaje,
    ROUND(total * (ARRAY[0.10, 0.05, 0.02])[nivel], 2)    AS monto,
    CASE
        WHEN estado = 'cancelado'             THEN 'revertida'
        WHEN nivel = 1 AND pedido_id % 4 = 0   THEN 'pagada'
        WHEN nivel = 1                         THEN 'aprobada'
        ELSE                                       'pendiente'
    END                                                AS estado_comm,
    email_usuario,
    created_at
  FROM upline
 WHERE ROUND(total * (ARRAY[0.10, 0.05, 0.02])[nivel], 2) > 0;

-- ═════════════════════════════════════════════════════════════════════════════
-- 5. Contadores referidos.ventas
--
-- `ventas` es un acumulado best-effort que el checkout incrementa para el
-- upline del comprador (repository.go → ActualizarVentasUpline). Aquí se
-- calcula igual, sólo con pedidos no cancelados, para que el árbol de la red y
-- los reportes cuadren con el ledger.
-- ═════════════════════════════════════════════════════════════════════════════
UPDATE referidos r
   SET ventas = COALESCE(sub.total, 0)
  FROM (
      SELECT c.referido_id, SUM(c.monto) AS total
        FROM comisiones c
       WHERE c.estado <> 'revertida'
       GROUP BY c.referido_id
  ) AS sub
 WHERE sub.referido_id = r.id;

-- Los nodos que novable de este generador conservan su valor previo (nodos 1-3).
UPDATE referidos SET ventas = 0 WHERE usuario_id IS NULL AND ventas = 0;

COMMIT;

-- ─────────────────────────────────────────────────────────────────────────────
-- Resumen
-- ─────────────────────────────────────────────────────────────────────────────
\echo ''
\echo '── Datos de demostración generados ──'
SELECT 'usuarios'  AS tabla, count(*) AS filas FROM usuarios
UNION ALL SELECT 'productos',  count(*) FROM productos
UNION ALL SELECT 'referidos',  count(*) FROM referidos
UNION ALL SELECT 'pedidos',    count(*) FROM pedidos
UNION ALL SELECT 'items',      count(*) FROM pedido_items
UNION ALL SELECT 'comisiones', count(*) FROM comisiones
ORDER BY 1;

\echo ''
\echo '── Pedidos por estado ──'
SELECT estado, count(*), ROUND(SUM(total), 2) AS total
  FROM pedidos GROUP BY estado ORDER BY estado;

\echo ''
\echo '── Comisiones por estado ──'
SELECT estado, count(*), ROUND(SUM(monto), 2) AS total
  FROM comisiones GROUP BY estado ORDER BY estado;

\echo ''
\echo 'Cuentas de prueba (password: 12345678):'
\echo '  admin@upse.edu.ec / 12345678   (admin)'
\echo '  ana@demo.ec     / 12345678     (nivel 2 de la red)'
\echo '  sofia@demo.ec   / 12345678     (nivel 3 — genera cadena de 3 niveles)'
\echo '  luis@demo.ec    / 12345678     (otra rama)'