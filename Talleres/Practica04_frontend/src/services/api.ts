const API_URL = "http://localhost:3000";

interface LoginResponse {
  token: string;
  email: string;
}

interface Producto {
  id: number;
  nombre: string;
  precio: number;
  img: string;
}

export const loginAPI = async (email: string, password: string): Promise<LoginResponse> => {
  const res = await fetch(`${API_URL}/api/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
  });

  if (!res.ok) {
    const error = await res.json();
    throw new Error(error.error || "Login fallido");
  }

  return res.json();
};

export const getProductosAPI = async (): Promise<Producto[]> => {
  const token = localStorage.getItem("token");

  const res = await fetch(`${API_URL}/api/productos`, {
    headers: token ? { "Authorization": `Bearer ${token}` } : {},
  });

  if (!res.ok) {
    throw new Error("Error al obtener productos");
  }

  return res.json();
};
