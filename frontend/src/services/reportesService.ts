// Capa de servicios para /api/reportes/* (admin only).
// Sigue la misma convención que pedidosService: apiFetch + query-string
// construida a mano, sin dependencias extra.
import { apiFetch } from './http';
import type { Comision, ReporteResumen, TopProducto, TopReferido } from '../types/api';

export interface ReportesQuery {
  /** YYYY-MM-DD. `hasta` es inclusivo para el usuario. */
  desde?: string;
  hasta?: string;
}

export interface RankingQuery {
  /** 1..50. El backend recorta fuera de rango, pero se manda ya válido. */
  limit?: number;
}

export const getResumen = (q: ReportesQuery = {}): Promise<ReporteResumen> =>
  apiFetch<ReporteResumen>(`/api/reportes/resumen?${buildQS(q)}`, { method: 'GET' });

export const getTopProductos = (q: RankingQuery = {}): Promise<TopProducto[]> =>
  apiFetch<TopProducto[]>(`/api/reportes/top-productos?${buildQS(q)}`, { method: 'GET' });

export const getTopReferidos = (q: RankingQuery = {}): Promise<TopReferido[]> =>
  apiFetch<TopReferido[]>(`/api/reportes/top-referidos?${buildQS(q)}`, { method: 'GET' });

export const getComisionesPendientes = (q: RankingQuery = {}): Promise<Comision[]> =>
  apiFetch<Comision[]>(`/api/reportes/comisiones-pendientes?${buildQS(q)}`, {
    method: 'GET',
  });

/**
 * Construye el query-string omitiendo vacíos.
 *
 * El parámetro es `object` y no `Record<string, unknown>` a propósito: una
 * interfaz sin index signature no es asignable a Record, y haría falta un
 * cast en cada llamada. `Object.entries` funciona igual sobre `object`.
 */
function buildQS(q: object): string {
  const sp = new URLSearchParams();
  for (const [k, v] of Object.entries(q)) {
    if (v !== undefined && v !== null && v !== '') sp.set(k, String(v));
  }
  return sp.toString();
}