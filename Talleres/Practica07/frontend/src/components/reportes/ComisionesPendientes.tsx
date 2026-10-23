import EstadoWidget from './EstadoWidget';
import Tarjeta from './Tarjeta';
import { formatCurrency, formatFecha } from './format';
import type { Comision } from '../../types/api';

interface Props {
  data: Comision[] | null;
  cargando: boolean;
  error: string | null;
}

const NIVEL_LABEL = (n: number): string => (n === 1 ? 'Directo' : `Nivel ${n}`);

/**
 * Ledger pendiente de liquidar.
 *
 * Es la lista de trabajo del admin antes de pulsar "liquidar": qué se debe y a
 * quién. Por eso el backend la limita a `estado = 'pendiente'` — mostrar aquí
 * una comisión ya pagada haría que el admin intente liquidar dos veces.
 *
 * El total en cabecera es la suma de lo que hay en pantalla, no la deuda total
 * de la organización: si el endpoint trajera 50 filas de un total de 500, el
 * número mentía. Se rotula como "en pantalla" por eso.
 */
export default function ComisionesPendientes({ data, cargando, error }: Props) {
  const filas = data ?? [];
  const totalEnPantalla = filas.reduce((suma, c) => suma + c.monto, 0);

  return (
    <Tarjeta
      titulo="Comisiones pendientes de liquidar"
      acciones={
        filas.length > 0 ? (
          <span className="text-sm text-slate-500">
            {formatCurrency(totalEnPantalla)} en pantalla
          </span>
        ) : undefined
      }
    >
      <EstadoWidget
        cargando={cargando && filas.length === 0}
        error={error}
        vacio={!cargando && !error && filas.length === 0}
        etiqueta="comisiones pendientes"
      />
      {filas.length > 0 && (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-slate-100 text-left text-xs uppercase tracking-wide text-slate-400">
                <th className="pb-2 font-semibold">Fecha</th>
                <th className="pb-2 font-semibold">Distribuidor</th>
                <th className="pb-2 font-semibold">Nivel</th>
                <th className="pb-2 text-right font-semibold">%</th>
                <th className="pb-2 text-right font-semibold">Monto</th>
              </tr>
            </thead>
            <tbody>
              {filas.map((c) => (
                <tr
                  key={c.id}
                  className="border-b border-slate-50 last:border-0"
                >
                  <td className="py-2 whitespace-nowrap text-slate-500">
                    {formatFecha(c.creado_en)}
                  </td>
                  <td className="py-2 text-slate-700">
                    {c.referido_id ? `#${c.referido_id}` : '—'}
                    {c.pedido_id ? (
                      <span className="ml-2 text-xs text-slate-400">
                        pedido #{c.pedido_id}
                      </span>
                    ) : null}
                  </td>
                  <td className="py-2 text-slate-600">{NIVEL_LABEL(c.nivel)}</td>
                  <td className="py-2 text-right tabular-nums text-slate-600">
                    {c.porcentaje.toFixed(2)}%
                  </td>
                  <td className="py-2 text-right font-medium tabular-nums text-slate-800">
                    {formatCurrency(c.monto)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </Tarjeta>
  );
}