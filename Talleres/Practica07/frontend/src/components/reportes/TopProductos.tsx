import EstadoWidget from './EstadoWidget';
import Tarjeta from './Tarjeta';
import { anchoBarra, formatCurrency, formatNumero } from './format';
import type { TopProducto } from '../../types/api';

interface Props {
  data: TopProducto[] | null;
  cargando: boolean;
  error: string | null;
}

/**
 * Ranking de productos por ingresos, en barras CSS.
 *
 * Barras CSS y no recharts a propósito: el proyecto no tiene esa dependencia
 * y meterla por dos gráficos añade ~500 KB por algo que un `div` con un width
 * porcentual ya resuelve. Si el requerimiento pide interactividad (hover con
 * tooltip, zoom), ahí sí vale la pena.
 *
 * `nombre` sale de `pedido_items`, la foto del nombre en el momento de la
 * compra, no del catálogo actual.
 */
export default function TopProductos({ data, cargando, error }: Props) {
  const filas = data ?? [];
  const maxIngresos = filas.reduce((max, f) => Math.max(max, f.ingresos), 0);

  return (
    <Tarjeta titulo="Top productos por ingresos">
      <EstadoWidget
        cargando={cargando && filas.length === 0}
        error={error}
        vacio={!cargando && !error && filas.length === 0}
        etiqueta="ventas de productos"
      />
      {filas.length > 0 && (
        <ul className="space-y-3">
          {filas.map((f) => (
            <li key={f.producto_id}>
              <div className="flex items-baseline justify-between gap-4 text-sm">
                <span className="truncate font-medium text-slate-700">{f.nombre}</span>
                <span className="shrink-0 tabular-nums text-slate-500">
                  {formatCurrency(f.ingresos)}
                  <span className="ml-2 text-xs text-slate-400">
                    {formatNumero(f.unidades)} u.
                  </span>
                </span>
              </div>
              <div className="mt-1 h-2 w-full overflow-hidden rounded-full bg-slate-100">
                <div
                  className="h-full rounded-full bg-marca-500"
                  style={{ width: `${anchoBarra(f.ingresos, maxIngresos)}%` }}
                />
              </div>
            </li>
          ))}
        </ul>
      )}
    </Tarjeta>
  );
}