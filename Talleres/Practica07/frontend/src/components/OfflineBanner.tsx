import { MdCloudOff } from 'react-icons/md';
import { useOnlineStatus } from '../hooks/useOnlineStatus';

/**
 * Aviso de "estás sin conexión".
 *
 * Es informativo, no bloqueante: la app sigue navegable porque el catálogo
 * público se sirve desde el service worker. Lo que NO funciona offline son
 * pedidos, comisiones, red, reportes y el checkout, y eso lo reporta cada
 * componente al fallar su propia llamada.
 *
 * Decirlo aquí evita el fallo silencioso más común: el usuario ve una pantalla
 * vacía, asume que no hay datos y no entiende que el problema es la red.
 */
export default function OfflineBanner() {
  const online = useOnlineStatus();

  if (online) return null;

  return (
    <div
      className="flex items-center justify-center gap-2 bg-amber-400 px-4 py-2 text-sm font-medium text-amber-950"
      role="alert"
      aria-live="assertive"
    >
      <MdCloudOff aria-hidden />
      <span>
        Sin conexión. Puedes ver el catálogo; los pedidos y el pago necesitan red.
      </span>
    </div>
  );
}