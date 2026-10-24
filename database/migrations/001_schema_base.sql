-- ═════════════════════════════════════════════════════════════════════════════
-- 001_schema_base.sql · MultiCatálogo
--
-- Esquema base: lo mínimo que necesita el checkout para existir. Todo lo demás
-- entra en 002 (back-office). Este archivo NO lleva seed — los datos de
-- desarrollo van en 003_seed.sql, para que un entorno limpio sea explícito.
--
-- Idempotente: todos los CREATE usan IF NOT EXISTS. Se aplica en orden
-- alfabético (docker-entrypoint-initdb.d) o con `make db-migrate`.
-- ═════════════════════════════════════════════════════════════════════════════

BEGIN;

-- crypt()/gen_salt('bf') para los hashes bcrypt de usuarios.password_hash.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ── Pedidos ──────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS pedidos (
    id           SERIAL PRIMARY KEY,
    email_usuario TEXT NOT NULL,
    total        NUMERIC(10,2) NOT NULL,
    metodo_pago  TEXT NOT NULL,
    created_at   TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pedidos_email ON pedidos (email_usuario);

-- ── Ítems del pedido ────────────────────────────────────────────────────────
-- nombre/precio se copian aquí a propósito: el pedido es una foto histórica
-- del precio cobrado, immune a cambios posteriores en productos.
CREATE TABLE IF NOT EXISTS pedido_items (
    id          SERIAL PRIMARY KEY,
    pedido_id   INTEGER REFERENCES pedidos(id) ON DELETE CASCADE,
    producto_id INTEGER NOT NULL,
    nombre      TEXT NOT NULL,
    precio      NUMERIC(10,2) NOT NULL,
    cantidad    INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_pedido_items_pedido ON pedido_items (pedido_id);

-- ── Comisiones (ledger) ─────────────────────────────────────────────────────
-- origen_email: quién compró y generó la comisión. pedido_id y referido_id
-- se agregan en 002 con sus FKs.
CREATE TABLE IF NOT EXISTS comisiones (
    id            SERIAL PRIMARY KEY,
    nivel         INTEGER NOT NULL,
    monto         NUMERIC(10,2) NOT NULL,
    origen_email  TEXT NOT NULL,
    created_at    TIMESTAMPTZ DEFAULT NOW()
);

COMMIT;