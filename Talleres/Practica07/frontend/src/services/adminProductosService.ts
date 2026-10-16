// Capa de servicios para endpoints admin de productos.
// Usa apiFetch (auto-Auth + auto-JSON + errores tipados).
import { apiFetch } from './http';
import type { Pagina, Producto, ProductoCreate, ProductoUpdate } from '../types/api';

export interface AdminProductosQuery {
  q?: string;
  activo?: 'all' | 'true' | 'false';
  page?: number;
  limit?: number;
}

export const listProductosAdmin = (q: AdminProductosQuery): Promise<Pagina<Producto>> =>
  apiFetch<Pagina<Producto>>(`/api/admin/productos?${buildQS(q)}`, { method: 'GET' });

export const createProducto = (d: ProductoCreate): Promise<Producto> =>
  apiFetch<Producto>('/api/admin/productos', { method: 'POST', body: d });

export const updateProducto = (id: number, d: ProductoUpdate): Promise<Producto> =>
  apiFetch<Producto>(`/api/admin/productos/${id}`, { method: 'PUT', body: d });

export const softDeleteProducto = (id: number): Promise<void> =>
  apiFetch<void>(`/api/admin/productos/${id}`, { method: 'DELETE' });

function buildQS(q: AdminProductosQuery): string {
  const sp = new URLSearchParams();
  if (q.q) sp.set('q', q.q);
  if (q.activo && q.activo !== 'all') sp.set('activo', q.activo);
  if (q.page) sp.set('page', String(q.page));
  if (q.limit) sp.set('limit', String(q.limit));
  return sp.toString();
}
