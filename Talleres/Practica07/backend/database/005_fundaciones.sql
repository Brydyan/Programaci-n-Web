-- ─────────────────────────────────────────────────────────────────────────────
-- 005_fundaciones.sql  ·  Sprint S1  ·  MultiCatálogo
--
-- A2 del plan: migraciones idempotentes para preparar el back-office.
-- Aplicar DESPUÉS de init.sql y de las migraciones previas (002, 003, 004).
--
-- Idempotente: todos los CREATE usan IF NOT EXISTS; ALTER usa ADD COLUMN IF NOT EXISTS.
-- Seguro contra re-ejecución y contra volumen con datos.
--
-- Cubre F0 del PLAN_IMPLEMENTACION:
--   * Columnas nuevas en tablas existentes
--   * Tablas nuevas (liquidaciones, indices adicionales)
--   * Enriquecer `comisiones` para soportar ledger auditable
--   * Enlazar seed usuarios → referidos (admin→0 'TU', cliente→2 'LUIS')
--   * Generar codigos de referido únicos para el seed
-- ─────────────────────────────────────────────────────────────────────────────

BEGIN;

-- ── Extensión pgcrypto (crypt/gen_salt ya existe si 003_seguridad.sql corrió)
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ─────────────────────────────────────────────────────────────────────────────
-- 1. TABLAS QUE FALTABAN EN init.sql (proyecto inicial solo traía pedidos)
-- ─────────────────────────────────────────────────────────────────────────────

-- Usuarios: bcrypt ya se almacena en password_hash (si 003_seguridad.sql corrió,
-- la columna se llama password; si no, se crea con password_hash).
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.tables WHERE table_name = 'usuarios'
    ) THEN
        CREATE TABLE usuarios (
            id            SERIAL PRIMARY KEY,
            email         VARCHAR(255) UNIQUE NOT NULL,
            password_hash VARCHAR(255),
            rol           VARCHAR(20) NOT NULL DEFAULT 'cliente'
                          CHECK (rol IN ('cliente','admin')),
            nombre        VARCHAR(120),
            activo        BOOLEAN NOT NULL DEFAULT true,
            referido_id   INTEGER,           -- FK declarada al final para evitar orden
            creado_en     TIMESTAMPTZ NOT NULL DEFAULT NOW()
        );
        CREATE INDEX idx_usuarios_email   ON usuarios (email);
        CREATE INDEX idx_usuarios_rol     ON usuarios (rol);
        CREATE INDEX idx_usuarios_activo  ON usuarios (activo);
        CREATE INDEX idx_usuarios_refid   ON usuarios (referido_id);
    END IF;
END $$;

-- Productos
CREATE TABLE IF NOT EXISTS productos (
    id        SERIAL PRIMARY KEY,
    nombre    VARCHAR(200) NOT NULL,
    precio    NUMERIC(10,2) NOT NULL CHECK (precio >= 0),
    categoria VARCHAR(80)  NOT NULL,
    stock     INTEGER      NOT NULL DEFAULT 0 CHECK (stock >= 0),
    img       TEXT         NOT NULL DEFAULT '',
    activo    BOOLEAN      NOT NULL DEFAULT true,
    creado_en TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_productos_categoria ON productos (categoria);
CREATE INDEX IF NOT EXISTS idx_productos_activo    ON productos (activo);

-- Referidos (nodos del árbol multinivel)
CREATE TABLE IF NOT EXISTS referidos (
    id         SERIAL PRIMARY KEY,
    usuario_id INTEGER UNIQUE REFERENCES usuarios(id) ON DELETE CASCADE,
    codigo     VARCHAR(16) UNIQUE NOT NULL,
    parent_id  INTEGER REFERENCES referidos(id) ON DELETE SET NULL,
    ventas     NUMERIC(12,2) NOT NULL DEFAULT 0,
    creado_en  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_referidos_parent ON referidos (parent_id);
CREATE INDEX IF NOT EXISTS idx_referidos_codigo ON referidos (codigo);

-- Ahora que referidos existe, declaramos la FK de usuarios.referido_id
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'usuarios_referido_id_fkey'
    ) THEN
        ALTER TABLE usuarios
            ADD CONSTRAINT usuarios_referido_id_fkey
            FOREIGN KEY (referido_id) REFERENCES referidos(id) ON DELETE SET NULL;
    END IF;
END $$;

-- ─────────────────────────────────────────────────────────────────────────────
-- 2. COLUMNAS NUEVAS EN TABLAS EXISTENTES (init.sql + checkout)
-- ─────────────────────────────────────────────────────────────────────────────

-- pedidos: agregar usuario_id (FK), estado con CHECK, conservar email_usuario
ALTER TABLE pedidos
    ADD COLUMN IF NOT EXISTS usuario_id INTEGER;
ALTER TABLE pedidos
    ADD COLUMN IF NOT EXISTS estado VARCHAR(20) NOT NULL DEFAULT 'pendiente';
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'pedidos_estado_check'
    ) THEN
        ALTER TABLE pedidos
            ADD CONSTRAINT pedidos_estado_check
            CHECK (estado IN ('pendiente','preparando','enviado','entregado','cancelado'));
    END IF;
END $$;
CREATE INDEX IF NOT EXISTS idx_pedidos_usuario  ON pedidos (usuario_id);
CREATE INDEX IF NOT EXISTS idx_pedidos_estado   ON pedidos (estado);
CREATE INDEX IF NOT EXISTS idx_pedidos_created  ON pedidos (created_at DESC);

-- FK usuario_id (no se hizo arriba por orden de creación)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'pedidos_usuario_id_fkey'
    ) THEN
        ALTER TABLE pedidos
            ADD CONSTRAINT pedidos_usuario_id_fkey
            FOREIGN KEY (usuario_id) REFERENCES usuarios(id) ON DELETE SET NULL;
    END IF;
END $$;

-- pedido_items: asegurar FK a productos (antes podía ser huérfano)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'pedido_items_producto_id_fkey'
    ) THEN
        ALTER TABLE pedido_items
            ADD CONSTRAINT pedido_items_producto_id_fkey
            FOREIGN KEY (producto_id) REFERENCES productos(id) ON DELETE RESTRICT;
    END IF;
END $$;

-- comisiones: ampliar a ledger auditable
ALTER TABLE comisiones
    ADD COLUMN IF NOT EXISTS pedido_id   INTEGER;
ALTER TABLE comisiones
    ADD COLUMN IF NOT EXISTS referido_id INTEGER;
ALTER TABLE comisiones
    ADD COLUMN IF NOT EXISTS porcentaje  NUMERIC(5,2);
ALTER TABLE comisiones
    ADD COLUMN IF NOT EXISTS estado      VARCHAR(20) NOT NULL DEFAULT 'pendiente';
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'comisiones_estado_check'
    ) THEN
        ALTER TABLE comisiones
            ADD CONSTRAINT comisiones_estado_check
            CHECK (estado IN ('pendiente','aprobada','pagada','revertida'));
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'comisiones_pedido_id_fkey'
    ) THEN
        ALTER TABLE comisiones
            ADD CONSTRAINT comisiones_pedido_id_fkey
            FOREIGN KEY (pedido_id) REFERENCES pedidos(id) ON DELETE CASCADE;
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'comisiones_referido_id_fkey'
    ) THEN
        ALTER TABLE comisiones
            ADD CONSTRAINT comisiones_referido_id_fkey
            FOREIGN KEY (referido_id) REFERENCES referidos(id) ON DELETE SET NULL;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_comisiones_referido  ON comisiones (referido_id, estado);
CREATE INDEX IF NOT EXISTS idx_comisiones_pedido    ON comisiones (pedido_id);
CREATE INDEX IF NOT EXISTS idx_comisiones_estado    ON comisiones (estado);

-- ─────────────────────────────────────────────────────────────────────────────
-- 3. TABLAS NUEVAS
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS liquidaciones (
    id         SERIAL PRIMARY KEY,
    referido_id INTEGER NOT NULL REFERENCES referidos(id) ON DELETE RESTRICT,
    total      NUMERIC(12,2) NOT NULL CHECK (total >= 0),
    periodo    VARCHAR(20),
    estado     VARCHAR(20) NOT NULL DEFAULT 'pagada'
               CHECK (estado IN ('pendiente','pagada')),
    creado_en  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    pagada_en  TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_liquidaciones_ref    ON liquidaciones (referido_id);
CREATE INDEX IF NOT EXISTS idx_liquidaciones_periodo ON liquidaciones (periodo);

-- Vincular comisiones a liquidaciones
ALTER TABLE comisiones
    ADD COLUMN IF NOT EXISTS liquidacion_id INTEGER;
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'comisiones_liquidacion_id_fkey'
    ) THEN
        ALTER TABLE comisiones
            ADD CONSTRAINT comisiones_liquidacion_id_fkey
            FOREIGN KEY (liquidacion_id) REFERENCES liquidaciones(id) ON DELETE SET NULL;
    END IF;
END $$;
CREATE INDEX IF NOT EXISTS idx_comisiones_liquidacion ON comisiones (liquidacion_id);

-- ─────────────────────────────────────────────────────────────────────────────
-- 4. SEED MÍNIMO PARA DESARROLLO (idempotente con ON CONFLICT)
--    Si los usuarios ya existen (por 003_seguridad.sql), no se duplican.
-- ─────────────────────────────────────────────────────────────────────────────

-- Nodo raíz del MLM (id=1 reservado para "Tú"/admin)
INSERT INTO referidos (id, codigo, parent_id)
    VALUES (1, 'TU', NULL)
    ON CONFLICT (id) DO NOTHING;
SELECT setval('referidos_id_seq', GREATEST((SELECT MAX(id) FROM referidos), 1));

-- Usuarios seed (admin + cliente demo).
-- Password = '123456' hasheada con bcrypt cost 12 vía pgcrypto.
INSERT INTO usuarios (email, password_hash, rol, nombre)
    VALUES
        ('admin@upse.edu.ec',   crypt('12345678', gen_salt('bf', 12)), 'admin',   'Administrador'),
        ('cliente@upse.edu.ec', crypt('12345678', gen_salt('bf', 12)), 'cliente', 'Cliente Demo')
    ON CONFLICT (email) DO NOTHING;

-- Enlazar admin → nodo 1 (Tú)
INSERT INTO referidos (id, codigo, parent_id)
    SELECT 2, 'ADMIN', NULL
    WHERE NOT EXISTS (SELECT 1 FROM referidos WHERE id = 2);

UPDATE referidos r
   SET usuario_id = (SELECT id FROM usuarios WHERE email = 'admin@upse.edu.ec')
 WHERE r.id = 2;

UPDATE usuarios u
   SET referido_id = 1
 WHERE u.email = 'admin@upse.edu.ec';

-- Nodo "Luis Poveda" (id=3) hijo de "Tú" — el cliente seed queda debajo de admin
INSERT INTO referidos (id, codigo, parent_id)
    SELECT 3, 'LUIS', 1
    WHERE NOT EXISTS (SELECT 1 FROM referidos WHERE id = 3);

UPDATE referidos r
   SET usuario_id = (SELECT id FROM usuarios WHERE email = 'cliente@upse.edu.ec')
 WHERE r.id = 3;

UPDATE usuarios u
   SET referido_id = 3
 WHERE u.email = 'cliente@upse.edu.ec';

SELECT setval('referidos_id_seq', GREATEST((SELECT MAX(id) FROM referidos), 1));

-- Productos seed (catálogo mínimo de cosméticos). Idempotente por nombre.
INSERT INTO productos (nombre, precio, categoria, stock, img)
    SELECT * FROM (VALUES
        ('Serum Revitalizante', 45.00::numeric, 'rostro',   50, 'https://picsum.photos/seed/serum/150'),
        ('Crema Hidratante Pro', 32.50::numeric, 'rostro',   40, 'https://picsum.photos/seed/crema/150'),
        ('Tónico Purificante',  28.00::numeric, 'rostro',    35, 'https://picsum.photos/seed/tonico/150'),
        ('Mascarilla Nocturna', 50.00::numeric, 'tratamiento', 25, 'https://picsum.photos/seed/mascarilla/150')
    ) AS v(nombre, precio, categoria, stock, img)
    WHERE NOT EXISTS (
        SELECT 1 FROM productos p WHERE p.nombre = v.nombre
    );

COMMIT;

-- ─────────────────────────────────────────────────────────────────────────────
-- Verificación post-aplicación (ejecutar manualmente en psql):
--   \d usuarios
--   \d productos
--   \d referidos
--   \d pedidos
--   \d comisiones
--   \d liquidaciones
--   SELECT email, rol, referido_id FROM usuarios;
--   SELECT id, codigo, parent_id, usuario_id FROM referidos;
-- ─────────────────────────────────────────────────────────────────────────────