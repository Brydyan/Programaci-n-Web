const API_URL = "http://localhost:3000";

export interface LoginResponse {
  token: string;
  email: string;
  rol: "admin" | "cliente";
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
