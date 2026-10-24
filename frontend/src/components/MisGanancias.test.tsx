import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import MisGanancias from './MisGanancias';

// Los servicios se mockean por módulo: la pantalla depende de dos endpoints
// independientes y el comportamiento interesante es qué pasa cuando UNO de
// los dos falla. Con fetch global eso sería un testeo del cliente HTTP.

vi.mock('../services/api', () => ({
  getComisionesMiasAPI: vi.fn(),
  getLiquidacionesMiasAPI: vi.fn(),
}));

vi.mock('../context/auth', () => ({
  useAuth: () => ({ userEmail: 'ana@demo.ec' }),
}));

import { getComisionesMiasAPI, getLiquidacionesMiasAPI } from '../services/api';
import type { ComisionResumen, Pagina } from '../types/api';

const mockComisiones = vi.mocked(getComisionesMiasAPI);
const mockLiquidaciones = vi.mocked(getLiquidacionesMiasAPI);

const COMISIONES: ComisionResumen = {
  data: [
    {
      id: 1,
      pedido_id: 10,
      referido_id: 10,
      nivel: 1,
      porcentaje: 10,
      monto: 8.5,
      estado: 'aprobada',
      creado_en: '2026-10-01T10:00:00Z',
    },
    {
      id: 2,
      pedido_id: 11,
      referido_id: 10,
      nivel: 1,
      porcentaje: 10,
      monto: 3.25,
      estado: 'pagada',
      creado_en: '2026-10-02T10:00:00Z',
    },
    {
      id: 3,
      pedido_id: 12,
      referido_id: 10,
      nivel: 2,
      porcentaje: 5,
      monto: 1.5,
      estado: 'revertida',
      creado_en: '2026-10-03T10:00:00Z',
    },
  ],
  page: 1,
  limit: 20,
  total: 3,
  totales: [
    { estado: 'aprobada', total: 8.5, cantidad: 1 },
    { estado: 'pagada', total: 3.25, cantidad: 1 },
    { estado: 'revertida', total: 1.5, cantidad: 1 },
  ],
};

const LIQUIDACIONES: Pagina<{
  id: number;
  referido_id: number;
  total: number;
  periodo?: string;
  estado: 'pendiente' | 'pagada';
  creado_en: string;
  pagada_en?: string | null;
}> = {
  data: [
    {
      id: 5,
      referido_id: 10,
      total: 20,
      periodo: '2026-09',
      estado: 'pagada',
      creado_en: '2026-10-01T12:00:00Z',
      pagada_en: '2026-10-05T12:00:00Z',
    },
  ],
  limit: 20,
  page: 1,
  total: 1,
};

const renderEn = () =>
  render(
    <MemoryRouter>
      <MisGanancias />
    </MemoryRouter>,
  );

// Los montos se repiten a propósito en pantalla: cada comisión aparece en la
// fila de la tabla y su total vuelve a salir en la tarjeta "Por estado". Buscar
// por texto global daría "found multiple elements", así que se acota al panel:
// la tabla de comisiones o la tarjeta de resumen correspondiente.
const panelComisiones = () =>
  screen.getByText('Comisiones generadas').closest('div') as HTMLElement;
const panelLiquidaciones = () =>
  screen.getByText('Mis cobros').closest('div') as HTMLElement;

beforeEach(() => {
  vi.clearAllMocks();
  mockComisiones.mockResolvedValue(COMISIONES);
  mockLiquidaciones.mockResolvedValue(LIQUIDACIONES);
});

describe('MisGanancias', () => {
  it('separa lo generado de lo cobrado en tarjetas separadas', async () => {
    renderEn();
    await waitFor(() => expect(mockComisiones).toHaveBeenCalled());

    // Generado = aprobada + pagada = 8.50 + 3.25. La revertida de 1.50 NO
    // cuenta: el pedido se canceló y ese dinero se devolvió.
    expect(within(screen.getByTestId('tarjeta-generado')).getByText('$11.75')).toBeInTheDocument();
    // Cobrado solo cuenta liquidaciones pagadas: $20.00.
    expect(within(screen.getByTestId('tarjeta-cobrado')).getByText('$20.00')).toBeInTheDocument();
  });

  it('excluye las revertidas del total generado', async () => {
    // Fixture donde lo revertido es casi todo el ledger. Si la pantalla
    // sumara las revertidas mostraría $41.75 en vez de $20.00.
    mockComisiones.mockResolvedValue({
      data: [
        {
          id: 1,
          pedido_id: 1,
          referido_id: 10,
          nivel: 1,
          porcentaje: 10,
          monto: 20,
          estado: 'aprobada',
          creado_en: '2026-10-01T10:00:00Z',
        },
        {
          id: 2,
          pedido_id: 2,
          referido_id: 10,
          nivel: 1,
          porcentaje: 10,
          monto: 21.75,
          estado: 'revertida',
          creado_en: '2026-10-02T10:00:00Z',
        },
      ],
      page: 1,
      limit: 20,
      total: 2,
      totales: [
        { estado: 'aprobada', total: 20, cantidad: 1 },
        { estado: 'revertida', total: 21.75, cantidad: 1 },
      ],
    });
    renderEn();

    await waitFor(() => expect(mockComisiones).toHaveBeenCalled());
    expect(within(screen.getByTestId('tarjeta-generado')).getByText('$20.00')).toBeInTheDocument();
  });

  it('lista las comisiones con pedido, nivel y estado', async () => {
    renderEn();
    await waitFor(() => expect(screen.getByText('#10')).toBeInTheDocument());

    const tabla = panelComisiones();
    expect(within(tabla).getByText('#10')).toBeInTheDocument();
    expect(within(tabla).getByText('#11')).toBeInTheDocument();
    expect(within(tabla).getByText('#12')).toBeInTheDocument();
    expect(within(tabla).getByText('$8.50')).toBeInTheDocument();
    expect(within(tabla).getAllByText('Aprobada').length).toBeGreaterThan(0);
  });

  it('filtra por estado en cliente sin volver a pedir los datos', async () => {
    renderEn();
    await waitFor(() => expect(screen.getByText('#10')).toBeInTheDocument());

    await userEvent.click(screen.getByRole('button', { name: 'Pendientes' }));

    // Ninguna de las tres comisiones está en estado pendiente.
    expect(screen.queryByText('#10')).not.toBeInTheDocument();
    expect(
      screen.getByText(/No hay comisiones con este filtro/i),
    ).toBeInTheDocument();

    // El filtro no dispara peticiones: /comisiones/mias no pagina.
    expect(mockComisiones).toHaveBeenCalledTimes(1);
  });

  it('marca el filtro activo con aria-pressed', async () => {
    renderEn();
    await waitFor(() => expect(mockComisiones).toHaveBeenCalled());

    const pagadas = screen.getByRole('button', { name: 'Pagadas' });
    expect(pagadas).toHaveAttribute('aria-pressed', 'false');
    await userEvent.click(pagadas);
    expect(pagadas).toHaveAttribute('aria-pressed', 'true');
  });

  it('muestra el periodo y la fecha de pago de las liquidaciones', async () => {
    renderEn();
    await waitFor(() => expect(screen.getByText('2026-09')).toBeInTheDocument());

    const tabla = panelLiquidaciones();
    expect(within(tabla).getByText('2026-09')).toBeInTheDocument();
    expect(within(tabla).getByText('$20.00')).toBeInTheDocument();
    expect(within(tabla).getByText('Pagada')).toBeInTheDocument();
  });

  it('sigue mostrando el ledger si falla el endpoint de liquidaciones', async () => {
    mockLiquidaciones.mockRejectedValue(new Error('service unavailable'));
    renderEn();

    await waitFor(() => expect(screen.getByText('#10')).toBeInTheDocument());
    // El ledger llegó igual: son dos recursos independientes.
    expect(within(screen.getByTestId('tarjeta-generado')).getByText('$11.75')).toBeInTheDocument();
  });

  it('avisa del error cuando fallan los dos endpoints', async () => {
    mockComisiones.mockRejectedValue(new Error('red caída'));
    mockLiquidaciones.mockRejectedValue(new Error('red caída'));
    renderEn();

    await waitFor(() => expect(screen.getByText('red caída')).toBeInTheDocument());
    expect(screen.getByRole('button', { name: 'Reintentar' })).toBeInTheDocument();
  });

  it('explica que no hay cobros cuando el listado viene vacío', async () => {
    mockLiquidaciones.mockResolvedValue({
      data: [],
      limit: 20,
      page: 1,
      total: 0,
    });
    renderEn();

    await waitFor(() =>
      expect(screen.getByText(/Todavía no te han pagado comisiones/i)).toBeInTheDocument(),
    );
  });

  it('deshabilita "Anterior" en la primera página', async () => {
    mockLiquidaciones.mockResolvedValue({
      data: LIQUIDACIONES.data,
      limit: 20,
      page: 1,
      total: 40,
    });
    renderEn();

    await waitFor(() => expect(screen.getByText('1–20 de 40')).toBeInTheDocument());
    expect(screen.getByRole('button', { name: 'Anterior' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Siguiente' })).toBeEnabled();
  });

  it('cambia de página pidiendo el siguiente bloque', async () => {
    mockLiquidaciones.mockResolvedValue({
      data: LIQUIDACIONES.data,
      limit: 20,
      page: 1,
      total: 40,
    });
    renderEn();
    await waitFor(() => expect(screen.getByText('1–20 de 40')).toBeInTheDocument());

    await userEvent.click(screen.getByRole('button', { name: 'Siguiente' }));

    await waitFor(() =>
      expect(mockLiquidaciones).toHaveBeenCalledWith({ page: 2, limit: 20 }),
    );
  });
});