import { createContext, useContext, useState, useEffect, type ReactNode } from 'react';
import { loginAPI } from '../services/api';

interface AuthContextType {
    isAuthenticated: boolean;
    userEmail: string | null;
    userRole: "admin" | "cliente" | null;
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
    const [userRole, setUserRole] = useState<"admin" | "cliente" | null>(null);
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    // Restaurar sesión desde localStorage al montar
    useEffect(() => {
        const token = localStorage.getItem("token");
        const email = localStorage.getItem("userEmail");
        const role = localStorage.getItem("userRole");

        if (token && email && role) {
            setIsAuthenticated(true);
            setUserEmail(email);
            setUserRole(role as "admin" | "cliente");
        }
    }, []);

    const login = async (email: string, password: string) => {
        setIsLoading(true);
        setError(null);
        try {
            const data = await loginAPI(email, password);
            localStorage.setItem("token", data.token);
            localStorage.setItem("userEmail", data.email);
            localStorage.setItem("userRole", data.rol);
            setIsAuthenticated(true);
            setUserEmail(data.email);
            setUserRole(data.rol);
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
        setIsAuthenticated(false);
        setUserEmail(null);
        setUserRole(null);
        setError(null);
    };

    return (
        <AuthContext.Provider value={{ isAuthenticated, userEmail, userRole, login, logout, isLoading, error }}>
            {children}
        </AuthContext.Provider>
    );
};
