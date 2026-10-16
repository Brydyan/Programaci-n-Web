// Cliente HTTP único. Centraliza auth + parsing de errores tipados.
import { ApiError, type APIError } from '../types/api';

const TOKEN_KEY = 'token';

const BASE_URL: string =
  (import.meta.env.VITE_API_URL as string | undefined) ?? 'http://localhost:3000';

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY);
}
export function setToken(token: string | null): void {
  if (token === null) localStorage.removeItem(TOKEN_KEY);
  else localStorage.setItem(TOKEN_KEY, token);
}

interface ApiFetchInit extends Omit<RequestInit, 'body'> {
  body?: unknown; // se serializa a JSON si no es FormData
}

export async function apiFetch<T>(path: string, init: ApiFetchInit = {}): Promise<T> {
  const { body, headers, ...rest } = init;
  const finalHeaders = new Headers(headers);

  const isFormData = body instanceof FormData;
  if (!isFormData && body !== undefined && !finalHeaders.has('Content-Type')) {
    finalHeaders.set('Content-Type', 'application/json');
  }
  const token = getToken();
  if (token && !finalHeaders.has('Authorization')) {
    finalHeaders.set('Authorization', `Bearer ${token}`);
  }

  const url = path.startsWith('http') ? path : `${BASE_URL}${path}`;

  const res = await fetch(url, {
    ...rest,
    headers: finalHeaders,
    body: body === undefined ? undefined : isFormData ? body : JSON.stringify(body),
  });

  if (!res.ok) {
    let parsed: APIError | null = null;
    try {
      parsed = (await res.json()) as APIError;
    } catch {
      // cuerpo no parseable → fabricamos uno genérico
    }
    if (import.meta.env.DEV) {
      console.warn(`[apiFetch] ${rest.method ?? 'GET'} ${path} → ${res.status}`);
    }
    throw new ApiError(
      parsed ?? {
        status: res.status,
        code: 'INTERNAL',
        message: `Error ${res.status}`,
      },
    );
  }

  // 204 No Content
  if (res.status === 204) return undefined as T;
  return (await res.json()) as T;
}