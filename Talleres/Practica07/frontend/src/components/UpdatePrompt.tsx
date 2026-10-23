import { useRegisterSW } from 'virtual:pwa-register/react';

/**
 * Aviso de nueva versión del service worker.
 *
 * Usa `useRegisterSW` del plugin en vez de registro manual + `setInterval` de
 * actualización cada 60s. Motivos:
 *
 *   - El polling a mano nunca limpia su intervalo: en HMR se acumulan varios.
 *   - El plan anterior, para "aplicar" la actualización, desregistraba TODAS
 *     las service workers. Eso borra el caché entero y obliga a reinstalar:
 *     un mazo donde bastaba `skipWaiting()`.
 *   - `registerType: 'prompt'` en vite.config.ts ya resuelve esto; sólo hay
 *     que_CONTROLARLO, no reimplementarlo.
 */
export default function UpdatePrompt() {
  const {
    needRefresh: [needRefresh, setNeedRefresh],
    updateServiceWorker,
  } = useRegisterSW({
    onRegisteredSW(url, reg) {
      // Comprobar si hay versión nueva al volver a la pestaña, no en un
      // intervalo que consume batería de fondo.
      if (!reg) return;
      document.addEventListener('visibilitychange', () => {
        if (document.visibilityState === 'visible') void reg.update();
      });
      // Silencia el aviso de "la página se sirve del caché" en desarrollo.
      if (import.meta.env.DEV && url) {
        console.debug(`[PWA] SW en ${url}`);
      }
    },
    onRegisterError(error) {
      console.error('[PWA] error registrando el service worker:', error);
    },
  });

  if (!needRefresh) return null;

  return (
    <div
      className="fixed inset-x-4 top-4 z-50 mx-auto max-w-sm rounded-lg bg-emerald-700 p-4 text-white shadow-xl"
      role="status"
      aria-live="polite"
    >
      <h2 className="mb-1 font-semibold">Hay una versión nueva</h2>
      <p className="mb-3 text-sm text-emerald-100">
        Recarga para aplicar la actualización.
      </p>
      <div className="flex gap-2">
        <button
          type="button"
          onClick={() => void updateServiceWorker(true)}
          className="flex-1 rounded-md bg-white px-3 py-2 text-sm font-medium text-emerald-800 hover:bg-emerald-50"
        >
          Actualizar
        </button>
        <button
          type="button"
          onClick={() => setNeedRefresh(false)}
          className="rounded-md bg-emerald-600 px-3 py-2 text-sm hover:bg-emerald-500"
        >
          Más tarde
        </button>
      </div>
    </div>
  );
}