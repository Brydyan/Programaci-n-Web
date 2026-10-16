// Capa de servicios: funciones semánticas por dominio.
// Para peticiones nuevas, preferir apiFetch de ./http.
import type {
  Health,
  LoginResponse,
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
export interface Referido {
  id: number;
  nombre: string;
  nivel: string;
  ventas: string;
}
// payload que acepta el BE actual (se endurecerá en F3 con ciudad/telefono requeridos)
export interface CheckoutRequestLegacy {
  nombre: string;
  email: string;
  direccion: string;
  ciudad?: string;
  telefono?: string;
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

export const getRedAPI = async (): Promise<Referido[]> =>
  legacyFetch<Referido[]>('/api/red');

export const checkoutAPI = async (
  payload: CheckoutRequestLegacy,
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