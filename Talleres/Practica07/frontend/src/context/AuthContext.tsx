import { useState, type ReactNode } from 'react';
import { loginAPI, registerAPI } from '../services/api';
import { purgarCachePWA } from '../services/pwaCache';
import {
  AuthContext,
  EMAIL_KEY,
  ROL_KEY,
  TOKEN_KEY,
  readRol,
} from './auth';
import type { LoginResponse, Rol } from '../types/api';

interface AuthProviderProps {
  children: ReactNode;
}

/**
 * Proveedor de sesión. Guarda token/email/rol en localStorage y expone
 * login, register y logout. El hook `useAuth` vive en `auth.ts`.
 */
export const AuthProvider = ({ children }: AuthProviderProps) => {
  const [isAuthenticated, setIsAuthenticated] = useState<boolean>(
    () => !!localStorage.getItem(TOKEN_KEY),
  );
  const [userEmail, setUserEmail] = useState<string | null>(
    () => localStorage.getItem(EMAIL_KEY),
  );
  const [rol, setRol] = useState<Rol | null>(() => readRol());
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const login = async (email: string, password: string): Promise<LoginResponse> => {
    setIsLoading(true);
    setError(null);
    try {
      const data = await loginAPI(email, password);
      localStorage.setItem(TOKEN_KEY, data.token);
      localStorage.setItem(EMAIL_KEY, data.email);
      localStorage.setItem(ROL_KEY, data.rol);
      setIsAuthenticated(true);
      setUserEmail(data.email);
      setRol(data.rol);
      return data;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Error al iniciar sesión');
      throw err;
    } finally {
      setIsLoading(false);
    }
  };

  const logout = () => {
    localStorage.removeItem(TOKEN_KEY);
    localStorage.removeItem(EMAIL_KEY);
    localStorage.removeItem(ROL_KEY);
    // Purga el catálogo cacheado por el service worker. Se dispara sin await
    // a propósito: el cambio de pantalla no debe esperar a una promesa de
    // red/almacenamiento, y si falla no hay nada que recuperar (el catálogo es
    // público y se vuelve a pedir al backend).
    void purgarCachePWA();
    setIsAuthenticated(false);
    setUserEmail(null);
    setRol(null);
    setError(null);
  };

  const register = async (nombre: string, email: string, password: string, ref?: string) => {
    setError(null);
    setIsLoading(true);
    try {
      await registerAPI({ nombre, email, password, ref });
      await login(email, password);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Error al registrar');
      throw err;
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <AuthContext.Provider
      value={{ isAuthenticated, userEmail, rol, login, register, logout, isLoading, error }}
    >
      {children}
    </AuthContext.Provider>
  );
};