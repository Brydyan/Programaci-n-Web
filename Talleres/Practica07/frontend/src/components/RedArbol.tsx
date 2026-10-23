import type { NodoRed } from '../types/api';
import { formatCurrency } from './reportes/format';
import type { ResumenRed } from './redUtils';

/**
 * Árbol multinivel, compartido por "Mi Red" (subárbol del usuario) y
 * "Red Global" (árbol completo, admin).
 *
 * Vive aparte porque MiRed.tsx tenía el dibujado dentro del componente y la
 * segunda vista habría copiado cien líneas. Los totales y el filtrado van en
 * redUtils.ts para que este archivo solo exporte componentes y no rompa el
 * Fast Refresh de Vite.
 */

/** Tarjetas de resumen de una rama: tamaño, ventas generadas y profundidad. */
export const ResumenCards = ({ resumen }: { resumen: ResumenRed }) => (
  <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-6">
    <div className="bg-white p-4 rounded-lg border border-slate-200">
      <p className="text-xs uppercase font-semibold text-slate-500">Personas</p>
      <p className="text-2xl font-bold text-indigo-600">{resumen.nodos}</p>
    </div>
    <div className="bg-white p-4 rounded-lg border border-slate-200">
      <p className="text-xs uppercase font-semibold text-slate-500">Ventas generadas</p>
      <p className="text-2xl font-bold text-indigo-600">{formatCurrency(resumen.ventas)}</p>
    </div>
    <div className="bg-white p-4 rounded-lg border border-slate-200">
      <p className="text-xs uppercase font-semibold text-slate-500">Profundidad</p>
      <p className="text-2xl font-bold text-indigo-600">{resumen.nivelMaximo}</p>
    </div>
  </div>
);

/** Una rama del árbol, con líneas de guía. `nivel` es la profundidad visual. */
export const RamaRed = ({ nodos, nivel }: { nodos: NodoRed[]; nivel: number }) => {
  if (nodos.length === 0) {
    return (
      <div className="bg-white rounded-lg border border-dashed border-slate-300 p-6 text-center text-slate-500 text-sm">
        Nadie se ha unido a esta rama todavía.
      </div>
    );
  }

  return (
    <ul className={nivel === 1 ? 'space-y-3' : 'space-y-2 mt-2'}>
      {nodos.map((n) => (
        <li key={n.id}>
          <div
            className={`bg-white rounded-lg border border-slate-200 p-3 shadow-sm ${
              nivel === 1 ? 'border-l-4 border-l-indigo-400' : 'border-l-4 border-l-slate-300 ml-3'
            }`}
          >
            <div className="flex items-center justify-between gap-3">
              <div className="min-w-0">
                <p className="font-semibold text-slate-800 truncate">{n.nombre}</p>
                <p className="text-xs text-slate-500">
                  Nivel {n.nivel} · <span className="font-mono">{n.codigo}</span>
                </p>
              </div>
              <p className="text-sm font-semibold text-indigo-600 shrink-0">
                {formatCurrency(n.ventas ?? 0)}
              </p>
            </div>
          </div>
          {n.hijos && n.hijos.length > 0 && <RamaRed nodos={n.hijos} nivel={nivel + 1} />}
        </li>
      ))}
    </ul>
  );
};