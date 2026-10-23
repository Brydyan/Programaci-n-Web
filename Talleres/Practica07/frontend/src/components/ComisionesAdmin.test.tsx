import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import ComisionesAdmin from './ComisionesAdmin';

vi.mock('../services/comisionesService', () => ({
  getComisionesAdmin: vi.fn(),
}));

import { getComisionesAdmin } from '../services/comisionesService';
import type { ComisionResumen } from '../types/api';

const mockAdmin = vi.mocked(getComisionesAdmin);

const base = (over: Partial<ComisionResumen> = {}): ComisionResumen => ({
  data: [
    {
      id: 1,
      pedido_id: 10,
      referido_id: 2,
      nivel: 1,
      porcentaje: 10,
      monto: 8.5,
      estado: 'aprobada',
      creado_en: '2026-10-01T10:00:00Z',
    },
  ],
  page: 1,
  limit: 20,
  total: 1,
  totales: [{ estado: 'aprobada', total: 8.5, cantidad: 1 }],
  ...over,
});

const renderEn = () => render(<ComisionesAdmin />);

beforeEach(() => {
  vi.clearAllMocks();
  mockAdmin.mockResolvedValue(base());
});

describe('ComisionesAdmin', () => {
  it('lista el ledger con pedido, distribuidor y monto', async () => {
    renderEn();
    await waitFor(() => expect(screen.getByText('#10')).toBeInTheDocument());

    expect(screen.getByText('$8.50', { selector: 'td' })).toBeInTheDocument();
    expect(screen.getByText('10%')).toBeInTheDocument();
    expect(screen.getByText('Aprobada', { selector: 'span' })).toBeInTheDocument();
  });

  it('avisa que los totales son globales, no del filtro', async () => {
    renderEn();
    await waitFor(() => expect(screen.getByText('#10')).toBeInTheDocument());

    // Es la diferencia que más confunde al usar esta pantalla: alguien
    // puede leer $243.70 thinking que es el subtotal de lo que ve.
    expect(screen.getByText(/Totales de toda la organización/i)).toBeInTheDocument();
    expect(screen.getByText(/No cambian con el filtro/i)).toBeInTheDocument();
  });

  it('pide el endpoint sin filtro al cargar', async () => {
    renderEn();
    await waitFor(() => expect(mockAdmin).toHaveBeenCalled());
    expect(mockAdmin).toHaveBeenCalledWith({
      estado: undefined,
      referido_id: undefined,
      page: 1,
    });
  });

  it('filtra por estado y vuelve a la primera página', async () => {
    mockAdmin.mockResolvedValue(base({ total: 45, page: 3, data: [] }));
    renderEn();
    await waitFor(() => expect(mockAdmin).toHaveBeenCalled());

    // Se simula estar en la página 3 con un filtro que deja 3 filas: si no
    // se resetea, la tabla sale vacía sin explicación.
    await userEvent.click(screen.getByRole('button', { name: 'Pendiente' }));

    await waitFor(() =>
      expect(mockAdmin).toHaveBeenCalledWith(
        expect.objectContaining({ estado: 'pendiente', page: 1 }),
      ),
    );
  });

  it('filtra por distribuidor', async () => {
    renderEn();
    await waitFor(() => expect(mockAdmin).toHaveBeenCalled());

    await userEvent.type(screen.getByLabelText('Distribuidor (id)'), '10');

    await waitFor(() =>
      expect(mockAdmin).toHaveBeenCalledWith(
        expect.objectContaining({ referido_id: 10, page: 1 }),
      ),
    );
  });

  it('muestra el error de estado inválido que devuelve el backend', async () => {
    mockAdmin.mockRejectedValue(new Error("estado 'inventado' no es válido"));
    renderEn();

    await waitFor(() =>
      expect(screen.getByText(/estado 'inventado' no es válido/)).toBeInTheDocument(),
    );
  });

  it('explica cuando el filtro no devuelve nada', async () => {
    mockAdmin.mockResolvedValue(base({ data: [], total: 0, totales: [] }));
    renderEn();

    await waitFor(() =>
      expect(screen.getByText(/No hay comisiones con este filtro/i)).toBeInTheDocument(),
    );
  });

  it('deshabilita paginación cuando todo cabe en una página', async () => {
    renderEn();
    await waitFor(() => expect(screen.getByText('#10')).toBeInTheDocument());

    expect(screen.getByRole('button', { name: 'Anterior' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Siguiente' })).toBeDisabled();
    expect(screen.getByText('1–1 de 1')).toBeInTheDocument();
  });
});