import { createContext, useContext } from 'react';
import type { LoginResponse, Rol } from '../types/api';

/**
 * Tipos, objeto de contexto y hook de sesión.
 *
 * Separado de `AuthContext.tsx` (sólo Provider) para el Fast Refresh de Vite.
 *
 * Claves de localStorage: el token vive aquí, no en el Provider, para que
 * ambos archivos compartan una sola definición.
 */

export interface AuthContextType {
  isAuthenticated: boolean;
  userEmail: string | null;
  rol: Rol | null;
  login: (email: string, password: string) => Promise<LoginResponse>;
  register: (nombre: string, email: string, password: string, ref?: string) => Promise<void>;
  logout: () => void;
  isLoading: boolean;
  error: string | null;
}

export const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const TOKEN_KEY = 'token';
export const EMAIL_KEY = 'userEmail';
export const ROL_KEY = 'userRol';

/** Lee el rol persistido; cualquier valor fuera del enum devuelve null. */
export const readRol = (): Rol | null => {
  const r = localStorage.getItem(ROL_KEY);
  return r === 'admin' || r === 'cliente' ? r : null;
};

/** Acceso a la sesión. Lanza si se usa fuera del `AuthProvider`. */
export const useAuth = (): AuthContextType => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth debe ser usado dentro de un AuthProvider');
  }
  return context;
};