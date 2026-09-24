export interface Usuario {
    email: string;
    rol: "admin" | "cliente";
}

export interface AuthContextType {
    isAuthenticated: boolean;
    user: Usuario | null;
    login: (email: string, password: string) => Promise<void>;
    logout: () => void;
    isLoading: boolean;
    error: string | null;
}
