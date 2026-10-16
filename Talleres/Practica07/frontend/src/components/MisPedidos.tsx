import { useEffect, useMemo, useState } from 'react';
import { MdExpandMore, MdExpandLess, MdRefresh } from 'react-icons/md';
import {
  getPedidoDetalle,
  listMisPedidos,
  type MisPedidosQuery,
} from '../services/pedidosService';
import type { PedidoDetalle, PedidoResumen } from '../types/api';
import { ApiError, type Pedido } from '../types/api';

type Estado = Pedido['estado'];

const ESTADO_PILL: Record<Estado, string> = {
  pendiente: 'bg-slate-200 text-slate-700',
  preparando: 'bg-amber-100 text-amber-700',
  enviado: 'bg-blue-100 text-blue-700',
  entregado: 'bg-emerald-100 text-emerald-700',
  cancelado: 'bg-rose-100 text-rose-700',
};

const formatCurrency = (n: number): string => `$${n.toFixed(2)}`;
const formatFecha = (iso: string): string => {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleDateString('es-EC', { year: 'numeric', month: 'short', day: '2-digit' });
};

const MisPedidos = () => {
  const [pedidos, setPedidos] = useState<PedidoResumen[]>([]);
  const [page, setPage] = useState<number>(1);
  const [limit, setLimit] = useState<number>(20);
  const [total, setTotal] = useState<number>(0);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  const [expanded, setExpanded] = useState<Set<number>>(new Set());
  const [detalles, setDetalles] = useState<Map<number, PedidoDetalle>>(new Map());
  const [detalleLoading, setDetalleLoading] = useState<Set<number>>(new Set());
  const [detalleError, setDetalleError] = useState<Map<number, string>>(new Map());

  const totalPages = useMemo<number>(
    () => (total > 0 ? Math.max(1, Math.ceil(total / limit)) : 1),
    [total, limit],
  );

  const load = async (next?: Partial<MisPedidosQuery>) => {
    setError(null);
    setLoading(true);
    try {
      const res = await listMisPedidos({
        page: next?.page ?? page,
        limit: next?.limit ?? limit,
      });
      setPedidos(res.data);
      setTotal(res.total);
      setPage(res.page);
      setLimit(res.limit);
      // limpia caché de detalles que ya no están visibles
      setDetalles((m) => {
        const keep = new Set(res.data.map((p) => p.id));
        const next2 = new Map<number, PedidoDetalle>();
        m.forEach((v, k) => {
          if (keep.has(k)) next2.set(k, v);
        });
        return next2;
      });
    } catch (err) {
      if (err instanceof ApiError) setError(err.message);
      else setError(err instanceof Error ? err.message : 'Error desconocido');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    /* eslint-disable react-hooks/set-state-in-effect */
    void load();
    /* eslint-enable react-hooks/set-state-in-effect */
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const toggleExpand = async (id: number) => {
    const isOpen = expanded.has(id);
    const next = new Set(expanded);
    if (isOpen) next.delete(id);
    else next.add(id);
    setExpanded(next);

    if (!isOpen && !detalles.has(id)) {
      setDetalleLoading((s) => new Set(s).add(id));
      setDetalleError((m) => {
        const m2 = new Map(m);
        m2.delete(id);
        return m2;
      });
      try {
        const detalle = await getPedidoDetalle(id);
        setDetalles((m) => new Map(m).set(id, detalle));
      } catch (err) {
        const msg =
          err instanceof ApiError
            ? err.message
            : err instanceof Error
              ? err.message
              : 'Error desconocido';
        setDetalleError((m) => new Map(m).set(id, msg));
      } finally {
        setDetalleLoading((s) => {
          const s2 = new Set(s);
          s2.delete(id);
          return s2;
        });
      }
    }
  };

  return (
    <div className="space-y-4 md:space-y-6">
      {/* Top bar */}
      <div className="flex flex-col gap-3">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
          <div>
            <h1 className="text-2xl font-bold text-slate-800">Mis Pedidos</h1>
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

        <div className="flex items-center justify-end gap-2 text-sm text-slate-600">
          <span>Mostrar</span>
          <select
            value={limit}
            onChange={(e) => {
              setLimit(Number(e.target.value));
              setPage(1);
            }}
            className="min-h-[36px] px-2 py-1 rounded border border-slate-300 bg-white focus:ring-2 focus:ring-indigo-600 focus:border-indigo-600 outline-none"
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
          <div className="inline-block w-6 h-6 border-2 border-indigo-600 border-t-transparent rounded-full animate-spin" />
          <p className="mt-2 text-sm">Cargando pedidos...</p>
        </div>
      )}

      {/* Empty */}
      {!loading && !error && pedidos.length === 0 && (
        <div className="bg-white border border-slate-200 rounded-lg p-10 text-center">
          <div className="text-5xl mb-3">📦</div>
          <p className="text-slate-700 font-semibold mb-1">Aún no has hecho pedidos.</p>
          <p className="text-sm text-slate-500">¡Explora el catálogo!</p>
        </div>
      )}

      {/* Cards */}
      {!loading && pedidos.length > 0 && (
        <div className="space-y-3">
          {pedidos.map((p) => {
            const isOpen = expanded.has(p.id);
            const cargandoDetalle = detalleLoading.has(p.id);
            const errorDetalle = detalleError.get(p.id);
            const detalle = detalles.get(p.id);
            return (
              <div
                key={p.id}
                className="bg-white border border-slate-200 rounded-lg overflow-hidden"
              >
                <div className="p-4 flex items-center justify-between gap-3">
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2 flex-wrap">
                      <span className="text-lg font-bold text-slate-800">#{p.id}</span>
                      <span
                        className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-semibold capitalize ${
                          ESTADO_PILL[p.estado]
                        }`}
                      >
                        {p.estado}
                      </span>
                    </div>
                    <p className="text-sm text-slate-500 mt-1">{formatFecha(p.created_at)}</p>
                  </div>
                  <div className="flex flex-col items-end gap-2">
                    <span className="text-lg font-bold text-slate-800 whitespace-nowrap">
                      {formatCurrency(p.total)}
                    </span>
                    <button
                      type="button"
                      onClick={() => void toggleExpand(p.id)}
                      className="inline-flex items-center gap-1 text-sm text-indigo-600 hover:text-indigo-700 font-semibold"
                      aria-expanded={isOpen}
                      aria-controls={`detalle-${p.id}`}
                    >
                      {isOpen ? (
                        <>
                          <MdExpandLess className="text-lg" />
                          Ocultar
                        </>
                      ) : (
                        <>
                          <MdExpandMore className="text-lg" />
                          Ver detalle
                        </>
                      )}
                    </button>
                  </div>
                </div>

                {isOpen && (
                  <div
                    id={`detalle-${p.id}`}
                    className="border-t border-slate-100 bg-slate-50 p-4"
                  >
                    {cargandoDetalle && (
                      <div className="text-center text-slate-500 text-sm py-3">
                        <span className="inline-block w-5 h-5 border-2 border-indigo-600 border-t-transparent rounded-full animate-spin mr-2" />
                        Cargando detalle...
                      </div>
                    )}
                    {errorDetalle && !cargandoDetalle && (
                      <div className="text-sm text-red-600">{errorDetalle}</div>
                    )}
                    {detalle && !cargandoDetalle && (
                      <div className="space-y-4">
                        <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 text-sm">
                          <div>
                            <span className="text-slate-500">Método de pago: </span>
                            <span className="font-semibold text-slate-800">
                              {detalle.metodo_pago}
                            </span>
                          </div>
                          <div>
                            <span className="text-slate-500">Cliente: </span>
                            <span className="font-semibold text-slate-800 break-all">
                              {detalle.email_usuario}
                            </span>
                          </div>
                        </div>

                        {detalle.items && detalle.items.length > 0 && (
                          <div>
                            <p className="text-xs uppercase tracking-wide text-slate-500 font-semibold mb-2">
                              Ítems
                            </p>
                            <ul className="space-y-2">
                              {detalle.items.map((it, idx) => (
                                <li
                                  key={`${it.producto_id}-${idx}`}
                                  className="flex items-center justify-between gap-3 bg-white border border-slate-200 rounded p-2 text-sm"
                                >
                                  <div className="min-w-0 flex-1">
                                    <p className="font-medium text-slate-800 truncate">
                                      {it.nombre}
                                    </p>
                                    <p className="text-xs text-slate-500">
                                      {it.cantidad} × {formatCurrency(it.precio)}
                                    </p>
                                  </div>
                                  <span className="font-semibold text-slate-800 whitespace-nowrap">
                                    {formatCurrency(it.cantidad * it.precio)}
                                  </span>
                                </li>
                              ))}
                            </ul>
                          </div>
                        )}
                      </div>
                    )}
                  </div>
                )}
              </div>
            );
          })}
        </div>
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

export default MisPedidos;
