CREATE TABLE IF NOT EXISTS pedidos (
    id          SERIAL PRIMARY KEY,
    email_usuario TEXT NOT NULL,
    total       NUMERIC(10,2) NOT NULL,
    metodo_pago TEXT NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pedido_items (
    id          SERIAL PRIMARY KEY,
    pedido_id   INTEGER REFERENCES pedidos(id) ON DELETE CASCADE,
    producto_id INTEGER NOT NULL,
    nombre      TEXT NOT NULL,
    precio      NUMERIC(10,2) NOT NULL,
    cantidad    INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS comisiones (
    id            SERIAL PRIMARY KEY,
    nivel         INTEGER NOT NULL,
    monto         NUMERIC(10,2) NOT NULL,
    origen_email  TEXT NOT NULL,
    created_at    TIMESTAMPTZ DEFAULT NOW()
);
