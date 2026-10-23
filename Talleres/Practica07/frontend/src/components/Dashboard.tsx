import { useCallback } from 'react';
import { MdRefresh } from 'react-icons/md';
import ComisionesPendientes from './reportes/ComisionesPendientes';
import ResumenCards from './reportes/ResumenCards';
import TopProductos from './reportes/TopProductos';
import TopReferidos from './reportes/TopReferidos';
import { useReportes } from '../hooks/useReportes';
import {
  getComisionesPendientes,
  getResumen,
  getTopProductos,
  getTopReferidos,
} from '../services/reportesService';

/** Límite del ranking. El backend acepta 1..50. */
const LIMIT_RANKING = 10;

const Dashboard = () => {
  // Los cuatro widgets son independientes: cada uno pide su endpoint por
  // separado. El resumen no bloquea a los rankings ni al revés, así que un
  // 500 en uno no deja la pantalla en blanco.
  const resumen = useReportes(() => getResumen(), []);
  const topProductos = useReportes(
    () => getTopProductos({ limit: LIMIT_RANKING }),
    [],
  );
  const topReferidos = useReportes(
    () => getTopReferidos({ limit: LIMIT_RANKING }),
    [],
  );
  const pendientes = useReportes(
    () => getComisionesPendientes({ limit: 25 }),
    [],
  );

  const recargarTodo = useCallback(() => {
    resumen.reload();
    topProductos.reload();
    topReferidos.reload();
    pendientes.reload();
  }, [resumen, topProductos, topReferidos, pendientes]);

  const refreshing =
    resumen.cargando ||
    topProductos.cargando ||
    topReferidos.cargando ||
    pendientes.cargando;

  return (
    <div className="space-y-6">
      <header className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-slate-800">Dashboard</h1>
          <p className="text-sm text-slate-500">
            Ventas, comisiones y liquidaciones de la organización.
          </p>
        </div>
        <button
          type="button"
          onClick={recargarTodo}
          disabled={refreshing}
          className="flex items-center gap-2 rounded-md bg-slate-800 px-4 py-2 text-sm font-medium text-white transition hover:bg-slate-700 disabled:cursor-not-allowed disabled:opacity-50"
        >
          <MdRefresh
            className={refreshing ? 'animate-spin' : undefined}
            aria-hidden
          />
          {refreshing ? 'Actualizando…' : 'Actualizar'}
        </button>
      </header>

      <ResumenCards
        data={resumen.data}
        cargando={resumen.cargando}
        error={resumen.error}
      />

      <div className="grid grid-cols-1 gap-6 xl:grid-cols-2">
        <TopProductos
          data={topProductos.data}
          cargando={topProductos.cargando}
          error={topProductos.error}
        />
        <TopReferidos
          data={topReferidos.data}
          cargando={topReferidos.cargando}
          error={topReferidos.error}
        />
      </div>

      <ComisionesPendientes
        data={pendientes.data}
        cargando={pendientes.cargando}
        error={pendientes.error}
      />
    </div>
  );
};

export default Dashboard;