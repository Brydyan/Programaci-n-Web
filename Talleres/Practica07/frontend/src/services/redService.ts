// Servicios de la red: el subárbol del cliente y el árbol global del admin.
import { apiFetch } from './http';
import type { NodoRed } from '../types/api';

/** Subárbol del usuario autenticado. Él es el nodo raíz, con nivel 0. */
export const getRedMiaAPI = (): Promise<NodoRed[]> =>
  apiFetch<NodoRed[]>('/api/red/mia', { method: 'GET' });

/** Árbol completo de la organización. Admin only. */
export const getRedGlobalAPI = (): Promise<NodoRed[]> =>
  apiFetch<NodoRed[]>('/api/red', { method: 'GET' });