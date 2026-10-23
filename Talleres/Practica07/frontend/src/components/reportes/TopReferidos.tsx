import EstadoWidget from './EstadoWidget';
import Tarjeta from './Tarjeta';
import { anchoBarra, formatCurrency } from './format';
import type { TopReferido } from '../../types/api';

interface Props {
  data: TopReferido[] | null;
  cargando: boolean;
  error: string | null;
}

/**
 * Ranking de distribuidores por ventas, en barras CSS.
 *
 * `ventas` viene del contador acumulado `referidos.ventas` y `comisiones` del
 * ledger real. No tienen por qué cuadrar: el contador se actualiza en una
 * goroutine tras el commit del checkout, así que va un instante por detrás.
 * Se muestran las dos columnas precisamente para que la diferencia sea
 * visible y no parezca un bug.
 */
export default function TopReferidos({ data, cargando, error }: Props) {
  const filas = data ?? [];
  const maxVentas = filas.reduce((max, f) => Math.max(max, f.ventas), 0);

  return (
    <Tarjeta titulo="Top distribuidores por ventas">
      <EstadoWidget
        cargando={cargando && filas.length === 0}
        error={error}
        vacio={!cargando && !error && filas.length === 0}
        etiqueta="distribuidores"
      />
      {filas.length > 0 && (
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-slate-100 text-left text-xs uppercase tracking-wide text-slate-400">
              <th className="pb-2 font-semibold">Distribuidor</th>
              <th className="pb-2 text-right font-semibold">Ventas</th>
              <th className="pb-2 text-right font-semibold">Comisiones</th>
            </tr>
          </thead>
          <tbody>
            {filas.map((f) => (
              <tr key={f.referido_id} className="border-b border-slate-50 last:border-0">
                <td className="py-2">
                  <span className="font-medium text-slate-700">{f.nombre}</span>
                  <span className="ml-2 text-xs text-slate-400">{f.codigo}</span>
                  <div className="mt-1 h-1.5 w-full max-w-[220px] overflow-hidden rounded-full bg-slate-100">
                    <div
                      className="h-full rounded-full bg-emerald-500"
                      style={{ width: `${anchoBarra(f.ventas, maxVentas)}%` }}
                    />
                  </div>
                </td>
                <td className="py-2 text-right tabular-nums text-slate-600">
                  {formatCurrency(f.ventas)}
                </td>
                <td className="py-2 text-right tabular-nums text-slate-600">
                  {formatCurrency(f.comisiones)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </Tarjeta>
  );
}