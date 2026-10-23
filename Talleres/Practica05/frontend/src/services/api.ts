const API_URL = "http://localhost:3000";

export interface LoginResponse {
  token: string;
  email: string;
}

export interface Producto {
  id: number;
  nombre: string;
  precio: number;
  img: string;
}

export interface Referido {
  id: number;
  nombre: string;
  nivel: string;
  ventas: string;
}

export const loginAPI = async (email: string, password: string): Promise<LoginResponse> => {
  const res = await fetch(`${API_URL}/api/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
  });

  if (!res.ok) {
    const error = await res.json().catch(() => ({}));
    throw new Error(error.message || error.error || "Login fallido");
  }

  return res.json();
};

export const getProductosAPI = async (): Promise<Producto[]> => {
  const token = localStorage.getItem("token");

  const res = await fetch(`${API_URL}/api/productos`, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  });

  if (!res.ok) {
    const error = await res.json().catch(() => ({}));
    throw new Error(error.message || error.error || "Error al obtener productos");
  }

  return res.json();
};

export const getProductoPorIdAPI = async (id: number | string): Promise<Producto> => {
  const token = localStorage.getItem("token");

  const res = await fetch(`${API_URL}/api/productos/${id}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  });

  if (!res.ok) {
    const error = await res.json().catch(() => ({}));
    throw new Error(error.message || error.error || "Error al obtener el producto");
  }

  return res.json();
};

export const getRedAPI = async (): Promise<Referido[]> => {
  const token = localStorage.getItem("token");

  const res = await fetch(`${API_URL}/api/red`, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  });

  if (!res.ok) {
    const error = await res.json().catch(() => ({}));
    throw new Error(error.message || error.error || "Error al obtener la red");
  }

  return res.json();
};
