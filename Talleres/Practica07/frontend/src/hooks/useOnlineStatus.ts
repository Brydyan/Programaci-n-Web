import { useEffect, useState } from 'react';

/**
 * Estado de conexión, sin peticiones de red.
 *
 * Se usa `navigator.onLine` y NO un `fetch('/api/health')` para comprobarlo.
 * La razón es concreta, no doctrinal:
 *
 *   - `navigator.onLine` responde en microsegundos. Un fetch de sondeo añade
 *     una request completa ANTES de cada lectura, es decir duplica la
 *     latencia de la app para producir un dato que ya está disponible.
 *   - `navigator.onLine` sólo dice si hay interfaz de red, que es
 *     exactamente lo que necesita la UI para decidir si avisar. No pretende
 *     saber si el backend responde: para eso está el error de la petición,
 *     que la app ya sabe manejar.
 *
 * Limitación conocida: un enlace Wi-Fi conectado sin salida a internet hace
 * que `navigator.onLine` devuelva `true`. Por eso el banner NO depende sólo
 * de este valor para bloqueear acciones — el fallo real de red lo reporta la
 * propia llamada y lo muestra cada componente.
 */
export function useOnlineStatus(): boolean {
  const [online, setOnline] = useState<boolean>(() =>
    typeof navigator === 'undefined' ? true : navigator.onLine,
  );

  useEffect(() => {
    const goOnline = () => setOnline(true);
    const goOffline = () => setOnline(false);

    window.addEventListener('online', goOnline);
    window.addEventListener('offline', goOffline);

    return () => {
      window.removeEventListener('online', goOnline);
      window.removeEventListener('offline', goOffline);
    };
  }, []);

  return online;
}