export const initializeAuth = () => {
    const token = localStorage.getItem("token");
    const email = localStorage.getItem("userEmail");
    const role = localStorage.getItem("userRole");

    if (token && email && role) {
        return {
            isAuthenticated: true,
            user: { email, rol: role as "admin" | "cliente" }
        };
    }
    return { isAuthenticated: false, user: null };
};
