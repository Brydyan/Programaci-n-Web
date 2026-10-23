import EstadoWidget from './EstadoWidget';
import { formatCurrency, formatNumero } from './format';
import type { ReporteResumen } from '../../types/api';

interface Props {
  data: ReporteResumen | null;
  cargando: boolean;
  error: string | null;
}

/**
 * Las cuatro tarjetas KPI del dashboard.
 *
 * Muestran exactamente los campos de `ReporteResumen` — ni uno más. La versión
 * anterior de este componente mostraba "Ventas Totales", "Referidos Activos" y
 * "Nivel Actual" con números inventados ($12,450.00, 24, Diamante): datos que
 * no venían de ningún endpoint y que nadie podía verificar.
 *
 * Sólo muestra el estado de carga la primera vez: si no, un refetch borraría
 * las tarjetas y la pantalla parpadearía a vacío.
 */
export default function ResumenCards({ data, cargando, error }: Props) {
  const esperando = cargando && data === null;

  if (esperando || error || !data) {
    return (
      <EstadoWidget
        cargando={esperando}
        error={error}
        vacio={false}
        etiqueta="el resumen"
      />
    );
  }

  const tarjetas = [
    {
      etiqueta: 'Ventas totales',
      valor: formatCurrency(data.total_vendido),
      detalle: `${formatNumero(data.num_pedidos)} ${
        data.num_pedidos === 1 ? 'pedido' : 'pedidos'
      } (sin cancelados)`,
      acento: 'text-emerald-600',
    },
    {
      etiqueta: 'Ticket promedio',
      valor: formatCurrency(data.ticket_promedio),
      detalle: 'ventas ÷ pedidos',
      acento: 'text-marca-600',
    },
    {
      etiqueta: 'Comisiones devengadas',
      valor: formatCurrency(data.comisiones_devengadas),
      detalle: 'excluye revertidas',
      acento: 'text-amber-600',
    },
    {
      etiqueta: 'Pedidos',
      valor: formatNumero(data.num_pedidos),
      detalle: 'en el periodo filtrado',
      acento: 'text-slate-700',
    },
  ];

  return (
    <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 xl:grid-cols-4">
      {tarjetas.map((t) => (
        <article
          key={t.etiqueta}
          className="rounded-lg border border-slate-100 bg-white p-6 shadow-sm"
        >
          <p className="text-sm font-semibold uppercase tracking-wide text-slate-500">
            {t.etiqueta}
          </p>
          <p className={`mt-2 text-3xl font-bold ${t.acento}`}>{t.valor}</p>
          <p className="mt-1 text-xs text-slate-400">{t.detalle}</p>
        </article>
      ))}
    </div>
  );
}