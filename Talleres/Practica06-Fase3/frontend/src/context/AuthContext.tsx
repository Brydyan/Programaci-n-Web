import { createContext, useContext, useState, type ReactNode } from 'react';
import { loginAPI } from '../services/api';

interface AuthContextType {
    isAuthenticated: boolean;
    userEmail: string | null;
    login: (email: string, password: string) => Promise<void>;
    logout: () => void;
    isLoading: boolean;
    error: string | null;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

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
    const [isAuthenticated, setIsAuthenticated] = useState<boolean>(false);
    const [userEmail, setUserEmail] = useState<string | null>(null);
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const login = async (email: string, password: string) => {
        setIsLoading(true);
        setError(null);
        try {
            const data = await loginAPI(email, password);
            localStorage.setItem("token", data.token);
            setIsAuthenticated(true);
            setUserEmail(data.email);
        } catch (err) {
            setError(err instanceof Error ? err.message : "Error al iniciar sesión");
            throw err;
        } finally {
            setIsLoading(false);
        }
    };

    const logout = () => {
        localStorage.removeItem("token");
        setIsAuthenticated(false);
        setUserEmail(null);
        setError(null);
    };

    return (
        <AuthContext.Provider value={{ isAuthenticated, userEmail, login, logout, isLoading, error }}>
            {children}
        </AuthContext.Provider>
    );
};
