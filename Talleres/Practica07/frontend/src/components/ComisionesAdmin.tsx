import { useCallback, useEffect, useState } from 'react';
import { getComisionesAdmin } from '../services/comisionesService';
import {
  getErrorMessage,
  type Comision,
  type ComisionResumen,
  type EstadoComision,
} from '../types/api';
import { formatCurrency, formatFecha } from './reportes/format';

/**
 * "Auditoría de comisiones" — admin only.
 *
 * El endpoint /api/admin/comisiones devuelve el ledger completo con filtro
 * por estado y por distribuidor. Es la pantalla de soporte: la que se abre
 * cuando alguien reclama que su comisión no le apareció.
 *
 * Un detalle del contrato que conviene no pasar por alto: los `totales` que
 * devuelve la respuesta son SIEMPRE globales, no cambian con el filtro. Sirven
 * justamente para eso — comparar "lo que veo" contra "lo que hay en total" —
 * y por eso la etiqueta lo dice, para que nadie los lea como el subtotal de lo
 * filtrado.
 */

const ESTADOS: EstadoComision[] = ['pendiente', 'aprobada', 'pagada', 'revertida'];

const ESTILO: Record<EstadoComision, { texto: string; clase: string }> = {
  pendiente: { texto: 'Pendiente', clase: 'bg-amber-100 text-amber-800' },
  aprobada: { texto: 'Aprobada', clase: 'bg-sky-100 text-sky-800' },
  pagada: { texto: 'Pagada', clase: 'bg-emerald-100 text-emerald-800' },
  revertida: { texto: 'Revertida', clase: 'bg-slate-200 text-slate-600' },
};

const ComisionesAdmin = () => {
  const [filtroEstado, setFiltroEstado] = useState<EstadoComision | ''>('');
  const [referidoId, setReferidoId] = useState('');
  const [pagina, setPagina] = useState(1);
  const [datos, setDatos] = useState<ComisionResumen | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const cargar = useCallback(async () => {
    setError(null);
    try {
      setDatos(
        await getComisionesAdmin({
          estado: filtroEstado || undefined,
          referido_id: referidoId ? Number(referidoId) : undefined,
          page: pagina,
        }),
      );
    } catch (err) {
      setError(getErrorMessage(err));
      setDatos(null);
    } finally {
      setIsLoading(false);
    }
  }, [filtroEstado, referidoId, pagina]);

  useEffect(() => {
    let vivo = true;
    void (async () => {
      try {
        const d = await getComisionesAdmin({
          estado: filtroEstado || undefined,
          referido_id: referidoId ? Number(referidoId) : undefined,
          page: pagina,
        });
        if (vivo) setDatos(d);
      } catch (err) {
        if (!vivo) return;
        setError(getErrorMessage(err));
        setDatos(null);
      } finally {
        if (vivo) setIsLoading(false);
      }
    })();
    return () => {
      vivo = false;
    };
  }, [filtroEstado, referidoId, pagina]);

  // Cambiar un filtro vuelve a la primera página: si se estaba en la 3 con
  // 20 filas y se pasa a un filtro que deja 3 resultados, quedarse en la 3
  // muestra una tabla vacía sin explicación.
  const setFiltro = (v: EstadoComision | '') => {
    setFiltroEstado(v);
    setPagina(1);
    setIsLoading(true);
  };
  const setReferido = (v: string) => {
    setReferidoId(v);
    setPagina(1);
    setIsLoading(true);
  };

  if (isLoading && !datos) {
    return (
      <div className="flex justify-center items-center min-h-96">
        <p className="text-slate-600 font-medium">Cargando comisiones...</p>
      </div>
    );
  }

  const filas = datos?.data ?? [];
  const ultima = !datos || datos.total <= datos.limit;
  const primera = datos ? (datos.page - 1) * datos.limit + 1 : 0;
  const ultimaFila = datos ? Math.min(datos.page * datos.limit, datos.total) : 0;

  return (
    <div>
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between mb-6 gap-4">
        <div>
          <h1 className="text-2xl font-bold text-slate-800">Auditoría de comisiones</h1>
          <p className="text-sm text-slate-500">Ledger completo de la organización</p>
        </div>
        <button
          onClick={cargar}
          className="text-xs bg-indigo-50 text-indigo-600 hover:bg-indigo-100 px-3 py-2 rounded-md font-medium transition self-start min-h-[44px]"
        >
          Actualizar
        </button>
      </div>

      {error && (
        <div className="bg-red-50 text-red-600 p-4 rounded-lg border border-red-200 mb-6">
          <p className="text-sm font-medium">{error}</p>
        </div>
      )}

      {/* Los totales son globales: el backend no los recalcula con el filtro.
          Por eso la etiqueta lo dice y no se presenta como subtotal. */}
      {datos && datos.totales.length > 0 && (
        <div className="bg-white rounded-lg border border-slate-200 p-4 mb-6 shadow-sm">
          <p className="text-xs font-semibold uppercase text-slate-500 mb-1">
            Totales de toda la organización
          </p>
          <p className="text-xs text-slate-400 mb-3">
            No cambian con el filtro de abajo: sirven para comparar.
          </p>
          <div className="flex flex-wrap gap-2">
            {datos.totales.map((t) => {
              const est = ESTILO[t.estado];
              return (
                <div key={t.estado} className={`rounded-md px-3 py-2 ${est.clase}`}>
                  <p className="text-xs font-medium">{est.texto}</p>
                  <p className="text-sm font-bold">{formatCurrency(t.total)}</p>
                  <p className="text-xs opacity-75">{t.cantidad} comisiones</p>
                </div>
              );
            })}
          </div>
        </div>
      )}

      <div className="bg-white rounded-lg border border-slate-200 p-4 mb-6 shadow-sm">
        <div className="flex flex-col sm:flex-row sm:items-end gap-3 mb-4">
          <div className="flex flex-wrap gap-2">
            <button
              onClick={() => setFiltro('')}
              aria-pressed={filtroEstado === ''}
              className={`text-xs px-3 py-1.5 rounded-md font-medium border transition min-h-[36px] ${
                filtroEstado === ''
                  ? 'bg-indigo-600 text-white border-indigo-600'
                  : 'bg-white text-slate-600 border-slate-300 hover:bg-slate-50'
              }`}
            >
              Todos
            </button>
            {ESTADOS.map((e) => (
              <button
                key={e}
                onClick={() => setFiltro(e)}
                aria-pressed={filtroEstado === e}
                className={`text-xs px-3 py-1.5 rounded-md font-medium border transition min-h-[36px] ${
                  filtroEstado === e
                    ? 'bg-indigo-600 text-white border-indigo-600'
                    : 'bg-white text-slate-600 border-slate-300 hover:bg-slate-50'
                }`}
              >
                {ESTILO[e].texto}
              </button>
            ))}
          </div>

          <div>
            <label
              htmlFor="filtro-referido"
              className="block text-xs font-medium text-slate-600 mb-1"
            >
              Distribuidor (id)
            </label>
            <input
              id="filtro-referido"
              type="number"
              value={referidoId}
              onChange={(e) => setReferido(e.target.value)}
              placeholder="Todos"
              className="w-32 px-3 py-2 border border-slate-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 min-h-[44px]"
            />
          </div>
        </div>

        {filas.length === 0 ? (
          <p className="text-sm text-slate-500">No hay comisiones con este filtro.</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead className="border-b border-slate-200">
                <tr>
                  <th className="py-2 pr-3 font-semibold text-slate-600">#</th>
                  <th className="py-2 pr-3 font-semibold text-slate-600">Pedido</th>
                  <th className="py-2 pr-3 font-semibold text-slate-600">Distribuidor</th>
                  <th className="py-2 pr-3 font-semibold text-slate-600">Nivel</th>
                  <th className="py-2 pr-3 font-semibold text-slate-600">%</th>
                  <th className="py-2 pr-3 font-semibold text-slate-600">Monto</th>
                  <th className="py-2 pr-3 font-semibold text-slate-600">Fecha</th>
                  <th className="py-2 font-semibold text-slate-600">Estado</th>
                </tr>
              </thead>
              <tbody>
                {filas.map((c: Comision) => {
                  const est = ESTILO[c.estado];
                  return (
                    <tr key={c.id} className="border-b border-slate-100">
                      <td className="py-2 pr-3 text-slate-400">{c.id}</td>
                      <td className="py-2 pr-3 text-slate-700">#{c.pedido_id ?? '—'}</td>
                      <td className="py-2 pr-3 text-slate-700">{c.referido_id ?? '—'}</td>
                      <td className="py-2 pr-3 text-slate-500">{c.nivel}</td>
                      <td className="py-2 pr-3 text-slate-500">{c.porcentaje}%</td>
                      <td className="py-2 pr-3 font-semibold text-slate-800">
                        {formatCurrency(c.monto)}
                      </td>
                      <td className="py-2 pr-3 text-slate-500">{formatFecha(c.creado_en)}</td>
                      <td className="py-2">
                        <span className={`rounded px-2 py-0.5 text-xs font-medium ${est.clase}`}>
                          {est.texto}
                        </span>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}

        {datos && datos.total > 0 && (
          <div className="flex items-center justify-between mt-4 text-sm">
            <p className="text-slate-500">
              {primera}–{ultimaFila} de {datos.total}
            </p>
            <div className="flex gap-2">
              <button
                onClick={() => setPagina(Math.max(1, pagina - 1))}
                disabled={pagina <= 1}
                className="px-3 py-1.5 rounded-md border border-slate-300 text-slate-600 disabled:opacity-40 hover:bg-slate-50 transition min-h-[36px]"
              >
                Anterior
              </button>
              <button
                onClick={() => setPagina(pagina + 1)}
                disabled={ultima}
                className="px-3 py-1.5 rounded-md border border-slate-300 text-slate-600 disabled:opacity-40 hover:bg-slate-50 transition min-h-[36px]"
              >
                Siguiente
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default ComisionesAdmin;