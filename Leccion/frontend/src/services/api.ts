import type { Producto } from "../data/productos";

const API_URL = "http://localhost:3000";

export interface LoginResponse {
  token: string;
  email: string;
  rol: "admin" | "cliente";
}

export type { Producto };

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

  try {
    const res = await fetch(`${API_URL}/api/productos`, {
      headers: token ? { "Authorization": `Bearer ${token}` } : {},
    });

    if (res.ok) {
      return res.json();
    }
  } catch (err) {
    console.warn("Backend no disponible, usando mock", err);
  }

  // Fallback a mock si backend falla o no está disponible
  const { productosMock } = await import("../data/productos");
  return productosMock;
};
