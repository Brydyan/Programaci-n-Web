import type { ReactNode } from 'react';
import type { Tono } from './tokens';

/**
 * `<Badge>` — la insignia de estado del proyecto.
 *
 * Antes cada pantalla declaraba su propio `Record<Estado, { texto, clase }>`
 * con los literales de Tailwind dentro: seis archivos repetían el mapeo y
 * además no coincidían entre sí (las comisiones usaban `text-emerald-800` y
 * los pedidos `text-emerald-700`). Aquí el color vive una sola vez.
 *
 * El tono llega como prop, no como clase: si el componente aceptara
 * `className` libre, cada pantalla volvería a poder inventarse su propio
 * color.
 */

/*
 * Clases de cada tono. Vive aquí y no se exporta para que el archivo sólo
 * exporte componentes: si se exportara, el Fast Refresh de Vite dejaría de
 * funcionar sobre él y habría que buscar el color en dos sitios otra vez.
 * Para las "tarjetas de estado" más grandes se usa el componente <Tono>.
 */
const CLASES: Record<Tono, string> = {
  neutro: 'bg-estado-neutro-bg text-estado-neutro-fg',
  aviso: 'bg-estado-aviso-bg text-estado-aviso-fg',
  info: 'bg-estado-info-bg text-estado-info-fg',
  exito: 'bg-estado-exito-bg text-estado-exito-fg',
  error: 'bg-estado-error-bg text-estado-error-fg',
};

export const Badge = ({
  tono,
  children,
  titulo,
  className = '',
}: {
  tono: Tono;
  children: ReactNode;
  titulo?: string;
  className?: string;
}) => (
  <span
    title={titulo}
    className={`inline-flex items-center rounded px-2 py-0.5 text-xs font-medium whitespace-nowrap ${CLASES[tono]} ${className}`}
  >
    {children}
  </span>
);

/**
 * Insignia de un estado con su texto ya resuelto. Evita repetir
 * `ESTADO_COMISION[c.estado].texto` en cada fila de tabla.
 */
export const BadgeEstado = ({
  definicion,
  children,
}: {
  definicion: { texto: string; tono: Tono };
  children?: ReactNode;
}) => (
  <Badge tono={definicion.tono} titulo={definicion.texto}>
    {children ?? definicion.texto}
  </Badge>
);

/**
 * Bloque con el fondo de un tono. Lo usan las tarjetas de totales por
 * estado, que son superficies y no insignias: comparten color con el Badge
 * sin necesitar las clases a mano.
 */
export const TonoBloque = ({
  tono,
  children,
  className = '',
}: {
  tono: Tono;
  children: ReactNode;
  className?: string;
}) => (
  <div className={`rounded-md px-3 py-2 ${CLASES[tono]} ${className}`}>{children}</div>
);

export default Badge;