import { Link } from 'react-router-dom';
import { useCallback, useEffect, useState } from 'react';
import { getComisionesMiasAPI } from '../services/api';
import { getRedMiaAPI } from '../services/redService';
import {
  getErrorMessage,
  type ComisionResumen,
  type EstadoComision,
  type NodoRed,
} from '../types/api';
import { useAuth } from '../context/auth';
import { RamaRed, ResumenCards } from './RedArbol';
import { resumenRed } from './redUtils';

/**
 * "Mi Red" — Sprint 1.
 *
 * Muestra el subárbol del usuario autenticado (GET /api/red/mia), no el árbol
 * global: lo accionable para un distribuidor es quién depende de él.
 *
 * El nodo raíz (nivel 0) es el propio usuario y su `codigo` es el que se
 * comparte en el link de invitación /registro?ref=CODIGO.
 */

const money = (n: number) => `$${n.toFixed(2)}`;

const MiRed = () => {
  const { userEmail } = useAuth();
  const [arbol, setArbol] = useState<NodoRed[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [copiado, setCopiado] = useState(false);
  const [comisiones, setComisiones] = useState<ComisionResumen | null>(null);

  const cargar = useCallback(async () => {
    try {
      setIsLoading(true);
      setError(null);
      setArbol(await getRedMiaAPI());
    } catch (err) {
      setError(getErrorMessage(err));
      setArbol([]);
    } finally {
      setIsLoading(false);
    }
  }, []);

  // La carga inicial corre en un microtask y está guardada por `vivo`: así el
  // setState ocurre tras el primer await (nada de setState síncrono en el
  // efecto) y no se pinta nada si el componente ya se desmontó.
  useEffect(() => {
    let vivo = true;
    void (async () => {
      try {
        const data = await getRedMiaAPI();
        if (vivo) setArbol(data);
      } catch (err) {
        if (!vivo) return;
        setError(getErrorMessage(err));
        setArbol([]);
      }
      // El ledger es un recurso aparte: si falla, la red sigue siendo útil.
      try {
        const ledger = await getComisionesMiasAPI();
        if (vivo) setComisiones(ledger);
      } catch {
        if (vivo) setComisiones(null);
      } finally {
        if (vivo) setIsLoading(false);
      }
    })();
    return () => {
      vivo = false;
    };
  }, []);

  // El nodo propio es el nivel 0 de la raíz que devuelve el endpoint.
  const miNodo = arbol[0];
  const linkReferido = miNodo
    ? `${window.location.origin}/registro?ref=${miNodo.codigo}`
    : '';

  const copiarLink = async () => {
    try {
      await navigator.clipboard.writeText(linkReferido);
      setCopiado(true);
      setTimeout(() => setCopiado(false), 2000);
    } catch {
      // Clipboard bloqueado (contexto no seguro): el link sigue visible
      // para copiarlo a mano, así que no es un fallo que warranta alerta.
    }
  };

  if (isLoading) {
    return (
      <div className="flex justify-center items-center min-h-96">
        <p className="text-slate-600 font-medium">Cargando tu red...</p>
      </div>
    );
  }

  const totales = resumenRed(arbol);

  return (
    <div>
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between mb-6 gap-4">
        <div>
          <h1 className="text-2xl font-bold text-slate-800">Mi Red</h1>
          <p className="text-sm text-slate-500">{userEmail}</p>
        </div>
        {!error && arbol.length > 0 && (
          <button
            onClick={cargar}
            className="text-xs bg-indigo-50 text-indigo-600 hover:bg-indigo-100 px-3 py-2 rounded-md font-medium transition self-start min-h-[44px]"
          >
            Actualizar
          </button>
        )}
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

      {!error && arbol.length === 0 && (
        <div className="bg-white rounded-lg border border-slate-200 p-8 text-center text-slate-500">
          Todavía no tienes referidos. Comparte tu link para que tu red crezca.
        </div>
      )}

      {!error && miNodo && (
        <>
          {/* Link de invitación: es la puerta de crecimiento del MLM. */}
          <div className="bg-white rounded-lg border border-slate-200 p-4 mb-6 shadow-sm">
            <p className="text-xs font-semibold uppercase text-slate-500 mb-2">
              Tu link de referido
            </p>
            <div className="flex flex-col sm:flex-row gap-3 sm:items-center">
              <code className="flex-1 bg-slate-50 border border-slate-200 rounded-md px-3 py-2 text-sm text-slate-700 break-all">
                {linkReferido}
              </code>
              <button
                onClick={copiarLink}
                className="min-h-[44px] px-4 py-2 bg-indigo-600 text-white rounded-md font-semibold hover:bg-indigo-700 transition text-sm"
              >
                {copiado ? '¡Copiado!' : 'Copiar'}
              </button>
            </div>
          </div>

          {/* Resumen de la rama. Compartido con "Red Global" para que las
              dos vistas no puedan mostrar cifras distintas. */}
          <ResumenCards resumen={totales} />

          <PanelComisiones resumen={comisiones} />

          <RamaRed nodos={miNodo.hijos ?? []} nivel={1} />
        </>
      )}
    </div>
  );
};

/** Etiqueta y color por estado del ledger. */
const ESTADO_COMISION: Record<EstadoComision, { texto: string; clase: string }> = {
  pendiente: { texto: 'Pendiente', clase: 'bg-amber-100 text-amber-800' },
  aprobada: { texto: 'Aprobada', clase: 'bg-sky-100 text-sky-800' },
  pagada: { texto: 'Pagada', clase: 'bg-emerald-100 text-emerald-800' },
  revertida: { texto: 'Revertida', clase: 'bg-slate-200 text-slate-600' },
};

/**
 * Resumen del ledger dentro de "Mi Red".
 *
 * Antes esta tabla listaba cada comisión aquí. Ahora vive entera en
 * "Mis Ganancias" (/mis-ganancias), que además trae los cobros y separa
 * devengado de cobrado. Aquí queda solo el agregado por estado: dentro de
 * la vista de red interesa "¿cuánto ha producido mi red?", no el
 * detalle línea por línea.
 */
const PanelComisiones = ({ resumen }: { resumen: ComisionResumen | null }) => {
  if (!resumen) {
    return (
      <div className="bg-white rounded-lg border border-slate-200 p-4 mb-6 text-sm text-slate-500">
        No se pudo cargar el detalle de tus comisiones.
      </div>
    );
  }
  if (resumen.total === 0) {
    return (
      <div className="bg-white rounded-lg border border-dashed border-slate-300 p-4 mb-6 text-sm text-slate-500">
        Aún no tienes comisiones registradas. Aparecen cuando alguien de tu red compra.
      </div>
    );
  }

  return (
    <div className="bg-white rounded-lg border border-slate-200 p-4 mb-6 shadow-sm">
      <div className="flex items-center justify-between gap-3 mb-3">
        <p className="text-xs font-semibold uppercase text-slate-500">Mis comisiones</p>
        <Link
          to="/mis-ganancias"
          className="text-xs text-indigo-600 hover:text-indigo-700 font-medium min-h-[36px] inline-flex items-center"
        >
          Ver detalle →
        </Link>
      </div>
      <div className="flex flex-wrap gap-2">
        {resumen.totales.map((t) => {
          const est = ESTADO_COMISION[t.estado];
          return (
            <div key={t.estado} className={`rounded-md px-3 py-2 ${est.clase}`}>
              <p className="text-xs font-medium">{est.texto}</p>
              <p className="text-sm font-bold">{money(t.total)}</p>
            </div>
          );
        })}
      </div>
    </div>
  );
};

export default MiRed;