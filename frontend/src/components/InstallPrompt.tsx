import { useEffect, useState } from 'react';

interface BeforeInstallPromptEvent extends Event {
  prompt: () => Promise<void>;
  userChoice: Promise<{ outcome: 'accepted' | 'dismissed' }>;
}

const KEY_DISMISSED = 'pwa:install-dismissed';

/**
 * Modal para instalar la app en la pantalla de inicio.
 *
 * - Se muestra como modal centrado cuando el navegador dispara beforeinstallprompt
 * - Al hacer "Ahora no", se oculta y marca como dismissed
 * - Se resetea cuando el usuario se vuelve a loguear (limpiado desde AuthContext)
 * - Desaparece automáticamente cuando el usuario instala la app
 *
 * `beforeinstallprompt` sólo existe cuando el navegador considera la app
 * instalable (manifest válido + service worker activo + HTTPS o localhost).
 * En escritorio Firefox o Safari nunca se dispara, y este componente no
 * renderiza nada — que es el comportamiento correcto, no un fallo.
 */
export default function InstallPrompt() {
  const [deferred, setDeferred] = useState<BeforeInstallPromptEvent | null>(null);
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    // Nunca insisto dos veces. El evento se puede lanzar una sola vez por
    // visita, así que registrarlo en un ref y leerlo en otro sitio es la
    // única forma de no perderlo si el componente monta tarde.
    let dismissed = false;
    try {
      dismissed = localStorage.getItem(KEY_DISMISSED) === '1';
    } catch {
      // localStorage bloqueado (modo privado estricto): se muestra igual.
    }

    const onBeforeInstall = (e: Event) => {
      e.preventDefault();
      setDeferred(e as BeforeInstallPromptEvent);
      if (!dismissed) setVisible(true);
    };

    const onInstalled = () => {
      setVisible(false);
      setDeferred(null);
      try {
        localStorage.removeItem(KEY_DISMISSED);
      } catch {
        /* sin localStorage no hay nada que limpiar */
      }
    };

    window.addEventListener('beforeinstallprompt', onBeforeInstall);
    window.addEventListener('appinstalled', onInstalled);

    return () => {
      window.removeEventListener('beforeinstallprompt', onBeforeInstall);
      window.removeEventListener('appinstalled', onInstalled);
    };
  }, []);

  const cerrar = () => {
    setVisible(false);
    try {
      localStorage.setItem(KEY_DISMISSED, '1');
    } catch {
      /* si no se puede guardar, se ocultará solo en esta sesión */
    }
  };

  const instalar = async () => {
    if (!deferred) return;
    await deferred.prompt();
    // `userChoice` es la única señal fiable: el usuario aceptó o rechazó.
    await deferred.userChoice;
    setVisible(false);
  };

  if (!visible || !deferred) return null;

  return (
    <>
      {/* Overlay oscuro de fondo — clickeable para cerrar */}
      <div
        className="fixed inset-0 z-40 bg-black/40"
        onClick={cerrar}
        role="presentation"
      />
      {/* Modal centrado */}
      <div
        className="fixed inset-0 z-50 flex items-center justify-center p-4"
        role="dialog"
        aria-label="Instalar MultiCatálogo"
      >
        <div className="w-full max-w-sm rounded-lg bg-white p-6 text-slate-900 shadow-2xl">
          <div className="mb-6">
            <h2 className="text-xl font-bold">Instala MultiCatálogo</h2>
            <p className="mt-2 text-sm text-slate-600">
              Tenla en la pantalla de inicio y abrela sin barra de navegador. Acceso rápido a tus pedidos, comisiones y red.
            </p>
          </div>
          <div className="flex gap-3">
            <button
              type="button"
              onClick={instalar}
              className="flex-1 rounded-md bg-marca-600 px-4 py-3 text-sm font-medium text-white hover:bg-marca-500 transition min-h-[44px]"
            >
              Instalar
            </button>
            <button
              type="button"
              onClick={cerrar}
              className="flex-1 rounded-md bg-slate-100 px-4 py-3 text-sm font-medium text-slate-700 hover:bg-slate-200 transition min-h-[44px]"
            >
              Ahora no
            </button>
          </div>
        </div>
      </div>
    </>
  );
}
