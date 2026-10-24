import type { ReactNode } from 'react';

interface Props {
  titulo: string;
  children: ReactNode;
  /** Acción opcional en la cabecera (filtros, botón recargar). */
  acciones?: ReactNode;
}

/**
 * Tarjeta compartida por los cuatro widgets del dashboard.
 *
 * Un solo contenedor en vez de cuatro copias de las mismas clases: si el
 * espaciado o la sombra cambian, se cambian una vez.
 */
export default function Tarjeta({ titulo, children, acciones }: Props) {
  return (
    <section className="rounded-lg border border-slate-100 bg-white p-6 shadow-sm">
      <header className="mb-4 flex items-center justify-between gap-4">
        <h2 className="text-lg font-semibold text-slate-800">{titulo}</h2>
        {acciones}
      </header>
      {children}
    </section>
  );
}