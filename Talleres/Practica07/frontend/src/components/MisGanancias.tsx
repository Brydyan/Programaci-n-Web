import { useCallback, useEffect, useState } from 'react';
import { getComisionesMiasAPI, getLiquidacionesMiasAPI } from '../services/api';
import {
  getErrorMessage,
  type Comision,
  type ComisionResumen,
  type EstadoComision,
  type Liquidacion,
  type Paginado,
} from '../types/api';
import { useAuth } from '../context/auth';
import { formatCurrency, formatFecha } from './reportes/format';

/**
 * "Mis Ganancias" — el lado del dinero del MLM.
 *
 * Dos recursos que el backend ya servía y que hasta ahora nadie miraba:
 *
 *   GET /api/comisiones/mias      el ledger: cuánto se devengó, por qué
 *                                 pedido y en qué nivel
 *   GET /api/liquidaciones/mias   los cobros: cuándo y cuánto me pagaron
 *
 * La diferencia importa y es la razón de que sean dos secciones y no una:
 * una comisión devengada no es dinero recibido. El usuario puede ver que
 * tiene $80 aprobada y $0 cobrados, que son hechos distintos con
 * consecuencias distintas.
 *
 * Por eso los totales se separan en "lo generado" y "lo cobrado", y no se
 * presenta un único número de "mis ganancias": ese número no existe.
 */

type FiltroEstado = EstadoComision | 'todas';

const FILTROS: { valor: FiltroEstado; texto: string }[] = [
  { valor: 'todas', texto: 'Todas' },
  { valor: 'pendiente', texto: 'Pendientes' },
  { valor: 'aprobada', texto: 'Aprobadas' },
  { valor: 'pagada', texto: 'Pagadas' },
  { valor: 'revertida', texto: 'Revertidas' },
];

const ESTADO_COMISION: Record<EstadoComision, { texto: string; clase: string }> = {
  pendiente: { texto: 'Pendiente', clase: 'bg-amber-100 text-amber-800' },
  aprobada: { texto: 'Aprobada', clase: 'bg-sky-100 text-sky-800' },
  pagada: { texto: 'Pagada', clase: 'bg-emerald-100 text-emerald-800' },
  revertida: { texto: 'Revertida', clase: 'bg-slate-200 text-slate-600' },
};

const ESTADO_LIQUIDACION = {
  pendiente: { texto: 'Pendiente', clase: 'bg-amber-100 text-amber-800' },
  pagada: { texto: 'Pagada', clase: 'bg-emerald-100 text-emerald-800' },
} as const;

const MisGanancias = () => {
  const { userEmail } = useAuth();
  const [comisiones, setComisiones] = useState<ComisionResumen | null>(null);
  const [liquidaciones, setLiquidaciones] = useState<Paginado<Liquidacion> | null>(null);
  const [filtro, setFiltro] = useState<FiltroEstado>('todas');
  const [pagina, setPagina] = useState(1);
  const [isLoading, setIsLoading] = useState(true);
  const [isRefrescando, setIsRefrescando] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const cargar = useCallback(async () => {
    setIsRefrescando(true);
    setError(null);
    // Los dos recursos son independientes: si las liquidaciones fallan,
    // el ledger sigue siendo información útil, y al revés.
    const [c, l] = await Promise.allSettled([
      getComisionesMiasAPI(),
      getLiquidacionesMiasAPI({ page: pagina, limit: 20 }),
    ]);

    if (c.status === 'fulfilled') setComisiones(c.value);
    if (l.status === 'fulfilled') setLiquidaciones(l.value);

    const fallo = [c, l].find((r) => r.status === 'rejected');
    setError(fallo ? getErrorMessage((fallo as PromiseRejectedResult).reason) : null);
    setIsRefrescando(false);
  }, [pagina]);

  // La carga inicial corre dentro del efecto y tras el primer await, con el
  // guard `vivo`: nada de setState síncrono en el cuerpo del efecto (React lo
  // marca como cascada de renders) y nada de pintar si el componente ya se
  // desmontó. El botón "Actualizar" sí llama a `cargar` directo, porque ahí el
  // setState viene de un evento de usuario.
  useEffect(() => {
    let vivo = true;
    void (async () => {
      const [c, l] = await Promise.allSettled([
        getComisionesMiasAPI(),
        getLiquidacionesMiasAPI({ page: pagina, limit: 20 }),
      ]);
      if (!vivo) return;
      if (c.status === 'fulfilled') setComisiones(c.value);
      if (l.status === 'fulfilled') setLiquidaciones(l.value);
      const fallo = [c, l].find((r) => r.status === 'rejected');
      setError(fallo ? getErrorMessage((fallo as PromiseRejectedResult).reason) : null);
      setIsLoading(false);
    })();
    return () => {
      vivo = false;
    };
  }, [pagina]);

  if (isLoading) {
    return (
      <div className="flex justify-center items-center min-h-96">
        <p className="text-slate-600 font-medium">Cargando tus ganancias...</p>
      </div>
    );
  }

  // El filtro es de cliente: /comisiones/mias devuelve el conjunto completo
  // con sus totales, así que filtrar aquí evita un request por estado y
  // deja los totales globales intactos arriba.
  const filas: Comision[] =
    filtro === 'todas' || !comisiones
      ? (comisiones?.data ?? [])
      : comisiones.data.filter((c) => c.estado === filtro);

  // Una comisión revertida NO es ganancia: el pedido se canceló y el dinero
  // se devolvió. El backend ya aplica esa misma regla en sus reportes
  // ("una comisión revertida nunca se devengó", repository.go), y sumarla
  // aquí daría un número que nadie puede cobrar.
  const totalGenerado =
    comisiones?.totales
      .filter((t) => t.estado !== 'revertida')
      .reduce((acc, t) => acc + t.total, 0) ?? 0;

  const totalCobrado = (liquidaciones?.data ?? [])
    .filter((l) => l.estado === 'pagada')
    .reduce((acc, l) => acc + l.total, 0);

  return (
    <div>
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between mb-6 gap-4">
        <div>
          <h1 className="text-2xl font-bold text-slate-800">Mis Ganancias</h1>
          <p className="text-sm text-slate-500">{userEmail}</p>
        </div>
        <button
          onClick={cargar}
          disabled={isRefrescando}
          className="text-xs bg-indigo-50 text-indigo-600 hover:bg-indigo-100 disabled:opacity-50 px-3 py-2 rounded-md font-medium transition self-start min-h-[44px]"
        >
          {isRefrescando ? 'Actualizando...' : 'Actualizar'}
        </button>
      </div>

      {error && (
        <div className="bg-red-50 text-red-600 p-4 rounded-lg border border-red-200 mb-6">
          <p className="text-sm font-medium">{error}</p>
          <button
            onClick={cargar}
            className="mt-3 text-sm bg-white border border-red-300 px-4 py-2 rounded-md font-medium min-h-[44px]"
          >
            Reintentar
          </button>
        </div>
      )}

      {/* Dos cifras separadas a propósito: devengado no es cobrado. */}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 mb-6">
        <TarjetaValor
          etiqueta="Generado"
          valor={formatCurrency(totalGenerado)}
          ayuda="Comisiones devengadas, sin las revertidas"
          testId="tarjeta-generado"
        />
        <TarjetaValor
          etiqueta="Cobrado"
          valor={formatCurrency(totalCobrado)}
          ayuda="Liquidaciones ya pagadas"
          testId="tarjeta-cobrado"
        />
      </div>

      {comisiones && comisiones.totales.length > 0 && (
        <div className="bg-white rounded-lg border border-slate-200 p-4 mb-6 shadow-sm">
          <p className="text-xs font-semibold uppercase text-slate-500 mb-3">
            Por estado
          </p>
          <div className="flex flex-wrap gap-2">
            {comisiones.totales.map((t) => {
              const est = ESTADO_COMISION[t.estado];
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

      <TablaComisiones filas={filas} filtro={filtro} setFiltro={setFiltro} />
      <PanelLiquidaciones datos={liquidaciones} pagina={pagina} setPagina={setPagina} />
    </div>
  );
};

const TarjetaValor = ({
  etiqueta,
  valor,
  ayuda,
  testId,
}: {
  etiqueta: string;
  valor: string;
  ayuda: string;
  testId: string;
}) => (
  <div
    data-testid={testId}
    className="bg-white p-4 rounded-lg border border-slate-200"
  >
    <p className="text-xs uppercase font-semibold text-slate-500">{etiqueta}</p>
    <p className="text-2xl font-bold text-indigo-600">{valor}</p>
    <p className="text-xs text-slate-400 mt-1">{ayuda}</p>
  </div>
);

const TablaComisiones = ({
  filas,
  filtro,
  setFiltro,
}: {
  filas: Comision[];
  filtro: FiltroEstado;
  setFiltro: (f: FiltroEstado) => void;
}) => (
  <div className="bg-white rounded-lg border border-slate-200 p-4 mb-6 shadow-sm">
    <p className="text-xs font-semibold uppercase text-slate-500 mb-3">
      Comisiones generadas
    </p>

    <div className="flex flex-wrap gap-2 mb-4">
      {FILTROS.map((f) => (
        <button
          key={f.valor}
          onClick={() => setFiltro(f.valor)}
          aria-pressed={filtro === f.valor}
          className={`text-xs px-3 py-1.5 rounded-md font-medium border transition min-h-[36px] ${
            filtro === f.valor
              ? 'bg-indigo-600 text-white border-indigo-600'
              : 'bg-white text-slate-600 border-slate-300 hover:bg-slate-50'
          }`}
        >
          {f.texto}
        </button>
      ))}
    </div>

    {filas.length === 0 ? (
      <p className="text-sm text-slate-500">
        No hay comisiones con este filtro. Aparecen cuando alguien de tu red compra.
      </p>
    ) : (
      <div className="overflow-x-auto">
        <table className="w-full text-left text-sm">
          <thead className="border-b border-slate-200">
            <tr>
              <th className="py-2 pr-3 font-semibold text-slate-600">Pedido</th>
              <th className="py-2 pr-3 font-semibold text-slate-600">Fecha</th>
              <th className="py-2 pr-3 font-semibold text-slate-600">Nivel</th>
              <th className="py-2 pr-3 font-semibold text-slate-600">%</th>
              <th className="py-2 pr-3 font-semibold text-slate-600">Monto</th>
              <th className="py-2 font-semibold text-slate-600">Estado</th>
            </tr>
          </thead>
          <tbody>
            {filas.map((c) => {
              const est = ESTADO_COMISION[c.estado];
              return (
                <tr key={c.id} className="border-b border-slate-100">
                  <td className="py-2 pr-3 text-slate-700">#{c.pedido_id ?? '—'}</td>
                  <td className="py-2 pr-3 text-slate-500">{formatFecha(c.creado_en)}</td>
                  <td className="py-2 pr-3 text-slate-500">{c.nivel}</td>
                  <td className="py-2 pr-3 text-slate-500">{c.porcentaje}%</td>
                  <td className="py-2 pr-3 font-semibold text-slate-800">
                    {formatCurrency(c.monto)}
                  </td>
                  <td className="py-2">
                    <span
                      className={`rounded px-2 py-0.5 text-xs font-medium ${est.clase}`}
                    >
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
  </div>
);

const PanelLiquidaciones = ({
  datos,
  pagina,
  setPagina,
}: {
  datos: Paginado<Liquidacion> | null;
  pagina: number;
  setPagina: (p: number) => void;
}) => {
  if (!datos) return null;
  const ultima = datos.total <= datos.limit;
  const primera = (pagina - 1) * datos.limit + 1;
  const ultimaFila = Math.min(pagina * datos.limit, datos.total);

  return (
    <div className="bg-white rounded-lg border border-slate-200 p-4 shadow-sm">
      <p className="text-xs font-semibold uppercase text-slate-500 mb-3">Mis cobros</p>

      {datos.total === 0 ? (
        <p className="text-sm text-slate-500">
          Todavía no te han pagado comisiones. Cuando el admin liquide tu deuda aparecerá
          aquí.
        </p>
      ) : (
        <>
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead className="border-b border-slate-200">
                <tr>
                  <th className="py-2 pr-3 font-semibold text-slate-600">Periodo</th>
                  <th className="py-2 pr-3 font-semibold text-slate-600">Emitida</th>
                  <th className="py-2 pr-3 font-semibold text-slate-600">Fecha de pago</th>
                  <th className="py-2 pr-3 font-semibold text-slate-600">Monto</th>
                  <th className="py-2 font-semibold text-slate-600">Estado</th>
                </tr>
              </thead>
              <tbody>
                {datos.data.map((l) => {
                  const est = ESTADO_LIQUIDACION[l.estado];
                  return (
                    <tr key={l.id} className="border-b border-slate-100">
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
                          className={`rounded px-2 py-0.5 text-xs font-medium ${est.clase}`}
                        >
                          {est.texto}
                        </span>
                      </td>
                    </tr>
                  );
                })}
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

export default MisGanancias;