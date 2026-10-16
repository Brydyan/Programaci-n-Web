import { createContext, useContext, useState, type ReactNode } from 'react';
import { loginAPI, registerAPI } from '../services/api';
import type { LoginResponse, Rol } from '../types/api';

interface AuthContextType {
  isAuthenticated: boolean;
  userEmail: string | null;
  rol: Rol | null;
  login: (email: string, password: string) => Promise<LoginResponse>;
  register: (nombre: string, email: string, password: string, ref?: string) => Promise<void>;
  logout: () => void;
  isLoading: boolean;
  error: string | null;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

const TOKEN_KEY = 'token';
const EMAIL_KEY = 'userEmail';
const ROL_KEY = 'userRol';

const readRol = (): Rol | null => {
  const r = localStorage.getItem(ROL_KEY);
  return r === 'admin' || r === 'cliente' ? r : null;
};

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth debe ser usado dentro de un AuthProvider');
  }
  return context;
};

interface AuthProviderProps {
  children: ReactNode;
}

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