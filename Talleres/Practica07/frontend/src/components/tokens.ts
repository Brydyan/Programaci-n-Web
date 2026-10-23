import type { EstadoComision, EstadoLiquidacion } from '../types/api';
import type { EstadoPedido } from '../services/pedidosService';

/**
 * Semántica de los estados: texto en español y tono de color, por dominio.
 *
 * Vive separado de Badge.tsx a propósito. Badge decide CÓMO se ve una
 * insignia; aquí se decide QUÉ significa cada estado. Mezclarlos obligaría a
 * que el componente del color supiera de pedidos, comisiones y liquidaciones,
 * y añadir un dominio tocaría la pieza visual.
 */

/** Los cinco tonos posibles. Badge los traduce a clases. */
export type Tono = 'neutro' | 'aviso' | 'info' | 'exito' | 'error';

export type DefinicionEstado<T extends string> = Record<T, { texto: string; tono: Tono }>;

/**
 * Estados de una comisión.
 *
 * pendiente → aviso: el dinero existe pero todavía no se puede cobrar.
 * revertida → neutro, no error: no es un fallo, es un pedido cancelado y el
 *   dinero devuelto. Pintarla en rojo haría pensar en un problema.
 */
export const ESTADO_COMISION: DefinicionEstado<EstadoComision> = {
  pendiente: { texto: 'Pendiente', tono: 'aviso' },
  aprobada: { texto: 'Aprobada', tono: 'info' },
  pagada: { texto: 'Pagada', tono: 'exito' },
  revertida: { texto: 'Revertida', tono: 'neutro' },
};

/** Estados de un pedido. El ciclo es pendiente → preparando → enviado → entregado. */
export const ESTADO_PEDIDO: DefinicionEstado<EstadoPedido> = {
  pendiente: { texto: 'Pendiente', tono: 'neutro' },
  preparando: { texto: 'Preparando', tono: 'aviso' },
  enviado: { texto: 'Enviado', tono: 'info' },
  entregado: { texto: 'Entregado', tono: 'exito' },
  cancelado: { texto: 'Cancelado', tono: 'error' },
};

export const ESTADO_LIQUIDACION: DefinicionEstado<EstadoLiquidacion> = {
  pendiente: { texto: 'Pendiente', tono: 'aviso' },
  pagada: { texto: 'Pagada', tono: 'exito' },
};