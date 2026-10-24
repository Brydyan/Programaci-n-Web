import { useCallback, useEffect, useRef, useState } from 'react';
import { getErrorMessage } from '../types/api';

export interface ReportesState<T> {
  data: T | null;
  /** Hay una petición en vuelo initiated por el usuario (recarga manual). */
  cargando: boolean;
  error: string | null;
  /** Fuerza un refetch. */
  reload: () => void;
}

/**
 * Carga un reporte con estados de carga/error y protección contra carreras.
 *
 * El guard de request-id es lo importante: al cambiar los filtros salen varias
 * peticiones y las respuestas pueden volver desordenadas. Sin este conteo, la
 * petición lenta del filtro anterior termina después y pisa la tabla con los
 * datos viejos — un fallo que sólo se ve en producción, con red lenta, y que el
 * usuario reporta como "el filtro no funciona".
 *
 * No se usa AbortController porque apiFetch no expone la señal; el guard de
 * identidad cubre el caso visible y no deja código muerto.
 *
 * `cargando` cubre la carga inicial y la recarga manual, NO un cambio de `deps`.
 * Eso es deliberado: es un stale-while-revalidate. Al cambiar el filtro se
 * sigue viendo el dato anterior hasta que llega el nuevo, en vez de parpadear
 * un spinner y dejar la tabla vacía un instante.
 *
 * @param fetcher Función que devuelve la promesa del reporte.
 * @param deps    Parámetros que, al cambiar, relanzan la petición.
 */
export function useReportes<T>(
  fetcher: () => Promise<T>,
  deps: readonly unknown[] = [],
): ReportesState<T> {
  const [data, setData] = useState<T | null>(null);
  const [cargando, setCargando] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [tick, setTick] = useState<number>(0);

  // El fetcher se guarda en un ref para no depender de su identidad: si fuera
  // una dependencia del efecto, se recrearía en cada render y el efecto no
  // pararía nunca. Se sincroniza en su propio efecto, declarado ANTES del de
  // carga para que el valor esté al día cuando lo lee el segundo.
  const fetcherRef = useRef(fetcher);
  useEffect(() => {
    fetcherRef.current = fetcher;
  });

  const reqId = useRef(0);

  useEffect(() => {
    const id = ++reqId.current;

    fetcherRef
      .current()
      .then((res) => {
        if (id !== reqId.current) return; // superseded por otra petición
        setData(res);
        setError(null);
      })
      .catch((err: unknown) => {
        if (id !== reqId.current) return;
        setError(getErrorMessage(err));
      })
      .finally(() => {
        if (id === reqId.current) setCargando(false);
      });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tick, ...deps]);

  const reload = useCallback(() => {
    setCargando(true);
    setTick((t) => t + 1);
  }, []);

  return { data, cargando, error, reload };
}