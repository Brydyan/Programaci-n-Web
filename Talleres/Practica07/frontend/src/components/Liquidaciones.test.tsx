import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import Liquidaciones from './Liquidaciones';

// Pagar dinero es la operación que esta pantalla habilita, así que los tests
// se centran en que NO se pueda pagar por accidente y en qué se ve cuando
// el backend rechaza.

vi.mock('../services/liquidacionesService', () => ({
  getDeuda: vi.fn(),
  listLiquidaciones: vi.fn(),
  liquidar: vi.fn(),
}));

import { getDeuda, liquidar, listLiquidaciones } from '../services/liquidacionesService';
import type { DeudaDistribuidor, Liquidacion, Pagina } from '../types/api';

const mockDeuda = vi.mocked(getDeuda);
const mockListar = vi.mocked(listLiquidaciones);
const mockLiquidar = vi.mocked(liquidar);

const DEUDA: DeudaDistribuidor[] = [
  { referido_id: 3, codigo: 'LUIS', nombre: 'Cliente Demo', total: 43.93, comisiones: 14 },
  { referido_id: 2, codigo: 'ADMIN', nombre: 'Administrador', total: 62.91, comisiones: 15, ultima_pago: '2026-09-15T10:00:00Z' },
];

const HISTORIAL: Pagina<Liquidacion> = {
  data: [
    {
      id: 7,
      referido_id: 3,
      total: 20,
      periodo: '2026-08',
      estado: 'pagada',
      creado_en: '2026-09-01T10:00:00Z',
      pagada_en: '2026-09-03T10:00:00Z',
    },
  ],
  limit: 20,
  page: 1,
  total: 1,
};

const renderEn = () => render(<Liquidaciones />);

beforeEach(() => {
  vi.clearAllMocks();
  mockDeuda.mockResolvedValue(DEUDA);
  mockListar.mockResolvedValue(HISTORIAL);
});

describe('Liquidaciones', () => {
  it('suma la deuda de todos los distribuidores', async () => {
    renderEn();
    await waitFor(() => expect(mockDeuda).toHaveBeenCalled());
    // 43.93 + 62.91
    expect(screen.getByText('$106.84')).toBeInTheDocument();
  });

  it('lista cada deudor con su código, comisiones y último pago', async () => {
    renderEn();
    await waitFor(() => expect(screen.getByText('Cliente Demo')).toBeInTheDocument());

    expect(screen.getByText('LUIS')).toBeInTheDocument();
    expect(screen.getByText('Nunca')).toBeInTheDocument();
    expect(screen.getByText('$43.93')).toBeInTheDocument();
  });

  it('no paga hasta que el admin confirma en el diálogo', async () => {
    renderEn();
    await waitFor(() => expect(screen.getByText('Cliente Demo')).toBeInTheDocument());

    await userEvent.click(screen.getByRole('button', { name: /Liquidar a Cliente Demo/ }));

    // El diálogo aparece con el monto exacto, pero aún no se ha pagado.
    // El monto aparece dos veces en el diálogo (frase y botón), así que se
    // comprueba sobre la frase, no con un getByText suelto.
    const dialogo = screen.getByRole('dialog');
    expect(within(dialogo).getByText(/Vas a pagar/)).toHaveTextContent('$43.93');
    expect(within(dialogo).getByText(/14 comisiones pendientes/i)).toBeInTheDocument();
    expect(mockLiquidar).not.toHaveBeenCalled();

    await userEvent.click(within(dialogo).getByRole('button', { name: /Pagar \$43\.93/ }));

    await waitFor(() => expect(mockLiquidar).toHaveBeenCalledWith(3));
  });

  it('cancela sin pagar cuando se cierra el diálogo', async () => {
    renderEn();
    await waitFor(() => expect(screen.getByText('Cliente Demo')).toBeInTheDocument());

    await userEvent.click(screen.getByRole('button', { name: /Liquidar a Cliente Demo/ }));
    await userEvent.click(screen.getByRole('button', { name: 'Cancelar' }));

    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    expect(mockLiquidar).not.toHaveBeenCalled();
  });

  it('cierra el diálogo con Escape sin pagar', async () => {
    renderEn();
    await waitFor(() => expect(screen.getByText('Cliente Demo')).toBeInTheDocument());

    await userEvent.click(screen.getByRole('button', { name: /Liquidar a Cliente Demo/ }));
    await userEvent.keyboard('{Escape}');

    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    expect(mockLiquidar).not.toHaveBeenCalled();
  });

  it('muestra el error 409 sin cerrar el diálogo', async () => {
    mockLiquidar.mockRejectedValue(new Error('no hay comisiones pendientes que liquidar'));
    renderEn();
    await waitFor(() => expect(screen.getByText('Cliente Demo')).toBeInTheDocument());

    await userEvent.click(screen.getByRole('button', { name: /Liquidar a Cliente Demo/ }));
    const dialogo = screen.getByRole('dialog');
    await userEvent.click(within(dialogo).getByRole('button', { name: /Pagar \$43\.93/ }));

    await waitFor(() =>
      expect(
        within(screen.getByRole('dialog')).getByText(/no hay comisiones pendientes/i),
      ).toBeInTheDocument(),
    );
  });

  it('confirma el pago y recarga la deuda', async () => {
    mockLiquidar.mockResolvedValue({
      id: 9,
      referido_id: 3,
      total: 43.93,
      periodo: '2026-10',
      estado: 'pagada',
      creado_en: '2026-10-23T12:00:00Z',
      pagada_en: '2026-10-23T12:00:00Z',
    });
    renderEn();
    await waitFor(() => expect(mockDeuda).toHaveBeenCalledTimes(1));

    await userEvent.click(screen.getByRole('button', { name: /Liquidar a Cliente Demo/ }));
    await userEvent.click(
      within(screen.getByRole('dialog')).getByRole('button', { name: /Pagar \$43\.93/ }),
    );

    await waitFor(() => expect(screen.getByRole('status')).toBeInTheDocument());
    expect(screen.getByRole('status')).toHaveTextContent(/Liquidación #9 por \$43\.93 a Cliente Demo/);
    // La deuda se vuelve a pedir para que la fila desaparezca.
    expect(mockDeuda).toHaveBeenCalledTimes(2);
  });

  it('avisa cuando no queda nada por cobrar', async () => {
    mockDeuda.mockResolvedValue([]);
    renderEn();

    await waitFor(() =>
      expect(screen.getByText(/No hay comisiones pendientes de cobrar/i)).toBeInTheDocument(),
    );
    expect(screen.queryByRole('button', { name: /Liquidar a/ })).not.toBeInTheDocument();
  });

  it('muestra el histórico de pagos', async () => {
    renderEn();
    await waitFor(() => expect(screen.getByText('2026-08')).toBeInTheDocument());
    expect(screen.getByText('$20.00')).toBeInTheDocument();
  });

  it('explica que aún no hay pagos registrados', async () => {
    mockListar.mockResolvedValue({ data: [], limit: 20, page: 1, total: 0 });
    renderEn();

    await waitFor(() =>
      expect(screen.getByText(/no se ha registrado ninguna liquidación/i)).toBeInTheDocument(),
    );
  });
});