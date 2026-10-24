import { useCallback, useEffect, useState } from 'react';
import { getRedGlobalAPI } from '../services/redService';
import { getErrorMessage, type NodoRed } from '../types/api';
import { RamaRed, ResumenCards } from './RedArbol';
import { resumenRed, useFiltroArbol } from './redUtils';

/**
 * "Red Global" — el árbol completo de la organización. Admin only.
 *
 * No es lo mismo que "Mi Red": ahí el endpoint devuelve el subárbol del
 * usuario con él en la raíz y nivel 0. Aquí GET /api/red devuelve la
 * jerarquía completa desde los nodos raíz, que es lo que necesita el admin
 * para ver la estructura.
 *
 * El dibujado del árbol y los totales están en RedArbol.tsx, compartidos con
 * "Mi Red" para que las dos vistas no puedan divergir.
 */

const RedGlobal = () => {
  const [arbol, setArbol] = useState<NodoRed[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const { texto, setTexto, filtrado } = useFiltroArbol(arbol);

  const cargar = useCallback(async () => {
    setError(null);
    try {
      setArbol(await getRedGlobalAPI());
    } catch (err) {
      setError(getErrorMessage(err));
      setArbol([]);
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    let vivo = true;
    void (async () => {
      try {
        const data = await getRedGlobalAPI();
        if (vivo) setArbol(data);
      } catch (err) {
        if (!vivo) return;
        setError(getErrorMessage(err));
        setArbol([]);
      } finally {
        if (vivo) setIsLoading(false);
      }
    })();
    return () => {
      vivo = false;
    };
  }, []);

  if (isLoading) {
    return (
      <div className="flex justify-center items-center min-h-96">
        <p className="text-slate-600 font-medium">Cargando la red global...</p>
      </div>
    );
  }

  const resumen = resumenRed(arbol);
  const resumenFiltrado = resumenRed(filtrado);

  return (
    <div>
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between mb-6 gap-4">
        <div>
          <h1 className="text-2xl font-bold text-slate-800">Red Global</h1>
          <p className="text-sm text-slate-500">
            Estructura completa de la organización
          </p>
        </div>
        <button
          onClick={cargar}
          className="text-xs bg-marca-50 text-marca-600 hover:bg-marca-100 px-3 py-2 rounded-md font-medium transition self-start min-h-[44px]"
        >
          Actualizar
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

      {!error && arbol.length === 0 ? (
        <div className="bg-white rounded-lg border border-slate-200 p-8 text-center text-slate-500">
          Todavía no hay distribuidores registrados en la red.
        </div>
      ) : (
        <>
          <ResumenCards resumen={resumen} />

          <div className="mb-6">
            <label htmlFor="buscar-red" className="sr-only">
              Buscar por nombre o código
            </label>
            <input
              id="buscar-red"
              type="search"
              value={texto}
              onChange={(e) => setTexto(e.target.value)}
              placeholder="Buscar por nombre o código..."
              className="w-full max-w-sm px-3 py-2 border border-slate-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-marca-500 min-h-[44px]"
            />
            {texto.trim() !== '' && (
              <p className="mt-2 text-xs text-slate-500">
                {resumenFiltrado.nodos} de {resumen.nodos} personas coinciden
              </p>
            )}
          </div>

          {filtrado.length === 0 ? (
            <div className="bg-white rounded-lg border border-dashed border-slate-300 p-6 text-center text-slate-500 text-sm">
              Ninguna persona coincide con «{texto}».
            </div>
          ) : (
            filtrado.map((raiz) => <RamaRed key={raiz.id} nodos={[raiz]} nivel={1} />)
          )}
        </>
      )}
    </div>
  );
};

export default RedGlobal;