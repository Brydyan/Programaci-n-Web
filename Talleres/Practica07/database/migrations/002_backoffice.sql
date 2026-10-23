-- ═════════════════════════════════════════════════════════════════════════════
-- 002_backoffice.sql · MultiCatálogo
--
-- Back-office: usuarios, catálogo, red multinivel, ledger de comisiones y
-- liquidaciones. Es lo que convierte el checkout en un negocio operable
-- (registro, back-office, consulta de pedidos).
--
-- Idempotente: CREATE ... IF NOT EXISTS + ALTER TABLE ... ADD COLUMN IF NOT
-- EXISTS + guardas DO $$ sobre information_schema para las FKs y los CHECK.
-- Seguro de re-aplicar sobre un volumen con datos.
--
-- Depende de: 001_schema_base.sql (pedidos, pedido_items, comisiones).
-- ═════════════════════════════════════════════════════════════════════════════

BEGIN;

-- ─────────────────────────────────────────────────────────────────────────────
-- 1. TABLAS NUEVAS
-- ─────────────────────────────────────────────────────────────────────────────

-- Usuarios. password_hash se llena con crypt('...', gen_salt('bf', 12))
-- (pgcrypto) o desde la API con bcrypt.GenerateFromPassword.
-- activo=false → login responde 403, no 401 (no es un email inválido).
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
            referido_id    INTEGER,      -- FK declarada al final: referidos aún no existe
            creado_en     TIMESTAMPTZ NOT NULL DEFAULT NOW()
        );
        CREATE INDEX idx_usuarios_email  ON usuarios (email);
        CREATE INDEX idx_usuarios_rol    ON usuarios (rol);
        CREATE INDEX idx_usuarios_activo ON usuarios (activo);
        CREATE INDEX idx_usuarios_refid  ON usuarios (referido_id);
    END IF;
END $$;

-- Catálogo. activo=false = borrado lógico: el producto sale del catálogo
-- público pero sigue anclado en pedido_items históricos.
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

-- Árbol multinivel. usuario_id UNIQUE = un nodo por persona (link user↔red
-- real, no coincidencia de ids). codigo UNIQUE = el link /registro?ref=XXX.
-- parent_id ON DELETE SET NULL: si cae un nodo, sus hijos se promotes a
-- raíz en vez de desaparecer con la cascada.
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

-- FK usuarios.referido_id → referidos.id (ahora sí existe el destino).
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
-- 2. COLUMNAS NUEVAS EN TABLAS EXISTENTES
-- ─────────────────────────────────────────────────────────────────────────────

-- pedidos: identidad real del comprador + ciclo de vida.
-- usuario_id se puebla desde el claim del JWT (sub), no desde el body.
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
CREATE INDEX IF NOT EXISTS idx_pedidos_usuario ON pedidos (usuario_id);
CREATE INDEX IF NOT EXISTS idx_pedidos_estado  ON pedidos (estado);
CREATE INDEX IF NOT EXISTS idx_pedidos_created ON pedidos (created_at DESC);

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

-- pedido_items.producto_id → productos.id. RESTRICT (no CASCADE): no se puede
-- borrar un producto que ya se vendió; el borrado de producto es lógico.
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

-- comisiones: de "contador" a ledger auditable (quién cobró qué, por qué pedido).
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

CREATE INDEX IF NOT EXISTS idx_comisiones_referido ON comisiones (referido_id, estado);
CREATE INDEX IF NOT EXISTS idx_comisiones_pedido   ON comisiones (pedido_id);
CREATE INDEX IF NOT EXISTS idx_comisiones_estado   ON comisiones (estado);

-- ─────────────────────────────────────────────────────────────────────────────
-- 3. LIQUIDACIONES — cierran el ciclo del dinero
-- ─────────────────────────────────────────────────────────────────────────────

-- Una liquidación = un pago grouping comisiones ya devengadas.
-- referred_id RESTRICT: no se puede borrar un distribuidor con historial de pago.
CREATE TABLE IF NOT EXISTS liquidaciones (
    id          SERIAL PRIMARY KEY,
    referido_id INTEGER NOT NULL REFERENCES referidos(id) ON DELETE RESTRICT,
    total       NUMERIC(12,2) NOT NULL CHECK (total >= 0),
    periodo     VARCHAR(20),
    estado      VARCHAR(20) NOT NULL DEFAULT 'pagada'
                CHECK (estado IN ('pendiente','pagada')),
    creado_en   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    pagada_en   TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_liquidaciones_ref     ON liquidaciones (referido_id);
CREATE INDEX IF NOT EXISTS idx_liquidaciones_periodo ON liquidaciones (periodo);

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

COMMIT;