import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import RedGlobal from './RedGlobal';

vi.mock('../services/redService', () => ({
  getRedGlobalAPI: vi.fn(),
  getRedMiaAPI: vi.fn(),
}));

import { getRedGlobalAPI } from '../services/redService';
import type { NodoRed } from '../types/api';

const mockRed = vi.mocked(getRedGlobalAPI);

const ARBOL: NodoRed[] = [
  {
    id: 1,
    codigo: 'TU',
    nombre: 'TU',
    nivel: 0,
    ventas: 50,
    hijos: [
      {
        id: 2,
        codigo: 'ADMIN',
        nombre: 'Administrador',
        nivel: 1,
        ventas: 30,
        hijos: [
          { id: 10, codigo: 'MKT-ANA', nombre: 'Ana', nivel: 2, ventas: 20, hijos: [] },
        ],
      },
    ],
  },
  { id: 3, codigo: 'LUIS', nombre: 'Cliente Demo', nivel: 0, ventas: 40, hijos: [] },
];

const renderEn = () => render(<RedGlobal />);

beforeEach(() => {
  vi.clearAllMocks();
  mockRed.mockResolvedValue(ARBOL);
});

describe('RedGlobal', () => {
  it('resumen los totales de toda la organización', async () => {
    renderEn();
    await waitFor(() => expect(screen.getByText('Administrador')).toBeInTheDocument());

    // 4 nodos, $140.00 de ventas, profundidad 2.
    expect(screen.getByText('4')).toBeInTheDocument();
    expect(screen.getByText('$140.00')).toBeInTheDocument();
    expect(screen.getByText('Personas')).toBeInTheDocument();
    expect(screen.getByText('Profundidad')).toBeInTheDocument();
  });

  it('dibuja todas las ramas desde las raíces', async () => {
    renderEn();
    await waitFor(() => expect(screen.getByText('LUIS')).toBeInTheDocument());

    expect(screen.getByText('Administrador')).toBeInTheDocument();
    expect(screen.getByText('Ana')).toBeInTheDocument();
    expect(screen.getByText('$20.00')).toBeInTheDocument();
  });

  it('filtra por nombre conservando la rama del que coincide', async () => {
    renderEn();
    await waitFor(() => expect(screen.getByText('LUIS')).toBeInTheDocument());

    await userEvent.type(screen.getByRole('searchbox'), 'Ana');

    // La raíz se conserva aunque la coincidencia sea una descendiente: sin
    // eso, "Ana" aparecería flotando sin contexto. Se comprueba por el
    // código porque el nodo raíz se llama "TU" y su código también "TU",
    // así que getByText('TU') daría "found multiple elements".
    await waitFor(() => expect(screen.queryByText('LUIS')).not.toBeInTheDocument());
    expect(screen.getByText('MKT-ANA')).toBeInTheDocument();
    expect(screen.getByText(/de 4 personas coinciden/)).toBeInTheDocument();
  });

  it('filtra por código', async () => {
    renderEn();
    await waitFor(() => expect(screen.getByText('LUIS')).toBeInTheDocument());

    await userEvent.type(screen.getByRole('searchbox'), 'LUIS');

    await waitFor(() => expect(screen.queryByText('TU')).not.toBeInTheDocument());
    expect(screen.getByText('Cliente Demo')).toBeInTheDocument();
  });

  it('avisa cuando nada coincide y no vuelve a pedir datos', async () => {
    renderEn();
    await waitFor(() => expect(mockRed).toHaveBeenCalledTimes(1));

    await userEvent.type(screen.getByRole('searchbox'), 'zzzz');

    expect(screen.getByText(/Ninguna persona coincide/i)).toBeInTheDocument();
    // El filtrado es de cliente: el árbol entero ya está en memoria.
    expect(mockRed).toHaveBeenCalledTimes(1);
  });

  it('explica que la red está vacía', async () => {
    mockRed.mockResolvedValue([]);
    renderEn();

    await waitFor(() =>
      expect(screen.getByText(/no hay distribuidores registrados/i)).toBeInTheDocument(),
    );
  });

  it('muestra el error y reintenta', async () => {
    mockRed.mockRejectedValueOnce(new Error('sin permiso'));
    renderEn();

    await waitFor(() => expect(screen.getByText('sin permiso')).toBeInTheDocument());
    await userEvent.click(screen.getByRole('button', { name: 'Reintentar' }));

    await waitFor(() => expect(mockRed).toHaveBeenCalledTimes(2));
    expect(screen.getByText('Administrador')).toBeInTheDocument();
  });
});