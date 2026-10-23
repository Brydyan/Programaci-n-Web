import { useEffect, useMemo, useState } from 'react';
import { MdEdit, MdToggleOn, MdToggleOff, MdDelete } from 'react-icons/md';
import {
  listProductosAdmin,
  softDeleteProducto,
  updateProducto,
  type AdminProductosQuery,
} from '../services/adminProductosService';
import type { Pagina, Producto } from '../types/api';
import { ApiError } from '../types/api';
import { Badge } from './Badge';
import { useDebouncedValue } from '../hooks/useDebouncedValue';
import ProductoForm from './ProductoForm';

type ActivoFilter = 'all' | 'true' | 'false';

const AdminProductos = () => {
  const [productos, setProductos] = useState<Producto[]>([]);
  const [page, setPage] = useState<number>(1);
  const [limit, setLimit] = useState<number>(20);
  const [total, setTotal] = useState<number>(0);
  const [q, setQ] = useState<string>('');
  const [activoFilter, setActivoFilter] = useState<ActivoFilter>('all');

  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  const [editing, setEditing] = useState<Producto | null>(null);
  const [creating, setCreating] = useState<boolean>(false);
  const [busyId, setBusyId] = useState<number | null>(null);

  const debouncedQ = useDebouncedValue<string>(q, 300);

  const totalPages = useMemo<number>(
    () => (total > 0 ? Math.max(1, Math.ceil(total / limit)) : 1),
    [total, limit],
  );

  const load = async (next?: Partial<AdminProductosQuery>) => {
    setError(null);
    setLoading(true);
    try {
      const params: AdminProductosQuery = {
        q: (next?.q ?? debouncedQ) || undefined,
        activo: next?.activo ?? activoFilter,
        page: next?.page ?? page,
        limit: next?.limit ?? limit,
      };
      const res: Pagina<Producto> = await listProductosAdmin(params);
      setProductos(res.data);
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
  }, [debouncedQ, activoFilter, limit]);

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

  const handleToggleActivo = async (p: Producto) => {
    setBusyId(p.id);
    try {
      await updateProducto(p.id, { activo: !p.activo });
      await load();
    } catch (err) {
      if (err instanceof ApiError) setError(err.message);
      else setError(err instanceof Error ? err.message : 'Error desconocido');
    } finally {
      setBusyId(null);
    }
  };

  const handleDelete = async (p: Producto) => {
    const ok = window.confirm(
      `¿Eliminar "${p.nombre}"? El producto no se borrará pero dejará de aparecer en el catálogo.`,
    );
    if (!ok) return;
    setBusyId(p.id);
    try {
      await softDeleteProducto(p.id);
      await load({ page: 1 });
    } catch (err) {
      if (err instanceof ApiError) setError(err.message);
      else setError(err instanceof Error ? err.message : 'Error desconocido');
    } finally {
      setBusyId(null);
    }
  };

  const handleSaved = async () => {
    setCreating(false);
    setEditing(null);
    await load({ page: 1 });
  };

  const showForm = creating || editing !== null;

  return (
    <div className="space-y-4 md:space-y-6">
      {/* Top bar */}
      <div className="flex flex-col gap-3">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
          <h1 className="text-2xl font-bold text-slate-800">Productos</h1>
          <button
            type="button"
            onClick={() => {
              setEditing(null);
              setCreating(true);
            }}
            className="min-h-[44px] px-4 py-2 rounded-lg bg-marca-600 text-white font-semibold hover:bg-marca-700 transition"
          >
            Crear producto
          </button>
        </div>

        <div className="flex flex-col sm:flex-row gap-3">
          <input
            type="search"
            value={q}
            onChange={(e) => setQ(e.target.value)}
            placeholder="Buscar por nombre..."
            className="flex-1 min-h-[44px] px-4 py-2 rounded-lg border border-slate-300 bg-white focus:ring-2 focus:ring-marca-600 focus:border-marca-600 outline-none transition"
            aria-label="Buscar productos"
          />
          <select
            value={activoFilter}
            onChange={(e) => {
              setActivoFilter(e.target.value as ActivoFilter);
              setPage(1);
            }}
            className="min-h-[44px] px-3 py-2 rounded-lg border border-slate-300 bg-white focus:ring-2 focus:ring-marca-600 focus:border-marca-600 outline-none transition"
            aria-label="Filtrar por estado"
          >
            <option value="all">Todos</option>
            <option value="true">Activos</option>
            <option value="false">Inactivos</option>
          </select>
        </div>
      </div>

      {/* Form overlay + table area */}
      <div className="relative">
        {showForm && (
          <div
            className="fixed inset-0 bg-black/50 z-30 flex items-start sm:items-center justify-center p-4 overflow-y-auto"
            role="dialog"
            aria-modal="true"
            onClick={(e) => {
              if (e.target === e.currentTarget) {
                setCreating(false);
                setEditing(null);
              }
            }}
          >
            <ProductoForm
              initial={editing ?? undefined}
              onCancel={() => {
                setCreating(false);
                setEditing(null);
              }}
              onSaved={handleSaved}
            />
          </div>
        )}

        {/* Error */}
        {error && (
          <div className="bg-red-50 text-red-700 border border-red-200 p-3 rounded-lg flex flex-col sm:flex-row sm:items-center sm:justify-between gap-2 mb-3">
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
            <p className="mt-2 text-sm">Cargando productos...</p>
          </div>
        )}

        {/* Empty */}
        {!loading && !error && productos.length === 0 && (
          <div className="bg-white border border-slate-200 rounded-lg p-8 text-center text-slate-500">
            No hay productos. Crea el primero.
          </div>
        )}

        {/* Table (desktop) / Cards (mobile) */}
        {!loading && productos.length > 0 && (
          <>
            {/* Desktop table */}
            <div className="hidden md:block bg-white border border-slate-200 rounded-lg overflow-hidden">
              <table className="w-full text-sm">
                <thead className="bg-slate-50 text-slate-600 uppercase text-xs">
                  <tr>
                    <th className="text-left px-4 py-3">Img</th>
                    <th className="text-left px-4 py-3">Nombre</th>
                    <th className="text-left px-4 py-3">Categoría</th>
                    <th className="text-right px-4 py-3">Precio</th>
                    <th className="text-right px-4 py-3">Stock</th>
                    <th className="text-left px-4 py-3">Estado</th>
                    <th className="text-right px-4 py-3">Acciones</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100">
                  {productos.map((p) => (
                    <tr key={p.id} className="hover:bg-slate-50">
                      <td className="px-4 py-3">
                        {p.img ? (
                          <img
                            src={p.img}
                            alt={p.nombre}
                            className="w-16 h-16 object-cover rounded-md border border-slate-200"
                          />
                        ) : (
                          <div className="w-16 h-16 rounded-md bg-slate-100 border border-slate-200 flex items-center justify-center text-slate-400 text-xs">
                            N/A
                          </div>
                        )}
                      </td>
                      <td className="px-4 py-3 font-medium text-slate-800">{p.nombre}</td>
                      <td className="px-4 py-3 text-slate-600">{p.categoria}</td>
                      <td className="px-4 py-3 text-right text-slate-800">
                        ${p.precio.toFixed(2)}
                      </td>
                      <td className="px-4 py-3 text-right text-slate-800">{p.stock}</td>
                      <td className="px-4 py-3">
                        <Badge tono={p.activo ? 'exito' : 'neutro'}>
                          {p.activo ? 'Activo' : 'Inactivo'}
                        </Badge>
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex items-center justify-end gap-1">
                          <button
                            type="button"
                            onClick={() => {
                              setCreating(false);
                              setEditing(p);
                            }}
                            className="p-2 rounded hover:bg-slate-100 text-slate-600 hover:text-slate-900 min-h-[44px]"
                            title="Editar"
                            aria-label={`Editar ${p.nombre}`}
                          >
                            <MdEdit className="text-lg" />
                          </button>
                          <button
                            type="button"
                            disabled={busyId === p.id}
                            onClick={() => void handleToggleActivo(p)}
                            className="p-2 rounded hover:bg-slate-100 text-slate-600 hover:text-slate-900 disabled:opacity-50 min-h-[44px]"
                            title={p.activo ? 'Desactivar' : 'Activar'}
                            aria-label={
                              p.activo ? `Desactivar ${p.nombre}` : `Activar ${p.nombre}`
                            }
                          >
                            {p.activo ? (
                              <MdToggleOn className="text-lg" />
                            ) : (
                              <MdToggleOff className="text-lg" />
                            )}
                          </button>
                          <button
                            type="button"
                            disabled={busyId === p.id}
                            onClick={() => void handleDelete(p)}
                            className="p-2 rounded hover:bg-red-50 text-red-600 disabled:opacity-50 min-h-[44px]"
                            title="Eliminar"
                            aria-label={`Eliminar ${p.nombre}`}
                          >
                            <MdDelete className="text-lg" />
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {/* Mobile cards */}
            <div className="md:hidden space-y-3">
              {productos.map((p) => (
                <div
                  key={p.id}
                  className="bg-white border border-slate-200 rounded-lg p-4 flex gap-3"
                >
                  {p.img ? (
                    <img
                      src={p.img}
                      alt={p.nombre}
                      className="w-16 h-16 object-cover rounded-md border border-slate-200 flex-shrink-0"
                    />
                  ) : (
                    <div className="w-16 h-16 rounded-md bg-slate-100 border border-slate-200 flex items-center justify-center text-slate-400 text-xs flex-shrink-0">
                      N/A
                    </div>
                  )}
                  <div className="flex-1 min-w-0">
                    <div className="flex items-start justify-between gap-2">
                      <div className="min-w-0">
                        <p className="font-medium text-slate-800 truncate">{p.nombre}</p>
                        <p className="text-xs text-slate-500">{p.categoria}</p>
                      </div>
                      <Badge tono={p.activo ? 'exito' : 'neutro'} className="flex-shrink-0">
                        {p.activo ? 'Activo' : 'Inactivo'}
                      </Badge>
                    </div>
                    <div className="mt-2 flex items-center justify-between text-sm text-slate-700">
                      <span>${p.precio.toFixed(2)}</span>
                      <span>Stock: {p.stock}</span>
                    </div>
                    <div className="mt-3 flex items-center justify-end gap-1">
                      <button
                        type="button"
                        onClick={() => {
                          setCreating(false);
                          setEditing(p);
                        }}
                        className="p-2 rounded hover:bg-slate-100 text-slate-600 min-h-[44px]"
                        title="Editar"
                        aria-label={`Editar ${p.nombre}`}
                      >
                        <MdEdit className="text-lg" />
                      </button>
                      <button
                        type="button"
                        disabled={busyId === p.id}
                        onClick={() => void handleToggleActivo(p)}
                        className="p-2 rounded hover:bg-slate-100 text-slate-600 disabled:opacity-50 min-h-[44px]"
                        title={p.activo ? 'Desactivar' : 'Activar'}
                        aria-label={p.activo ? `Desactivar ${p.nombre}` : `Activar ${p.nombre}`}
                      >
                        {p.activo ? (
                          <MdToggleOn className="text-lg" />
                        ) : (
                          <MdToggleOff className="text-lg" />
                        )}
                      </button>
                      <button
                        type="button"
                        disabled={busyId === p.id}
                        onClick={() => void handleDelete(p)}
                        className="p-2 rounded hover:bg-red-50 text-red-600 disabled:opacity-50 min-h-[44px]"
                        title="Eliminar"
                        aria-label={`Eliminar ${p.nombre}`}
                      >
                        <MdDelete className="text-lg" />
                      </button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </>
        )}
      </div>

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
          <div className="flex items-center justify-center gap-2 text-sm text-slate-600">
            <span>Mostrar</span>
            <select
              value={limit}
              onChange={(e) => {
                setLimit(Number(e.target.value));
                setPage(1);
              }}
              className="min-h-[36px] px-2 py-1 rounded border border-slate-300 bg-white focus:ring-2 focus:ring-marca-600 focus:border-marca-600 outline-none"
              aria-label="Límite por página"
            >
              <option value={10}>10</option>
              <option value={20}>20</option>
              <option value={50}>50</option>
            </select>
            <span>({total} total)</span>
          </div>
        </div>
      )}
    </div>
  );
};

export default AdminProductos;
