import { useCallback, useEffect, useState } from 'react';
import { getDeuda, liquidar, listLiquidaciones } from '../services/liquidacionesService';
import { getErrorMessage, type DeudaDistribuidor, type Liquidacion, type Pagina } from '../types/api';
import { formatCurrency, formatFecha } from './reportes/format';

/**
 * "Liquidaciones" — el lado del cobro.
 *
 * El backend tenía los tres endpoints desde el Sprint 3 (consultar deuda,
 * listar pagos, pagar) y no había forma de llegar a ellos desde la interfaz:
 * liquidar exigía curl con un token de admin. Esta pantalla cierra eso.
 *
 * Pagar mueve dinero, así que cada fila pide confirmación y muestra el
 * monto exacto antes de confirmar. El backend es la segunda barrera: un
 * segundo intento sobre un referido ya liquidado devuelve 409 y no cobra
 * de nuevo.
 */

const Liquidaciones = () => {
  const [deudas, setDeudas] = useState<DeudaDistribuidor[]>([]);
  const [historial, setHistorial] = useState<Pagina<Liquidacion> | null>(null);
  const [pagina, setPagina] = useState(1);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [porPagar, setPorPagar] = useState<DeudaDistribuidor | null>(null);
  const [pagando, setPagando] = useState(false);
  const [aviso, setAviso] = useState<string | null>(null);
  const [errorPago, setErrorPago] = useState<string | null>(null);

  const cargar = useCallback(async () => {
    setError(null);
    const [d, h] = await Promise.allSettled([getDeuda(), listLiquidaciones({ page: pagina, limit: 20 })]);
    if (d.status === 'fulfilled') setDeudas(d.value);
    if (h.status === 'fulfilled') setHistorial(h.value);
    const fallo = [d, h].find((r) => r.status === 'rejected');
    setError(fallo ? getErrorMessage((fallo as PromiseRejectedResult).reason) : null);
    setIsLoading(false);
  }, [pagina]);

  useEffect(() => {
    let vivo = true;
    void (async () => {
      const [d, h] = await Promise.allSettled([getDeuda(), listLiquidaciones({ page: pagina, limit: 20 })]);
      if (!vivo) return;
      if (d.status === 'fulfilled') setDeudas(d.value);
      if (h.status === 'fulfilled') setHistorial(h.value);
      const fallo = [d, h].find((r) => r.status === 'rejected');
      setError(fallo ? getErrorMessage((fallo as PromiseRejectedResult).reason) : null);
      setIsLoading(false);
    })();
    return () => {
      vivo = false;
    };
  }, [pagina]);

  const confirmarPago = async () => {
    if (!porPagar) return;
    setPagando(true);
    setErrorPago(null);
    try {
      const liq = await liquidar(porPagar.referido_id);
      setAviso(
        `Liquidación #${liq.id} por ${formatCurrency(liq.total)} a ${porPagar.nombre}. ` +
          `Periodo ${liq.periodo ?? '—'}, pagada el ${formatFecha(liq.pagada_en ?? liq.creado_en)}.`,
      );
      setPorPagar(null);
      await cargar();
    } catch (err) {
      setErrorPago(getErrorMessage(err));
    } finally {
      setPagando(false);
    }
  };

  if (isLoading) {
    return (
      <div className="flex justify-center items-center min-h-96">
        <p className="text-slate-600 font-medium">Cargando liquidaciones...</p>
      </div>
    );
  }

  const totalAPagar = deudas.reduce((acc, d) => acc + d.total, 0);

  return (
    <div>
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between mb-6 gap-4">
        <div>
          <h1 className="text-2xl font-bold text-slate-800">Liquidaciones</h1>
          <p className="text-sm text-slate-500">Cobro de comisiones a los distribuidores</p>
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

      {aviso && (
        <div role="status" className="bg-emerald-50 text-emerald-800 p-4 rounded-lg border border-emerald-200 mb-6">
          <p className="text-sm font-medium">{aviso}</p>
          <button onClick={() => setAviso(null)} className="mt-2 text-xs underline min-h-[32px]">
            Cerrar
          </button>
        </div>
      )}

      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 mb-6">
        <div className="bg-white p-4 rounded-lg border border-slate-200">
          <p className="text-xs uppercase font-semibold text-slate-500">Por pagar</p>
          <p className="text-2xl font-bold text-indigo-600">{formatCurrency(totalAPagar)}</p>
          <p className="text-xs text-slate-400 mt-1">
            {deudas.length} {deudas.length === 1 ? 'distribuidor' : 'distribuidores'}
          </p>
        </div>
        <div className="bg-white p-4 rounded-lg border border-slate-200">
          <p className="text-xs uppercase font-semibold text-slate-500">Pagos registrados</p>
          <p className="text-2xl font-bold text-slate-800">{historial?.total ?? 0}</p>
          <p className="text-xs text-slate-400 mt-1">Histórico completo</p>
        </div>
      </div>

      <PanelDeuda deudas={deudas} onPagar={setPorPagar} />
      <PanelHistorial datos={historial} pagina={pagina} setPagina={setPagina} />

      {porPagar && (
        <DialogoConfirmacion
          deuda={porPagar}
          pagando={pagando}
          error={errorPago}
          onCancelar={() => {
            setPorPagar(null);
            setErrorPago(null);
          }}
          onConfirmar={confirmarPago}
        />
      )}
    </div>
  );
};

const PanelDeuda = ({
  deudas,
  onPagar,
}: {
  deudas: DeudaDistribuidor[];
  onPagar: (d: DeudaDistribuidor) => void;
}) => (
  <div className="bg-white rounded-lg border border-slate-200 p-4 mb-6 shadow-sm">
    <p className="text-xs font-semibold uppercase text-slate-500 mb-3">Deuda pendiente</p>

    {deudas.length === 0 ? (
      <p className="text-sm text-slate-500">
        No hay comisiones pendientes de cobrar. Todo está liquidado.
      </p>
    ) : (
      <div className="overflow-x-auto">
        <table className="w-full text-left text-sm">
          <thead className="border-b border-slate-200">
            <tr>
              <th className="py-2 pr-3 font-semibold text-slate-600">Distribuidor</th>
              <th className="py-2 pr-3 font-semibold text-slate-600">Código</th>
              <th className="py-2 pr-3 font-semibold text-slate-600">Comisiones</th>
              <th className="py-2 pr-3 font-semibold text-slate-600">Último pago</th>
              <th className="py-2 pr-3 font-semibold text-slate-600">Total</th>
              <th className="py-2 font-semibold text-slate-600">Acción</th>
            </tr>
          </thead>
          <tbody>
            {deudas.map((d) => (
              <tr key={d.referido_id} className="border-b border-slate-100">
                <td className="py-2 pr-3 text-slate-800 font-medium">{d.nombre}</td>
                <td className="py-2 pr-3 font-mono text-slate-500">{d.codigo}</td>
                <td className="py-2 pr-3 text-slate-500">{d.comisiones}</td>
                <td className="py-2 pr-3 text-slate-500">
                  {d.ultima_pago ? formatFecha(d.ultima_pago) : 'Nunca'}
                </td>
                <td className="py-2 pr-3 font-semibold text-slate-800">
                  {formatCurrency(d.total)}
                </td>
                <td className="py-2">
                  <button
                    onClick={() => onPagar(d)}
                    aria-label={`Liquidar a ${d.nombre}`}
                    className="px-3 py-1.5 rounded-md bg-indigo-600 text-white text-xs font-semibold hover:bg-indigo-700 transition min-h-[36px]"
                  >
                    Liquidar
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    )}
  </div>
);

const DialogoConfirmacion = ({
  deuda,
  pagando,
  error,
  onCancelar,
  onConfirmar,
}: {
  deuda: DeudaDistribuidor;
  pagando: boolean;
  error: string | null;
  onCancelar: () => void;
  onConfirmar: () => void;
}) => {
  useEffect(() => {
    const alPulsar = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !pagando) onCancelar();
    };
    document.addEventListener('keydown', alPulsar);
    return () => document.removeEventListener('keydown', alPulsar);
  }, [pagando, onCancelar]);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/50 p-4">
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="titulo-liquidar"
        className="bg-white rounded-lg shadow-xl max-w-md w-full p-6"
      >
        <h2 id="titulo-liquidar" className="text-lg font-bold text-slate-800 mb-3">
          Confirmar liquidación
        </h2>

        <p className="text-sm text-slate-600 mb-4">
          Vas a pagar{' '}
          <strong className="text-slate-800">
            {formatCurrency(deuda.total)}
          </strong>{' '}
          a <strong className="text-slate-800">{deuda.nombre}</strong>{' '}
          ({deuda.codigo}), correspondientes a {deuda.comisiones} comisiones
          pendientes. Esta acción no se puede deshacer.
        </p>

        {error && (
          <div className="bg-red-50 text-red-700 text-sm p-3 rounded-md border border-red-200 mb-4">
            {error}
          </div>
        )}

        <div className="flex justify-end gap-2">
          <button
            onClick={onCancelar}
            disabled={pagando}
            className="px-4 py-2 rounded-md border border-slate-300 text-slate-600 hover:bg-slate-50 disabled:opacity-50 transition min-h-[44px]"
          >
            Cancelar
          </button>
          <button
            onClick={onConfirmar}
            disabled={pagando}
            className="px-4 py-2 rounded-md bg-indigo-600 text-white font-semibold hover:bg-indigo-700 disabled:opacity-50 transition min-h-[44px]"
          >
            {pagando ? 'Pagando...' : `Pagar ${formatCurrency(deuda.total)}`}
          </button>
        </div>
      </div>
    </div>
  );
};

const PanelHistorial = ({
  datos,
  pagina,
  setPagina,
}: {
  datos: Pagina<Liquidacion> | null;
  pagina: number;
  setPagina: (p: number) => void;
}) => {
  if (!datos) return null;
  const ultima = datos.total <= datos.limit;
  const primera = (pagina - 1) * datos.limit + 1;
  const ultimaFila = Math.min(pagina * datos.limit, datos.total);

  return (
    <div className="bg-white rounded-lg border border-slate-200 p-4 shadow-sm">
      <p className="text-xs font-semibold uppercase text-slate-500 mb-3">Histórico de pagos</p>

      {datos.total === 0 ? (
        <p className="text-sm text-slate-500">Todavía no se ha registrado ninguna liquidación.</p>
      ) : (
        <>
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead className="border-b border-slate-200">
                <tr>
                  <th className="py-2 pr-3 font-semibold text-slate-600">#</th>
                  <th className="py-2 pr-3 font-semibold text-slate-600">Referido</th>
                  <th className="py-2 pr-3 font-semibold text-slate-600">Periodo</th>
                  <th className="py-2 pr-3 font-semibold text-slate-600">Emitida</th>
                  <th className="py-2 pr-3 font-semibold text-slate-600">Pagada</th>
                  <th className="py-2 pr-3 font-semibold text-slate-600">Monto</th>
                  <th className="py-2 font-semibold text-slate-600">Estado</th>
                </tr>
              </thead>
              <tbody>
                {datos.data.map((l) => (
                  <tr key={l.id} className="border-b border-slate-100">
                    <td className="py-2 pr-3 text-slate-500">{l.id}</td>
                    <td className="py-2 pr-3 text-slate-700">{l.referido_id}</td>
                    <td className="py-2 pr-3 text-slate-700">{l.periodo ?? '—'}</td>
                    <td className="py-2 pr-3 text-slate-500">{formatFecha(l.creado_en)}</td>
                    <td className="py-2 pr-3 text-slate-500">
                      {l.pagada_en ? formatFecha(l.pagada_en) : '—'}
                    </td>
                    <td className="py-2 pr-3 font-semibold text-slate-800">
                      {formatCurrency(l.total)}
                    </td>
                    <td className="py-2">
                      <span
                        className={`rounded px-2 py-0.5 text-xs font-medium ${
                          l.estado === 'pagada'
                            ? 'bg-emerald-100 text-emerald-800'
                            : 'bg-amber-100 text-amber-800'
                        }`}
                      >
                        {l.estado === 'pagada' ? 'Pagada' : 'Pendiente'}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

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
        </>
      )}
    </div>
  );
};

export default Liquidaciones;