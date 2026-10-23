// Capa de servicios para liquidaciones (admin).
// Sigue el mismo patrón que pedidosService: apiFetch con auto-Auth,
// auto-JSON y errores tipados.
import { apiFetch } from './http';
import type { DeudaDistribuidor, Liquidacion, Pagina } from '../types/api';

export interface LiquidacionesQuery {
  page?: number;
  limit?: number;
}

/**
 * Deuda pendiente por distribuidor.
 *
 * Solo incluye comisiones que se pueden cobrar todavía: el backend excluye
 * las ya pagadas y las revertidas, así que el `total` de cada fila es
 * exactamente lo que sepagaría con un POST a /liquidaciones.
 */
export const getDeuda = (): Promise<DeudaDistribuidor[]> =>
  apiFetch<DeudaDistribuidor[]>('/api/admin/liquidaciones/deuda', { method: 'GET' });

/** Historial de pagos, paginado. */
export const listLiquidaciones = (q: LiquidacionesQuery = {}): Promise<Pagina<Liquidacion>> =>
  apiFetch<Pagina<Liquidacion>>(
    `/api/admin/liquidaciones?${buildQS(q as Record<string, unknown>)}`,
    { method: 'GET' },
  );

/**
 * Paga la deuda completa de un distribuidor.
 *
 * Mueve dinero y no es idempotente por diseño: el backend liquida todas las
 * comisiones pendientes del referido y devuelve 409 si no queda ninguna, de
 * modo que un doble clic accidental no paga dos veces — la segunda llamada
 * se topa con que ya no hay deuda. Aun así la pantalla pide confirmación
 * antes de llamar.
 */
export const liquidar = (referidoId: number): Promise<Liquidacion> =>
  apiFetch<Liquidacion>('/api/admin/liquidaciones', {
    method: 'POST',
    body: { referido_id: referidoId },
  });

function buildQS(q: Record<string, unknown>): string {
  const sp = new URLSearchParams();
  for (const [k, v] of Object.entries(q)) {
    if (v !== undefined && v !== null && v !== '') sp.set(k, String(v));
  }
  return sp.toString();
}