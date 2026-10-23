import { useEffect, useState } from 'react';

interface BeforeInstallPromptEvent extends Event {
  prompt: () => Promise<void>;
  userChoice: Promise<{ outcome: 'accepted' | 'dismissed' }>;
}

const KEY_DISMISSED = 'pwa:install-dismissed';

/**
 * Aviso para instalar la app en la pantalla de inicio.
 *
 * `beforeinstallprompt` sólo existe cuando el navegador considera la app
 * instalable (manifest válido + service worker activo + HTTPS o localhost).
 * En escritorio Firefox o Safari nunca se dispara, y este componente no
 * renderiza nada — que es el comportamiento correcto, no un fallo.
 *
 * La dismissal se guarda en localStorage: sin eso, el aviso reaparece en cada
 * navegación y molesta más de lo que informa. No se vuelve a mostrar en la
 * misma instalación; se limpia si el usuario la instala desde el navegador.
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
    <div
      className="fixed inset-x-4 bottom-4 z-50 mx-auto max-w-sm rounded-lg bg-slate-800 p-4 text-white shadow-xl"
      role="dialog"
      aria-label="Instalar MultiCatálogo"
    >
      <h2 className="mb-1 font-semibold">Instala MultiCatálogo</h2>
      <p className="mb-3 text-sm text-slate-300">
        Tenla en la pantalla de inicio y ábrela sin barra de navegador.
      </p>
      <div className="flex gap-2">
        <button
          type="button"
          onClick={instalar}
          className="flex-1 rounded-md bg-indigo-600 px-3 py-2 text-sm font-medium hover:bg-indigo-500"
        >
          Instalar
        </button>
        <button
          type="button"
          onClick={cerrar}
          className="rounded-md bg-slate-700 px-3 py-2 text-sm hover:bg-slate-600"
        >
          Ahora no
        </button>
      </div>
    </div>
  );
}