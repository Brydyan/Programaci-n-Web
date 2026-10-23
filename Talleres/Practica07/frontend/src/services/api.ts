// Capa de servicios: funciones semánticas por dominio.
// Para peticiones nuevas, preferir apiFetch de ./http.
import type {
  ComisionResumen,
  EstadoComision,
  Health,
  Liquidacion,
  LoginResponse,
  NodoRed,
  Pagina,
  RegisterRequest,
  RegisterResponse,
  Rol,
} from '../types/api';
import { apiFetch, getToken } from './http';

const API_URL = 'http://localhost:3000';

// Mantenemos compatibilidad con funciones legacy (token manual en header)
// mientras migran los componentes a apiFetch.
function authHeader(): Record<string, string> {
  const t = getToken();
  return t ? { Authorization: `Bearer ${t}` } : {};
}

function legacyFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  Object.entries(authHeader()).forEach(([k, v]) => headers.set(k, v));
  return fetch(`${API_URL}${path}`, { ...init, headers }).then(async (res) => {
    if (!res.ok) {
      const body = await res.json().catch(() => ({} as Record<string, string>));
      throw new Error(body.message ?? 'Error de red');
    }
    return res.json() as Promise<T>;
  });
}

// ── Tipos legacy re-exportados para componentes existentes ──
// (la forma real la define types/api.ts; estos solo preservan imports viejos)
export interface Producto {
  id: number;
  nombre: string;
  precio: number;
  img: string;
}
// Payload de POST /api/checkout.
//
// Ya NO es "legacy": el backend endureció el contrato y `ciudad`/`telefono`
// pasaron a ser obligatorios (openapi.yaml → CheckoutRequest, `required`).
// `email` desapareció: la identidad sale del JWT y el DTO del backend la
// declara con `json:"-"`, así que mandarlo no sirve de nada y además
// sugería que el servidor lo usaría.
export interface CheckoutPayload {
  nombre?: string;
  direccion: string;
  ciudad: string;
  telefono: string;
  metodo_pago: string;
  items: { id: number; nombre: string; precio: number; cantidad: number }[];
}
export interface PedidoLegacy {
  id: number;
  total: number;
  estado: string;
}

export const loginAPI = async (email: string, password: string): Promise<LoginResponse> => {
  const res = await legacyFetch<Partial<LoginResponse>>('/api/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
  });
  const rol: Rol = (res.rol as Rol | undefined) ?? 'cliente';
  return { token: res.token ?? '', email: res.email ?? email, rol };
};

export const getProductosAPI = async (): Promise<Producto[]> =>
  legacyFetch<Producto[]>('/api/productos');

export const getProductoPorIdAPI = async (id: number | string): Promise<Producto> =>
  legacyFetch<Producto>(`/api/productos/${id}`);

export const checkoutAPI = async (
  payload: CheckoutPayload,
): Promise<PedidoLegacy> =>
  legacyFetch<PedidoLegacy>('/api/checkout', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });

// ── Nuevos (Sprint S1) usan apiFetch + tipos estrictos ──
export const registerAPI = (payload: RegisterRequest): Promise<RegisterResponse> =>
  apiFetch<RegisterResponse>('/api/register', { method: 'POST', body: payload });

export const healthAPI = (): Promise<Health> => apiFetch<Health>('/api/health', { method: 'GET' });

/**
 * Subárbol del usuario autenticado (Sprint 1). Usa apiFetch para que viaje el
 * token: /api/red/mia está detrás de RequireAuth y devuelve el árbol anidado.
 * `GET /api/red` (vista global) queda reservada al admin.
 */
export const getRedMiaAPI = (): Promise<NodoRed[]> =>
  apiFetch<NodoRed[]>('/api/red/mia', { method: 'GET' });

/** Ledger del usuario autenticado (Sprint 2). Sin paginación: el panel de
 *  "Mis comisiones" muestra las más recientes y los totales por estado. */
export const getComisionesMiasAPI = (params?: { estado?: EstadoComision }): Promise<ComisionResumen> => {
  const qs = params?.estado ? `?estado=${encodeURIComponent(params.estado)}` : '';
  return apiFetch<ComisionResumen>(`/api/comisiones/mias${qs}`, { method: 'GET' });
};

/**
 * Cobros del usuario autenticado. A diferencia de `/comisiones/mias`, este
 * endpoint sí pagina: devuelve {data, limit, page, total}.
 */
export const getLiquidacionesMiasAPI = (
  params?: { page?: number; limit?: number },
): Promise<Pagina<Liquidacion>> => {
  const qs = new URLSearchParams();
  if (params?.page) qs.set('page', String(params.page));
  if (params?.limit) qs.set('limit', String(params.limit));
  const sufijo = qs.toString() ? `?${qs.toString()}` : '';
  return apiFetch<Pagina<Liquidacion>>(`/api/liquidaciones/mias${sufijo}`, { method: 'GET' });
};