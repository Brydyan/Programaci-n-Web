import { createContext, useContext, useState, type ReactNode } from 'react';
import { loginAPI } from '../services/api';
import { initializeAuth } from './authHelpers';
import type { Usuario, AuthContextType } from './authTypes';

export type { Usuario, AuthContextType };

const AuthContext = createContext<AuthContextType | undefined>(undefined);

// eslint-disable-next-line react-refresh/only-export-components
export const useAuth = () => {
    const context = useContext(AuthContext);
    if (!context) {
        throw new Error("useAuth debe ser usado dentro de un AuthProvider");
    }
    return context;
};

interface AuthProviderProps {
    children: ReactNode;
}

export const AuthProvider = ({ children }: AuthProviderProps) => {
    const [authState, setAuthState] = useState(() => initializeAuth());
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const login = async (email: string, password: string) => {
        setIsLoading(true);
        setError(null);
        try {
            const data = await loginAPI(email, password);
            localStorage.setItem("token", data.token);
            localStorage.setItem("userEmail", data.email);
            localStorage.setItem("userRole", data.rol);
            setAuthState({ isAuthenticated: true, user: { email: data.email, rol: data.rol } });
        } catch (err) {
            setError(err instanceof Error ? err.message : "Error al iniciar sesión");
            throw err;
        } finally {
            setIsLoading(false);
        }
    };

    const logout = () => {
        localStorage.removeItem("token");
        localStorage.removeItem("userEmail");
        localStorage.removeItem("userRole");
        setAuthState({ isAuthenticated: false, user: null });
        setError(null);
    };

    return (
        <AuthContext.Provider value={{ isAuthenticated: authState.isAuthenticated, user: authState.user, login, logout, isLoading, error }}>
            {children}
        </AuthContext.Provider>
    );
};
