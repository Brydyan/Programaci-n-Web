-- ═════════════════════════════════════════════════════════════════════════════
-- 003_seed.sql · MultiCatálogo
--
-- Datos de DESARROLLO. Idempotente con ON CONFLICT / NOT EXISTS: re-aplicar
-- este archivo sobre un volumen existente no duplica ni pisa nada.
--
-- Password de todas las cuentas seed: `12345678`
-- (hash bcrypt cost 12 vía pgcrypto: crypt('12345678', gen_salt('bf', 12))).
--
-- En producción este archivo NO se aplica: se usa un seed separado por entorno.
-- ═════════════════════════════════════════════════════════════════════════════

BEGIN;

-- ── Árbol MLM semilla ────────────────────────────────────────────────────────
-- id 1 "TU" = raíz. Los ids son fijos a propósito: el código y las vistas
-- los referencian por id. setval deja la secuencia por detrás del máximo.
INSERT INTO referidos (id, codigo, parent_id)
    VALUES (1, 'TU', NULL)
    ON CONFLICT (id) DO NOTHING;
SELECT setval('referidos_id_seq', GREATEST((SELECT MAX(id) FROM referidos), 1));

-- ── Cuentas ──────────────────────────────────────────────────────────────────
INSERT INTO usuarios (email, password_hash, rol, nombre)
    VALUES
        ('admin@upse.edu.ec',   crypt('12345678', gen_salt('bf', 12)), 'admin',   'Administrador'),
        ('cliente@upse.edu.ec', crypt('12345678', gen_salt('bf', 12)), 'cliente', 'Cliente Demo')
    ON CONFLICT (email) DO NOTHING;

-- ── Vínculo usuario ↔ nodo (la FK real que faltaba) ─────────────────────────
-- admin → nodo 2 "ADMIN"; cliente → nodo 3 "LUIS", hijo de la raíz.
INSERT INTO referidos (id, codigo, parent_id)
    SELECT 2, 'ADMIN', NULL
    WHERE NOT EXISTS (SELECT 1 FROM referidos WHERE id = 2);

INSERT INTO referidos (id, codigo, parent_id)
    SELECT 3, 'LUIS', 1
    WHERE NOT EXISTS (SELECT 1 FROM referidos WHERE id = 3);

SELECT setval('referidos_id_seq', GREATEST((SELECT MAX(id) FROM referidos), 1));

UPDATE referidos r
   SET usuario_id = (SELECT id FROM usuarios WHERE email = 'admin@upse.edu.ec')
 WHERE r.id = 2;

UPDATE referidos r
   SET usuario_id = (SELECT id FROM usuarios WHERE email = 'cliente@upse.edu.ec')
 WHERE r.id = 3;

UPDATE usuarios u
   SET referido_id = 1
 WHERE u.email = 'admin@upse.edu.ec';

UPDATE usuarios u
   SET referido_id = 3
 WHERE u.email = 'cliente@upse.edu.ec';

-- ── Catálogo mínimo ──────────────────────────────────────────────────────────
INSERT INTO productos (nombre, precio, categoria, stock, img)
    SELECT * FROM (VALUES
        ('Serum Revitalizante', 45.00::numeric, 'rostro',       50, 'https://picsum.photos/seed/serum/150'),
        ('Crema Hidratante Pro', 32.50::numeric, 'rostro',       40, 'https://picsum.photos/seed/crema/150'),
        ('Tónico Purificante',  28.00::numeric, 'rostro',       35, 'https://picsum.photos/seed/tonico/150'),
        ('Mascarilla Nocturna', 50.00::numeric, 'tratamiento',  25, 'https://picsum.photos/seed/mascarilla/150')
    ) AS v(nombre, precio, categoria, stock, img)
    WHERE NOT EXISTS (
        SELECT 1 FROM productos p WHERE p.nombre = v.nombre
    );

COMMIT;