import { MdErrorOutline } from 'react-icons/md';

interface EstadoProps {
  /** Hay una petición en vuelo. */
  cargando: boolean;
  error: string | null;
  /** Ya hay respuesta y no hay nada que pintar. */
  vacio: boolean;
  /** Qué es lo que no hay, para el mensaje: "productos", "comisiones"... */
  etiqueta: string;
}

/**
 * Estados de carga / error / vacío de un widget.
 *
 * Los tres se resuelven en un componente porque el error más común de un
 * dashboard no es que falle: es que pinte "0 resultados" cuando en realidad la
 * petición se cayó, y el admin se lleve una conclusión falsa. Por eso el error
 * tiene su propio estado visible y no se confunde con un dataset vacío.
 */
export default function EstadoWidget({
  cargando,
  error,
  vacio,
  etiqueta,
}: EstadoProps) {
  if (cargando) {
    return (
      <div
        className="py-8 text-center text-sm text-slate-400"
        role="status"
        aria-live="polite"
      >
        Cargando {etiqueta}…
      </div>
    );
  }

  if (error) {
    return (
      <div
        className="flex items-start gap-2 rounded-md bg-rose-50 p-4 text-sm text-rose-700"
        role="alert"
      >
        <MdErrorOutline className="mt-0.5 shrink-0" aria-hidden />
        <span>
          No se pudieron cargar {etiqueta}: {error}
        </span>
      </div>
    );
  }

  if (vacio) {
    return (
      <p className="py-8 text-center text-sm text-slate-400">
        No hay {etiqueta} en este periodo.
      </p>
    );
  }

  return null;
}