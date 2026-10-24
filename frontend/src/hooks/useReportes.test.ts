import { act, renderHook, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { useReportes } from './useReportes';

// `useReportes` tiene un guard de request-id cuya razón de existir es un fallo
// intermitente: si dos peticiones salen y la lenta llega después, la rápida
// queda pisada. Eso sólo se ve con red lenta, en producción, y el síntoma es
// "el filtro no funciona". Un test con `deferred` lo hace determinista.

/** Promesa con resolución manual, para ordenar las respuestas a voluntad. */
function diferido<T>() {
  let resolver!: (v: T) => void;
  let rechazar!: (e: unknown) => void;
  const promesa = new Promise<T>((res, rej) => {
    resolver = res;
    rechazar = rej;
  });
  return { promesa, resolver, rechazar };
}

describe('useReportes', () => {
  it('entrega los datos y apaga el loading', async () => {
    const fetcher = vi.fn().mockResolvedValue({ total: 42 });
    const { result } = renderHook(() => useReportes(fetcher));

    expect(result.current.cargando).toBe(true);
    expect(result.current.data).toBeNull();

    await waitFor(() => expect(result.current.cargando).toBe(false));
    expect(result.current.data).toEqual({ total: 42 });
    expect(result.current.error).toBeNull();
  });

  it('expone el mensaje de error sin perder el resto del estado', async () => {
    const fetcher = vi.fn().mockRejectedValue(new Error('boom'));
    const { result } = renderHook(() => useReportes(fetcher));

    await waitFor(() => expect(result.current.error).toBe('boom'));
    expect(result.current.cargando).toBe(false);
    expect(result.current.data).toBeNull();
  });

  it('recarga al llamar a reload()', async () => {
    const fetcher = vi.fn().mockResolvedValue({ n: 1 });
    const { result } = renderHook(() => useReportes(fetcher));

    await waitFor(() => expect(result.current.data).toEqual({ n: 1 }));
    expect(fetcher).toHaveBeenCalledTimes(1);

    fetcher.mockResolvedValue({ n: 2 });
    act(() => result.current.reload());

    await waitFor(() => expect(result.current.data).toEqual({ n: 2 }));
    expect(fetcher).toHaveBeenCalledTimes(2);
  });

  // El motivo del hook. Sin este test, el guard se puede borrar creyendo que
  // es un early-return inofensivo y el bug vuelve, intermitente.
  it('ignora una respuesta obsoleta que llega después de otra', async () => {
    const lenta = diferido<{ n: number }>();
    const rapida = diferido<{ n: number }>();

    const fetcher = vi
      .fn()
      .mockReturnValueOnce(lenta.promesa)
      .mockReturnValueOnce(rapida.promesa);

    const { result, rerender } = renderHook(
      ({ deps }) => useReportes(fetcher, [deps]),
      { initialProps: { deps: 1 } },
    );

    // Cambio de dependencia → sale la segunda petición.
    rerender({ deps: 2 });

    // La RÁPIDA responde primero: es la que debe ganar.
    await act(async () => {
      rapida.resolver({ n: 2 });
    });
    await waitFor(() => expect(result.current.data).toEqual({ n: 2 }));

    // La LENTA responde después. Si no hubiera guard, pisaría con { n: 1 }.
    await act(async () => {
      lenta.resolver({ n: 1 });
    });

    expect(result.current.data).toEqual({ n: 2 });
  });

  it('un error obsoleto no pisa el estado de éxito', async () => {
    const lenta = diferido<{ n: number }>();
    const rapida = diferido<{ n: number }>();

    const fetcher = vi
      .fn()
      .mockReturnValueOnce(lenta.promesa)
      .mockReturnValueOnce(rapida.promesa);

    const { result, rerender } = renderHook(
      ({ deps }) => useReportes(fetcher, [deps]),
      { initialProps: { deps: 1 } },
    );

    rerender({ deps: 2 });

    await act(async () => {
      rapida.resolver({ n: 2 });
    });
    await waitFor(() => expect(result.current.data).toEqual({ n: 2 }));

    await act(async () => {
      lenta.rechazar(new Error('fallo viejo'));
    });

    expect(result.current.error).toBeNull();
    expect(result.current.data).toEqual({ n: 2 });
  });

  it('relanza la petición al cambiar las dependencias', async () => {
    const fetcher = vi.fn().mockResolvedValue({ n: 1 });
    const { rerender } = renderHook(
      ({ deps }) => useReportes(fetcher, [deps]),
      { initialProps: { deps: 'a' } },
    );

    await waitFor(() => expect(fetcher).toHaveBeenCalledTimes(1));

    rerender({ deps: 'b' });
    await waitFor(() => expect(fetcher).toHaveBeenCalledTimes(2));

    // Mismas dependencias: no debe volver a pedir.
    rerender({ deps: 'b' });
    expect(fetcher).toHaveBeenCalledTimes(2);
  });

  // El fetcher se recrea en cada render. Si fuese dependencia del efecto, el
  // efecto se re-ejecutaría siempre y no pararía nunca.
  it('no entra en bucle aunque el fetcher cambie de identidad', async () => {
    const fetcher = vi.fn().mockResolvedValue({ n: 1 });

    const { rerender } = renderHook(() =>
      useReportes(() => fetcher(), []),
    );

    await waitFor(() => expect(fetcher).toHaveBeenCalledTimes(1));
    rerender();
    rerender();
    expect(fetcher).toHaveBeenCalledTimes(1);
  });
});