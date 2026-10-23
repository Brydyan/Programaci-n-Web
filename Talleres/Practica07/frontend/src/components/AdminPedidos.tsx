import { useEffect, useMemo, useState } from 'react';
import { MdRefresh } from 'react-icons/md';
import {
  cambiarEstadoPedido,
  listPedidosAdmin,
  type AdminPedidosQuery,
  type EstadoPedido,
} from '../services/pedidosService';
import {
  ApiError,
  ESTADOS_PEDIDO,
  TRANSICIONES_VALIDAS,
  type Pedido,
  type PedidoResumen,
} from '../types/api';
import { BadgeEstado } from './Badge';
import { ESTADO_PEDIDO } from './tokens';
import { formatCurrency, formatFecha } from './reportes/format';

type Estado = Pedido['estado'];

const ESTADOS_FILTRO: Array<{ value: '' | Estado; label: string }> = [
  { value: '', label: 'Todos' },
  { value: 'pendiente', label: 'Pendiente' },
  { value: 'preparando', label: 'Preparando' },
  { value: 'enviado', label: 'Enviado' },
  { value: 'entregado', label: 'Entregado' },
  { value: 'cancelado', label: 'Cancelado' },
];


const isEstado = (s: string): s is Estado =>
  (ESTADOS_PEDIDO as readonly string[]).includes(s);

const AdminPedidos = () => {
  const [pedidos, setPedidos] = useState<PedidoResumen[]>([]);
  const [page, setPage] = useState<number>(1);
  const [limit, setLimit] = useState<number>(20);
  const [total, setTotal] = useState<number>(0);
  const [estadoFiltro, setEstadoFiltro] = useState<'' | Estado>('');

  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  const [busyId, setBusyId] = useState<number | null>(null);

  const totalPages = useMemo<number>(
    () => (total > 0 ? Math.max(1, Math.ceil(total / limit)) : 1),
    [total, limit],
  );

  const load = async (next?: Partial<AdminPedidosQuery>) => {
    setError(null);
    setLoading(true);
    try {
      const params: AdminPedidosQuery = {
        estado: next?.estado !== undefined ? next.estado : estadoFiltro || undefined,
        page: next?.page ?? page,
        limit: next?.limit ?? limit,
      };
      const res = await listPedidosAdmin(params);
      setPedidos(res.data);
      setTotal(res.total);
      setPage(res.page);
      setLimit(res.limit);
    } catch (err) {
      if (err instanceof ApiError) setError(err.message);
      else setError(err instanceof Error ? err.message : 'Error desconocido');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    /* eslint-disable react-hooks/set-state-in-effect */
    void load({ page: 1 });
    /* eslint-enable react-hooks/set-state-in-effect */
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [estadoFiltro, limit]);

  useEffect(() => {
    /* eslint-disable react-hooks/set-state-in-effect */
    if (page > 1) {
      void load({ page });
    } else {
      void load({ page: 1 });
    }
    /* eslint-enable react-hooks/set-state-in-effect */
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page]);

  const handleCambiarEstado = async (p: PedidoResumen, nuevo: Estado) => {
    if (nuevo === p.estado) return;
    setBusyId(p.id);
    try {
      await cambiarEstadoPedido(p.id, nuevo as EstadoPedido);
      // actualiza local sin re-fetch completo
      setPedidos((arr) =>
        arr.map((it) => (it.id === p.id ? { ...it, estado: nuevo } : it)),
      );
    } catch (err) {
      const msg =
        err instanceof ApiError
          ? err.message
          : err instanceof Error
            ? err.message
            : 'No se pudo cambiar el estado';
      setError(msg);
    } finally {
      setBusyId(null);
    }
  };

  return (
    <div className="space-y-4 md:space-y-6">
      {/* Top bar */}
      <div className="flex flex-col gap-3">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
          <div>
            <h1 className="text-2xl font-bold text-slate-800">Pedidos</h1>
            <p className="text-sm text-slate-500 mt-1">
              Total: <span className="font-semibold text-slate-700">{total}</span>
            </p>
          </div>
          <button
            type="button"
            onClick={() => void load({ page })}
            className="inline-flex items-center gap-1 min-h-[44px] px-4 py-2 rounded-lg border border-slate-300 bg-white text-slate-700 font-semibold hover:bg-slate-50 transition"
            aria-label="Recargar pedidos"
          >
            <MdRefresh className="text-lg" />
            Recargar
          </button>
        </div>

        <div className="flex flex-col sm:flex-row gap-3">
          <select
            value={estadoFiltro}
            onChange={(e) => {
              const v = e.target.value;
              setEstadoFiltro(isEstado(v) ? v : '');
              setPage(1);
            }}
            className="min-h-[44px] px-3 py-2 rounded-lg border border-slate-300 bg-white focus:ring-2 focus:ring-marca-600 focus:border-marca-600 outline-none transition"
            aria-label="Filtrar por estado"
          >
            {ESTADOS_FILTRO.map((e) => (
              <option key={e.value} value={e.value}>
                {e.label}
              </option>
            ))}
          </select>
          <select
            value={limit}
            onChange={(e) => {
              setLimit(Number(e.target.value));
              setPage(1);
            }}
            className="min-h-[44px] px-3 py-2 rounded-lg border border-slate-300 bg-white focus:ring-2 focus:ring-marca-600 focus:border-marca-600 outline-none transition"
            aria-label="Límite por página"
          >
            <option value={10}>10</option>
            <option value={20}>20</option>
            <option value={50}>50</option>
          </select>
        </div>
      </div>

      {/* Error */}
      {error && (
        <div className="bg-red-50 text-red-700 border border-red-200 p-3 rounded-lg flex flex-col sm:flex-row sm:items-center sm:justify-between gap-2">
          <span className="text-sm">{error}</span>
          <button
            type="button"
            onClick={() => void load({ page: 1 })}
            className="min-h-[36px] px-3 py-1.5 rounded-md bg-red-600 text-white text-sm font-semibold hover:bg-red-700 transition"
          >
            Reintentar
          </button>
        </div>
      )}

      {/* Loading */}
      {loading && (
        <div className="bg-white border border-slate-200 rounded-lg p-8 text-center text-slate-500">
          <div className="inline-block w-6 h-6 border-2 border-marca-600 border-t-transparent rounded-full animate-spin" />
          <p className="mt-2 text-sm">Cargando pedidos...</p>
        </div>
      )}

      {/* Empty */}
      {!loading && !error && pedidos.length === 0 && (
        <div className="bg-white border border-slate-200 rounded-lg p-8 text-center text-slate-500">
          No hay pedidos que coincidan con el filtro.
        </div>
      )}

      {/* Tabla desktop */}
      {!loading && pedidos.length > 0 && (
        <>
          <div className="hidden md:block bg-white border border-slate-200 rounded-lg overflow-hidden">
            <table className="w-full text-sm">
              <thead className="bg-slate-50 text-slate-600 uppercase text-xs">
                <tr>
                  <th className="text-left px-4 py-3">#ID</th>
                  <th className="text-left px-4 py-3">Cliente</th>
                  <th className="text-left px-4 py-3">Fecha</th>
                  <th className="text-right px-4 py-3">Total</th>
                  <th className="text-left px-4 py-3">Estado</th>
                  <th className="text-right px-4 py-3">Acciones</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {pedidos.map((p) => {
                  const transiciones = TRANSICIONES_VALIDAS[p.estado];
                  return (
                    <tr key={p.id} className="hover:bg-slate-50">
                      <td className="px-4 py-3 font-bold text-slate-800">#{p.id}</td>
                      <td className="px-4 py-3 text-slate-700 break-all">
                        {p.email_usuario ?? (
                          <span className="text-slate-400 text-xs">cliente #{p.id}</span>
                        )}
                      </td>
                      <td className="px-4 py-3 text-slate-600">{formatFecha(p.created_at)}</td>
                      <td className="px-4 py-3 text-right text-slate-800">
                        {formatCurrency(p.total)}
                      </td>
                      <td className="px-4 py-3">
                        <BadgeEstado definicion={ESTADO_PEDIDO[p.estado]} />
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex items-center justify-end">
                          {transiciones.length === 0 ? (
                            <span className="text-xs text-slate-400 italic">
                              Estado final
                            </span>
                          ) : (
                            <select
                              value=""
                              disabled={busyId === p.id}
                              onChange={(e) => {
                                const v = e.target.value;
                                if (v && isEstado(v)) {
                                  void handleCambiarEstado(p, v);
                                }
                                e.currentTarget.value = '';
                              }}
                              className="min-h-[36px] px-2 py-1 rounded border border-slate-300 bg-white text-sm focus:ring-2 focus:ring-marca-600 focus:border-marca-600 outline-none disabled:opacity-50"
                              aria-label={`Cambiar estado del pedido ${p.id}`}
                            >
                              <option value="">Cambiar a…</option>
                              {transiciones.map((t) => (
                                <option key={t} value={t}>
                                  {ESTADO_PEDIDO[t].texto}
                                </option>
                              ))}
                            </select>
                          )}
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>

          {/* Mobile cards */}
          <div className="md:hidden space-y-3">
            {pedidos.map((p) => {
              const transiciones = TRANSICIONES_VALIDAS[p.estado];
              return (
                <div
                  key={p.id}
                  className="bg-white border border-slate-200 rounded-lg p-4"
                >
                  <div className="flex items-start justify-between gap-2">
                    <div className="min-w-0">
                      <p className="text-lg font-bold text-slate-800">#{p.id}</p>
                      <p className="text-xs text-slate-500">{formatFecha(p.created_at)}</p>
                      <p className="text-base font-semibold text-slate-800 mt-1">
                        {formatCurrency(p.total)}
                      </p>
                    </div>
                    <BadgeEstado definicion={ESTADO_PEDIDO[p.estado]} />
                  </div>
                  <div className="mt-3">
                    {transiciones.length === 0 ? (
                      <span className="text-xs text-slate-400 italic">
                        Estado final
                      </span>
                    ) : (
                      <select
                        value=""
                        disabled={busyId === p.id}
                        onChange={(e) => {
                          const v = e.target.value;
                          if (v && isEstado(v)) {
                            void handleCambiarEstado(p, v);
                          }
                          e.currentTarget.value = '';
                        }}
                        className="w-full min-h-[44px] px-3 py-2 rounded border border-slate-300 bg-white text-sm focus:ring-2 focus:ring-marca-600 focus:border-marca-600 outline-none disabled:opacity-50"
                        aria-label={`Cambiar estado del pedido ${p.id}`}
                      >
                        <option value="">Cambiar a…</option>
                        {transiciones.map((t) => (
                          <option key={t} value={t}>
                            {ESTADO_PEDIDO[t].texto}
                          </option>
                        ))}
                      </select>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        </>
      )}

      {/* Pagination */}
      {!loading && !error && total > 0 && (
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 bg-white border border-slate-200 rounded-lg p-3">
          <div className="flex items-center justify-center gap-2 text-sm">
            <button
              type="button"
              onClick={() => setPage(1)}
              disabled={page <= 1}
              className="min-h-[36px] min-w-[36px] px-2 rounded border border-slate-300 hover:bg-slate-50 disabled:opacity-40 disabled:cursor-not-allowed"
              aria-label="Primera página"
            >
              «
            </button>
            <button
              type="button"
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              disabled={page <= 1}
              className="min-h-[36px] min-w-[36px] px-2 rounded border border-slate-300 hover:bg-slate-50 disabled:opacity-40 disabled:cursor-not-allowed"
              aria-label="Página anterior"
            >
              ‹
            </button>
            <span className="text-slate-600 px-2">
              Página <strong className="text-slate-800">{page}</strong> de{' '}
              <strong className="text-slate-800">{totalPages}</strong>
            </span>
            <button
              type="button"
              onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
              disabled={page >= totalPages}
              className="min-h-[36px] min-w-[36px] px-2 rounded border border-slate-300 hover:bg-slate-50 disabled:opacity-40 disabled:cursor-not-allowed"
              aria-label="Página siguiente"
            >
              ›
            </button>
            <button
              type="button"
              onClick={() => setPage(totalPages)}
              disabled={page >= totalPages}
              className="min-h-[36px] min-w-[36px] px-2 rounded border border-slate-300 hover:bg-slate-50 disabled:opacity-40 disabled:cursor-not-allowed"
              aria-label="Última página"
            >
              »
            </button>
          </div>
          <div className="text-sm text-slate-600 text-center">
            {total} pedido{total === 1 ? '' : 's'}
          </div>
        </div>
      )}
    </div>
  );
};

export default AdminPedidos;
