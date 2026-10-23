// Servicios del ledger de comisiones (cliente + admin).
import { apiFetch } from './http';
import type { ComisionResumen, EstadoComision } from '../types/api';

export interface AdminComisionesQuery {
  estado?: EstadoComision;
  referido_id?: number;
  page?: number;
  limit?: number;
}

/**
 * Ledger completo para auditoría. Admin only.
 *
 * Un detalle del contrato que la pantalla respeta: los `totales` de la
 * respuesta son GLOBALES y no se recalculan con `estado` ni con
 * `referido_id`. Filtrar cambia `data` y `total`, pero no `totales`.
 */
export const getComisionesAdmin = (
  q: AdminComisionesQuery = {},
): Promise<ComisionResumen> =>
  apiFetch<ComisionResumen>(`/api/admin/comisiones?${buildQS(q)}`, { method: 'GET' });

function buildQS(q: AdminComisionesQuery): string {
  const sp = new URLSearchParams();
  if (q.estado) sp.set('estado', q.estado);
  if (q.referido_id !== undefined && q.referido_id !== null) {
    sp.set('referido_id', String(q.referido_id));
  }
  if (q.page !== undefined) sp.set('page', String(q.page));
  if (q.limit !== undefined) sp.set('limit', String(q.limit));
  return sp.toString();
}