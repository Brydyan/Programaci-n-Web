import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthProvider } from './AuthContext';
import { useAuth, EMAIL_KEY, ROL_KEY, TOKEN_KEY, readRol } from './auth';

// La sesión decide qué ve el usuario: rutas, botones, el dashboard entero. Un
// fallo aquí no da una pantalla rota, da acceso indebido o un 403 puzzling.
//
// Se mockea `services/api` en vez de fetch: aísla el contexto de la capa de
// red, que ya tiene sus propios tests en el backend. Si estos tests fallan
// cuando el backend cambia una URL, están probando lo equivocado.

const loginAPI = vi.fn();
const registerAPI = vi.fn();

vi.mock('../services/api', () => ({
  loginAPI: (...args: unknown[]) => loginAPI(...args),
  registerAPI: (...args: unknown[]) => registerAPI(...args),
}));

// El logout purga la caché del service worker. En jsdom `caches` no existe, y
// pwaCache lo trata como no-op, así que no hace falta mock: si algún día
// empieza a fallar, es que ese no-op dejó de ser cierto.
function MontarSesion() {
  const { isAuthenticated, userEmail, rol, login, logout, isLoading, error } = useAuth();

  return (
    <div>
      {/* `login` vuelve a lanzar el error a propósito para que cada pantalla
          pueda reaccionar (ver Login.tsx:24-29). El harness replica ese
          try/catch: si no, la promesa rechazada queda sin manejar y Vitest la
          reporta como Unhandled Rejection — ruido que oculta fallos reales. */}
      <button
        onClick={() => {
          void login('ana@x.com', '12345678').catch(() => undefined);
        }}
      >
        Entrar
      </button>
      <button onClick={logout}>Salir</button>
      <span data-testid="auth">{String(isAuthenticated)}</span>
      <span data-testid="email">{userEmail ?? '-'}</span>
      <span data-testid="rol">{rol ?? '-'}</span>
      <span data-testid="loading">{String(isLoading)}</span>
      <span data-testid="error">{error ?? '-'}</span>
    </div>
  );
}

const montar = () => render(<AuthProvider><MontarSesion /></AuthProvider>);
const txt = (id: string) => screen.getByTestId(id).textContent;

beforeEach(() => {
  loginAPI.mockReset();
  registerAPI.mockReset();
  localStorage.clear();
});

describe('readRol', () => {
  it('acepta sólo los roles del contrato', () => {
    localStorage.setItem(ROL_KEY, 'admin');
    expect(readRol()).toBe('admin');

    localStorage.setItem(ROL_KEY, 'cliente');
    expect(readRol()).toBe('cliente');
  });

  // Si alguien manipula localStorage, un rol arbitrario no debe colarse como
  // admin. Devolver el valor crudo sería confiar en el cliente.
  it('devuelve null para un rol desconocido o ausente', () => {
    localStorage.setItem(ROL_KEY, 'superadmin');
    expect(readRol()).toBeNull();

    localStorage.removeItem(ROL_KEY);
    expect(readRol()).toBeNull();
  });
});

describe('AuthProvider — estado inicial', () => {
  it('arranca sin sesión si no hay token', () => {
    montar();
    expect(txt('auth')).toBe('false');
    expect(txt('email')).toBe('-');
    expect(txt('rol')).toBe('-');
  });

  // Refrescar la página no debe cerrar la sesión: por eso el estado inicial
  // lee de localStorage y no de useState(false).
  it('restaura la sesión desde localStorage', () => {
    localStorage.setItem(TOKEN_KEY, 'jwt-abc');
    localStorage.setItem(EMAIL_KEY, 'ana@x.com');
    localStorage.setItem(ROL_KEY, 'cliente');

    montar();
    expect(txt('auth')).toBe('true');
    expect(txt('email')).toBe('ana@x.com');
    expect(txt('rol')).toBe('cliente');
  });

  // Token huérfano sin rol (p. ej. tras un logout a medias) no puede dejar la
  // app en un estado donde AdminRoute no sabe qué hacer.
  it('no restaura email ni rol ausentes', () => {
    localStorage.setItem(TOKEN_KEY, 'jwt-abc');
    montar();
    expect(txt('email')).toBe('-');
    expect(txt('rol')).toBe('-');
  });
});

describe('AuthProvider — login', () => {
  it('persiste token, email y rol', async () => {
    const user = userEvent.setup();
    loginAPI.mockResolvedValue({ token: 'jwt-abc', email: 'ana@x.com', rol: 'cliente' });
    montar();

    await user.click(screen.getByRole('button', { name: 'Entrar' }));

    await waitFor(() => expect(txt('auth')).toBe('true'));
    expect(localStorage.getItem(TOKEN_KEY)).toBe('jwt-abc');
    expect(localStorage.getItem(EMAIL_KEY)).toBe('ana@x.com');
    expect(localStorage.getItem(ROL_KEY)).toBe('cliente');
    expect(txt('rol')).toBe('cliente');
  });

  it('limpia el error de un intento fallido anterior', async () => {
    const user = userEvent.setup();
    loginAPI.mockRejectedValueOnce(new Error('credenciales incorrectas'));
    montar();

    await user.click(screen.getByRole('button', { name: 'Entrar' }));
    await waitFor(() => expect(txt('error')).toContain('credenciales incorrectas'));

    loginAPI.mockResolvedValueOnce({ token: 't', email: 'a@b.com', rol: 'admin' });
    await user.click(screen.getByRole('button', { name: 'Entrar' }));

    await waitFor(() => expect(txt('error')).toBe('-'));
  });

  it('expone el error del servidor y no deja sesión iniciada', async () => {
    const user = userEvent.setup();
    loginAPI.mockRejectedValue(new Error('credenciales incorrectas'));
    montar();

    await user.click(screen.getByRole('button', { name: 'Entrar' }));

    await waitFor(() => expect(txt('error')).toContain('credenciales incorrectas'));
    expect(txt('auth')).toBe('false');
    expect(localStorage.getItem(TOKEN_KEY)).toBeNull();
  });

  // El `finally` tiene que apagar el loading siempre; si no, el botón queda
  // bloqueado para siempre tras un error.
  it('termina el loading tanto si éxito como si fallo', async () => {
    const user = userEvent.setup();
    loginAPI.mockRejectedValue(new Error('boom'));
    montar();

    await user.click(screen.getByRole('button', { name: 'Entrar' }));
    await waitFor(() => expect(txt('loading')).toBe('false'));
  });
});

describe('AuthProvider — logout', () => {
  it('borra las tres claves y vuelve al estado inicial', async () => {
    const user = userEvent.setup();
    loginAPI.mockResolvedValue({ token: 'jwt-abc', email: 'ana@x.com', rol: 'admin' });
    montar();

    await user.click(screen.getByRole('button', { name: 'Entrar' }));
    await waitFor(() => expect(txt('auth')).toBe('true'));

    await user.click(screen.getByRole('button', { name: 'Salir' }));

    expect(txt('auth')).toBe('false');
    expect(txt('email')).toBe('-');
    expect(txt('rol')).toBe('-');
    expect(localStorage.getItem(TOKEN_KEY)).toBeNull();
    expect(localStorage.getItem(EMAIL_KEY)).toBeNull();
    expect(localStorage.getItem(ROL_KEY)).toBeNull();
  });
});

describe('useAuth', () => {
  it('lanza si se usa fuera del provider', () => {
    const Silencio = () => {
      useAuth();
      return null;
    };

    expect(() => render(<Silencio />)).toThrow(
      /useAuth debe ser usado dentro de un AuthProvider/,
    );
  });
});