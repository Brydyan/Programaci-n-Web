// Capa de servicios para endpoints de pedidos (cliente + admin).
// Usa apiFetch (auto-Auth + auto-JSON + errores tipados).
import { apiFetch } from './http';
import type { Pagina, Pedido, PedidoDetalle, PedidoResumen } from '../types/api';

export interface MisPedidosQuery {
  page?: number;
  limit?: number;
}
export interface AdminPedidosQuery {
  estado?: string;
  page?: number;
  limit?: number;
}
export type EstadoPedido = Pedido['estado'];

export const listMisPedidos = (q: MisPedidosQuery = {}): Promise<Pagina<PedidoResumen>> =>
  apiFetch<Pagina<PedidoResumen>>(`/api/pedidos/mios?${buildQS(q as Record<string, unknown>)}`, { method: 'GET' });

export const getPedidoDetalle = (id: number): Promise<PedidoDetalle> =>
  apiFetch<PedidoDetalle>(`/api/pedidos/${id}`, { method: 'GET' });

export const listPedidosAdmin = (q: AdminPedidosQuery = {}): Promise<Pagina<PedidoResumen>> =>
  apiFetch<Pagina<PedidoResumen>>(`/api/admin/pedidos?${buildQS(q as Record<string, unknown>)}`, { method: 'GET' });

export const cambiarEstadoPedido = (id: number, estado: EstadoPedido): Promise<void> =>
  apiFetch<void>(`/api/admin/pedidos/${id}/estado`, {
    method: 'PATCH',
    body: { estado },
  });

function buildQS(q: Record<string, unknown>): string {
  const sp = new URLSearchParams();
  for (const [k, v] of Object.entries(q)) {
    if (v !== undefined && v !== null && v !== '') sp.set(k, String(v));
  }
  return sp.toString();
}
