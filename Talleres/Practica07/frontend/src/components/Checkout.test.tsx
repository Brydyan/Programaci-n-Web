import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import Checkout from './Checkout';
import { CartProvider } from '../context/CartContext';
import { useCart, type CartItem } from '../context/cart';
import { formatCurrency } from './reportes/format';

// Checkout es donde el usuario mete dinero y datos personales. Estos tests
// cubren el contrato que T4 endureció: direccion, ciudad y telefono son
// obligatorios (openapi.yaml → CheckoutRequest), y `email` ya NO se manda
// porque el servidor lo toma del JWT.

const checkoutAPI = vi.fn();

vi.mock('../services/api', () => ({
  checkoutAPI: (...args: unknown[]) => checkoutAPI(...args),
}));

// Checkout navega a /confirmacion. Se mockea el hook puntual en vez de montar
// un MemoryRouter completo: el test es de formulario, no de enrutado.
const navigate = vi.fn();
vi.mock('react-router-dom', () => ({
  useNavigate: () => navigate,
}));

// `addToCart` recibe un Producto (sin cantidad) y SIEMPRE mete cantidad 1
// (ver CartContext.tsx:29). Por eso el total esperado es 15.50 y no el doble.
const SHAMPOO: CartItem = {
  id: 1,
  nombre: 'Shampoo',
  precio: 15.5,
  img: '/shampoo.jpg',
  cantidad: 1,
};

const CANTIDAD_ESPERADA = 1;
const TOTAL_ESPERADO = 15.5;

/**
 * Harness: el carrito del Provider empieza vacío y no hay forma de llenarlo
 * desde fuera, así que se expone un botón que hace de "carrito ya listo".
 * Se hace por UI y no(setState en effect) para no falsear el ciclo de render
 * que se quiere observar.
 */
function Escenario({ conCarrito }: { conCarrito: boolean }) {
  const { addToCart } = useCart();
  return (
    <>
      {conCarrito && (
        <button onClick={() => addToCart(SHAMPOO)}>Preparar carrito</button>
      )}
      <Checkout />
    </>
  );
}

const montar = (conCarrito = false) =>
  render(
    <CartProvider>
      <Escenario conCarrito={conCarrito} />
    </CartProvider>,
  );

/** Monta con un producto en el carrito, como el usuario real llega desde el catálogo. */
async function montarConCarrito() {
  const user = userEvent.setup();
  render(
    <CartProvider>
      <Escenario conCarrito />
    </CartProvider>,
  );
  await user.click(screen.getByRole('button', { name: 'Preparar carrito' }));
  return user;
}

const rellenarEnvio = async (
  user: ReturnType<typeof userEvent.setup>,
  datos: Partial<Record<'direccion' | 'ciudad' | 'telefono', string>> = {},
) => {
  await user.type(
    screen.getByLabelText(/dirección de entrega/i),
    datos.direccion ?? 'Av. 10 de Agosto 100',
  );
  await user.type(screen.getByLabelText(/ciudad/i), datos.ciudad ?? 'Quito');
  await user.type(screen.getByLabelText(/teléfono/i), datos.telefono ?? '0987654321');
};

beforeEach(() => {
  checkoutAPI.mockReset();
  navigate.mockReset();
  checkoutAPI.mockResolvedValue({ id: 1, total: 31, estado: 'pendiente' });
});

describe('Checkout — carrito vacío', () => {
  it('invita a volver al catálogo en vez de pintar un formulario inútil', () => {
    montar();
    expect(screen.getByText(/tu carrito está vacío/i)).toBeInTheDocument();
    expect(screen.queryByLabelText(/dirección de entrega/i)).not.toBeInTheDocument();
  });
});

describe('Checkout — formulario', () => {
  it('pide los campos que el backend exige', async () => {
    await montarConCarrito();

    expect(screen.getByLabelText(/dirección de entrega/i)).toBeRequired();
    expect(screen.getByLabelText(/ciudad/i)).toBeRequired();
    expect(screen.getByLabelText(/teléfono/i)).toBeRequired();
    expect(screen.getByLabelText(/método de pago/i)).toBeRequired();
  });

  // El nombre es opcional en el contrato (openapi: `nombre` no está en
  // `required`), así que el formulario no debe bloquear el envío sin él.
  it('deja el nombre en blanco sin bloquear', async () => {
    await montarConCarrito();
    expect(screen.getByLabelText(/nombre completo/i)).not.toBeRequired();
  });

  // Regresión de T4: el formulario tenía un campo email que el backend ya no
  // acepta (lo descarta vía `json:"-"` del DTO). Que no exista es lo correcto.
  it('no pide email: la identidad viene del token', async () => {
    await montarConCarrito();
    expect(screen.queryByLabelText(/correo|email/i)).not.toBeInTheDocument();
  });

  it('muestra el total del carrito', async () => {
    await montarConCarrito();
    // 15.50 × 1
    expect(screen.getByText(formatCurrency(TOTAL_ESPERADO))).toBeInTheDocument();
  });

  it('envía los datos de envío y los items', async () => {
    const user = await montarConCarrito();
    await rellenarEnvio(user);
    await user.click(screen.getByRole('button', { name: /confirmar pedido/i }));

    await waitFor(() => expect(checkoutAPI).toHaveBeenCalledTimes(1));
    expect(checkoutAPI).toHaveBeenCalledWith(
      expect.objectContaining({
        nombre: '',
        direccion: 'Av. 10 de Agosto 100',
        ciudad: 'Quito',
        telefono: '0987654321',
        metodo_pago: 'tarjeta',
        items: [
          {
            id: 1,
            nombre: 'Shampoo',
            precio: 15.5,
            cantidad: CANTIDAD_ESPERADA,
          },
        ],
      }),
    );
  });

  // El email del body se ignoraba en el servidor, pero mandarlo sugiere que
  // importa. Fijar el contrato exacto evita que vuelva a colarse.
  it('no incluye email en el payload', async () => {
    const user = await montarConCarrito();
    await rellenarEnvio(user);
    await user.click(screen.getByRole('button', { name: /confirmar pedido/i }));

    await waitFor(() => expect(checkoutAPI).toHaveBeenCalled());
    const payload = checkoutAPI.mock.calls[0]![0] as Record<string, unknown>;
    expect(payload).not.toHaveProperty('email');
  });

  it('navega a /confirmacion tras el éxito', async () => {
    const user = await montarConCarrito();
    await rellenarEnvio(user);
    await user.click(screen.getByRole('button', { name: /confirmar pedido/i }));

    await waitFor(() => expect(navigate).toHaveBeenCalledWith('/confirmacion', expect.anything()));
  });

  it('muestra el error del servidor sin perder lo escrito', async () => {
    const user = await montarConCarrito();
    checkoutAPI.mockRejectedValue(new Error('stock insuficiente'));

    await rellenarEnvio(user);
    await user.click(screen.getByRole('button', { name: /confirmar pedido/i }));

    await waitFor(() => expect(screen.getByText(/stock insuficiente/i)).toBeInTheDocument());
    // El usuario no tiene que volver a teclear la dirección para reintentar.
    expect(screen.getByLabelText(/dirección de entrega/i)).toHaveValue('Av. 10 de Agosto 100');
  });

  it('bloquea el botón mientras la petición está en vuelo', async () => {
    const user = await montarConCarrito();
    let resolver: (v: unknown) => void = () => {};
    checkoutAPI.mockReturnValue(new Promise((r) => { resolver = r; }));

    await rellenarEnvio(user);
    await user.click(screen.getByRole('button', { name: /confirmar pedido/i }));

    const boton = screen.getByRole('button', { name: /procesando/i });
    expect(boton).toBeDisabled();

    resolver({ id: 1, total: TOTAL_ESPERADO, estado: 'pendiente' });
    await waitFor(() => expect(navigate).toHaveBeenCalled());
  });

  // El navegador bloquea el envío con minLength insufficient, pero sólo para
  // usuarios con JS activo y en navegadores que respetan la validación HTML.
  // El server vuelve a validar; estos tests fijan las longitudes del contrato.
  it('respeta los minLength del contrato', async () => {
    await montarConCarrito();
    expect(screen.getByLabelText(/dirección de entrega/i)).toHaveAttribute('minlength', '5');
    expect(screen.getByLabelText(/ciudad/i)).toHaveAttribute('minlength', '2');
    expect(screen.getByLabelText(/teléfono/i)).toHaveAttribute('minlength', '6');
  });
});