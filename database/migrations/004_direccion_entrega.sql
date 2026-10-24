-- ═════════════════════════════════════════════════════════════════════════════
-- 004_direccion_entrega.sql · MultiCatálogo
--
-- Añade los datos de envío que el contrato ya exige pero la tabla no tenía.
--
-- Contexto: openapi.yaml → components.schemas.CheckoutRequest declara
-- `required: [direccion, ciudad, telefono, metodo_pago, items]`, y el
-- frontend (frontend/src/types/api.ts) ya manda esos campos. Pero 001 creó
-- `pedidos` sin ninguna de las tres columnas, así que el requisito era
-- incumplible desde el esquema. Esta migración cierra esa brecha.
--
-- Idempotente: ADD COLUMN IF NOT EXISTS. Se aplica con `make db-migrate` o por
-- orden alfabético (docker-entrypoint-initdb.d). Re-aplicarla sobre un volumen
-- con datos es seguro.
--
-- ⚠ Sobre NOT NULL DEFAULT '': las filas que ya existían reciben ''. Es un
-- placeholder honesto, no un dato: no se inventa una dirección. Si el negocio
-- necesita el histórico de esos pedidos, hay que rellenarlo a mano antes de
-- activar cualquier constraint NOT NULL sin default.
-- ═════════════════════════════════════════════════════════════════════════════

BEGIN;

ALTER TABLE pedidos
    ADD COLUMN IF NOT EXISTS direccion TEXT NOT NULL DEFAULT '';
ALTER TABLE pedidos
    ADD COLUMN IF NOT EXISTS ciudad    TEXT NOT NULL DEFAULT '';
ALTER TABLE pedidos
    ADD COLUMN IF NOT EXISTS telefono  TEXT NOT NULL DEFAULT '';

-- El teléfono sólo se valida en el borde (DTO). En la base se acota la
-- longitud para que un body malicioso no pueda escribir megabytes.
ALTER TABLE pedidos
    DROP CONSTRAINT IF EXISTS pedidos_telefono_len;
ALTER TABLE pedidos
    ADD CONSTRAINT pedidos_telefono_len
    CHECK (char_length(telefono) <= 30);

-- El mismo límite para la dirección y la ciudad.
ALTER TABLE pedidos
    DROP CONSTRAINT IF EXISTS pedidos_direccion_len;
ALTER TABLE pedidos
    ADD CONSTRAINT pedidos_direccion_len
    CHECK (char_length(direccion) <= 300);

ALTER TABLE pedidos
    DROP CONSTRAINT IF EXISTS pedidos_ciudad_len;
ALTER TABLE pedidos
    ADD CONSTRAINT pedidos_ciudad_len
    CHECK (char_length(ciudad) <= 120);

COMMIT;