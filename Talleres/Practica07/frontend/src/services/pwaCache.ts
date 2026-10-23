// Purga de cachés del service worker.
//
// Existe por una razón concreta: el catálogo público es lo ÚNICO que el
// service worker cachea (ver vite.config.ts → runtimeCaching). No hay datos de
// sesión en caché, así que cerrar sesión no es una fuga urgente… salvo por un
// detalle: el catálogo cacheado puede venir de una respuesta que el backend dio
// a un usuario autenticado, y `GET /api/productos` es público pero el backend lo
// sirve con el mismo handler en ambos casos.
//
// Purgarlo en el logout deja el estado limpio y es barato: es una promesa que
// se resuelve en microsegundos y no bloquea el cambio de pantalla. Si no hay
// service worker (dev, navegadores sin soporte), no-op silencioso.

const CACHE_NOMBRES = ['catalogo-publico'];

/**
 * Elimina las cachés que la PWA gestiona.
 *
 * Se exporta por separado para poder llamarla desde tests y desde el logout.
 * Nunca lanza: un fallo al purgar no debe impedir cerrar sesión.
 */
export async function purgarCachePWA(): Promise<void> {
  if (typeof caches === 'undefined') return;

  try {
    await Promise.all(CACHE_NOMBRES.map((nombre) => caches.delete(nombre)));
  } catch (err) {
    console.warn('[PWA] no se pudo purgar la caché:', err);
  }
}