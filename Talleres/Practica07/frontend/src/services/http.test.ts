import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ApiError } from '../types/api';
import { apiFetch, getToken, setToken } from './http';

// `http.ts` es la única puerta por la que salen todas las peticiones: aquí se
// decide si va el token, cómo se traduce un 401 a excepción y qué se hace con
// un 204 sin cuerpo. Un fallo aquí no rompe una pantalla: rompe la sesión
// entera, y de forma silenciosa.

const BASE = 'http://localhost:3000';

/** Sustituye fetch global y devuelve el mock para inspeccionar llamadas. */
function mockFetch(impl: (url: string, init?: RequestInit) => unknown) {
  const spy = vi.fn(async (url: string, init?: RequestInit) => impl(url, init));
  vi.stubGlobal('fetch', spy);
  return spy;
}

const okJson = (data: unknown, status = 200) => ({
  ok: status >= 200 && status < 300,
  status,
  json: async () => data,
});

beforeEach(() => {
  localStorage.clear();
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('getToken / setToken', () => {
  it('guarda y lee el token', () => {
    expect(getToken()).toBeNull();
    setToken('jwt-abc');
    expect(getToken()).toBe('jwt-abc');
    expect(localStorage.getItem('token')).toBe('jwt-abc');
  });

  it('setToken(null) borra la sesión', () => {
    setToken('jwt-abc');
    setToken(null);
    expect(getToken()).toBeNull();
    expect(localStorage.getItem('token')).toBeNull();
  });
});

describe('apiFetch — cabeceras', () => {
  it('añade Authorization cuando hay token', async () => {
    setToken('jwt-abc');
    const spy = mockFetch(() => okJson({ ok: true }));

    await apiFetch('/api/productos');

    const init = spy.mock.calls[0]![1] as RequestInit;
    const headers = init.headers as Headers;
    expect(headers.get('Authorization')).toBe('Bearer jwt-abc');
  });

  it('no añade Authorization si no hay token', async () => {
    const spy = mockFetch(() => okJson({ ok: true }));

    await apiFetch('/api/productos');

    const init = spy.mock.calls[0]![1] as RequestInit;
    expect((init.headers as Headers).get('Authorization')).toBeNull();
  });

  it('respeta un Authorization explícito sobre el del token', async () => {
    setToken('jwt-abc');
    const spy = mockFetch(() => okJson({ ok: true }));

    await apiFetch('/api/x', { headers: { Authorization: 'Bearer otro' } });

    const init = spy.mock.calls[0]![1] as RequestInit;
    expect((init.headers as Headers).get('Authorization')).toBe('Bearer otro');
  });

  // Sin esto, un POST sin body llega sin Content-Type y el servidor rechaza
  // el JSON antes de mirarlo.
  it('pone Content-Type en peticiones con body', async () => {
    const spy = mockFetch(() => okJson({ ok: true }));

    await apiFetch('/api/x', { method: 'POST', body: { a: 1 } });

    const init = spy.mock.calls[0]![1] as RequestInit;
    expect((init.headers as Headers).get('Content-Type')).toBe('application/json');
  });

  it('resuelve la ruta contra el backend, no contra el origen', async () => {
    const spy = mockFetch(() => okJson({}));

    await apiFetch('/api/health');

    expect(spy.mock.calls[0]![0]).toBe(`${BASE}/api/health`);
  });

  it('no prefixea una URL absoluta', async () => {
    const spy = mockFetch(() => okJson({}));

    await apiFetch('https://otro.example/api/x');

    expect(spy.mock.calls[0]![0]).toBe('https://otro.example/api/x');
  });
});

describe('apiFetch — cuerpo', () => {
  it('serializa el body a JSON', async () => {
    const spy = mockFetch(() => okJson({}));

    await apiFetch('/api/x', { method: 'POST', body: { nombre: 'Ana' } });

    const init = spy.mock.calls[0]![1] as RequestInit;
    expect(init.body).toBe('{"nombre":"Ana"}');
  });

  // Si se pre-serializara con JSON.stringify antes de llamar, http.ts lo
  // volvería a serializar y llegaría un string donde el backend espera un
  // objeto. Por eso el contrato es "pasa el objeto".
  it('el body es el objeto, no un string pre-serializado', async () => {
    const spy = mockFetch(() => okJson({}));

    await apiFetch('/api/x', {
      method: 'POST',
      body: { direccion: 'Av. 10', ciudad: 'Quito' },
    });

    const init = spy.mock.calls[0]![1] as RequestInit;
    const recibido = JSON.parse(init.body as string);
    expect(recibido).toEqual({ direccion: 'Av. 10', ciudad: 'Quito' });
    expect(typeof init.body).toBe('string');
  });

  it('no toca un FormData ni le pone Content-Type', async () => {
    const fd = new FormData();
    fd.append('campo', 'valor');
    const spy = mockFetch(() => okJson({}));

    await apiFetch('/api/x', { method: 'POST', body: fd });

    const init = spy.mock.calls[0]![1] as RequestInit;
    expect(init.body).toBe(fd);
    expect((init.headers as Headers).get('Content-Type')).toBeNull();
  });

  it('no manda body si no se pasó', async () => {
    const spy = mockFetch(() => okJson({}));

    await apiFetch('/api/x');

    const init = spy.mock.calls[0]![1] as RequestInit;
    expect(init.body).toBeUndefined();
  });
});

describe('apiFetch — respuestas', () => {
  it('deserializa el cuerpo en 200', async () => {
    mockFetch(() => okJson({ id: 7 }));

    await expect(apiFetch('/api/x')).resolves.toEqual({ id: 7 });
  });

  // Un 204 no tiene cuerpo; llamar a res.json() lanza. Es el caso real de
  // PATCH /api/admin/pedidos/:id/estado.
  it('devuelve undefined en 204 sin intentar parsear', async () => {
    mockFetch(() => ({
      ok: true,
      status: 204,
      json: async () => {
        throw new Error('no debería llamar a json()');
      },
    }));

    await expect(apiFetch('/api/x', { method: 'PATCH' })).resolves.toBeUndefined();
  });

  it('lanza ApiError con status y code del backend', async () => {
    mockFetch(() =>
      okJson({ status: 401, code: 'UNAUTHORIZED', message: 'Token requerido' }, 401),
    );

    await expect(apiFetch('/api/x')).rejects.toBeInstanceOf(ApiError);

    mockFetch(() =>
      okJson({ status: 401, code: 'UNAUTHORIZED', message: 'Token requerido' }, 401),
    );
    try {
      await apiFetch('/api/x');
      expect.unreachable('debería haber lanzado');
    } catch (err) {
      const apiErr = err as ApiError;
      expect(apiErr.status).toBe(401);
      expect(apiErr.code).toBe('UNAUTHORIZED');
      expect(apiErr.message).toBe('Token requerido');
    }
  });

  // Un 502 de un proxy devuelve HTML, no JSON. Sin este camino, el catch de
  // arriba no cubre y el error sube como "Unexpected token < in JSON".
  it('fabrica un ApiError genérico si el cuerpo no es JSON', async () => {
    mockFetch(() => ({
      ok: false,
      status: 502,
      json: async () => {
        throw new SyntaxError('Unexpected token <');
      },
    }));

    try {
      await apiFetch('/api/x');
      expect.unreachable('debería haber lanzado');
    } catch (err) {
      const apiErr = err as ApiError;
      expect(apiErr).toBeInstanceOf(ApiError);
      expect(apiErr.status).toBe(502);
      expect(apiErr.code).toBe('INTERNAL');
      expect(apiErr.message).toBe('Error 502');
    }
  });

  it('propaga un fallo de red sin envolver', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => {
        throw new TypeError('Failed to fetch');
      }),
    );

    await expect(apiFetch('/api/x')).rejects.toBeInstanceOf(TypeError);
  });
});