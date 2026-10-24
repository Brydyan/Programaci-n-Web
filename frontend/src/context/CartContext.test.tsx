import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import { CartProvider } from './CartContext';
import { useCart, type Producto } from './cart';

// El carrito es el estado con más superficie de error de toda la app: vive en
// contexto, se muta desde varios sitios y de él depende el total que se cobra.
// Un refactor aquí puede cambiar el total sin que nada se entere, y eso es
// dinero.

const SHAMPOO: Producto = {
  id: 1,
  nombre: 'Shampoo',
  precio: 15.5,
  img: '/shampoo.jpg',
};

const CREMA: Producto = {
  id: 2,
  nombre: 'Crema',
  precio: 40,
  img: '/crema.jpg',
};

/**
 * Harness mínimo: expone el contexto en un DOM legible para que el test
 * Assert sobre lo que el usuario ve, no sobre variables internas.
 */
function MontarCarrito() {
  const {
    cart,
    addToCart,
    removeFromCart,
    clearCart,
    totalItems,
    totalPrice,
  } = useCart();

  return (
    <div>
      <button onClick={() => addToCart(SHAMPOO)}>Añadir Shampoo</button>
      <button onClick={() => addToCart(CREMA)}>Añadir Crema</button>
      <button onClick={() => removeFromCart(SHAMPOO.id)}>Quitar Shampoo</button>
      <button onClick={clearCart}>Vaciar</button>
      <span data-testid="ids">{cart.map((i) => i.id).join(',')}</span>
      <span data-testid="unidades">{totalItems}</span>
      <span data-testid="precio">{totalPrice}</span>
    </div>
  );
}

const ids = () => screen.getByTestId('ids').textContent;
const unidades = () => screen.getByTestId('unidades').textContent;
const precio = () => Number(screen.getByTestId('precio').textContent);

const montar = () => render(<CartProvider><MontarCarrito /></CartProvider>);

describe('CartProvider', () => {
  it('empieza vacío', () => {
    montar();
    expect(ids()).toBe('');
    expect(unidades()).toBe('0');
    expect(precio()).toBe(0);
  });

  it('añade un producto con cantidad 1', async () => {
    const user = userEvent.setup();
    montar();

    await user.click(screen.getByRole('button', { name: 'Añadir Shampoo' }));

    expect(ids()).toBe('1');
    expect(unidades()).toBe('1');
    expect(precio()).toBe(15.5);
  });

  // El caso que distingue una implementación correcta de una que sólo añade:
  // dos clics sobre el mismo producto deben sumar, no duplicar la fila.
  it('incrementa la cantidad si el producto ya está en el carrito', async () => {
    const user = userEvent.setup();
    montar();

    await user.click(screen.getByRole('button', { name: 'Añadir Shampoo' }));
    await user.click(screen.getByRole('button', { name: 'Añadir Shampoo' }));

    expect(ids()).toBe('1');
    expect(unidades()).toBe('2');
    expect(precio()).toBe(31);
  });

  it('mantiene productos distintos en filas separadas', async () => {
    const user = userEvent.setup();
    montar();

    await user.click(screen.getByRole('button', { name: 'Añadir Shampoo' }));
    await user.click(screen.getByRole('button', { name: 'Añadir Crema' }));

    expect(ids()).toBe('1,2');
    expect(unidades()).toBe('2');
    expect(precio()).toBe(55.5);
  });

  it('el total multiplica precio por cantidad de cada línea', async () => {
    const user = userEvent.setup();
    montar();

    await user.click(screen.getByRole('button', { name: 'Añadir Crema' }));
    await user.click(screen.getByRole('button', { name: 'Añadir Crema' }));
    await user.click(screen.getByRole('button', { name: 'Añadir Crema' }));
    await user.click(screen.getByRole('button', { name: 'Añadir Shampoo' }));

    expect(unidades()).toBe('4');
    expect(precio()).toBe(40 * 3 + 15.5);
  });

  it('elimina sólo la fila indicada', async () => {
    const user = userEvent.setup();
    montar();

    await user.click(screen.getByRole('button', { name: 'Añadir Shampoo' }));
    await user.click(screen.getByRole('button', { name: 'Añadir Crema' }));
    await user.click(screen.getByRole('button', { name: 'Quitar Shampoo' }));

    expect(ids()).toBe('2');
    expect(precio()).toBe(40);
  });

  it('quitar un producto ausente no rompe el carrito', async () => {
    const user = userEvent.setup();
    montar();

    await user.click(screen.getByRole('button', { name: 'Añadir Crema' }));
    await user.click(screen.getByRole('button', { name: 'Quitar Shampoo' }));

    expect(ids()).toBe('2');
  });

  it('vaciar deja el carrito a cero', async () => {
    const user = userEvent.setup();
    montar();

    await user.click(screen.getByRole('button', { name: 'Añadir Shampoo' }));
    await user.click(screen.getByRole('button', { name: 'Añadir Crema' }));
    await user.click(screen.getByRole('button', { name: 'Vaciar' }));

    expect(ids()).toBe('');
    expect(unidades()).toBe('0');
    expect(precio()).toBe(0);
  });

  // Añadir después de vaciar tiene que volver a crear la fila, no fallar
  // por seguir operando sobre un array congelado del render anterior.
  it('permite añadir de nuevo tras vaciar', async () => {
    const user = userEvent.setup();
    montar();

    await user.click(screen.getByRole('button', { name: 'Añadir Shampoo' }));
    await user.click(screen.getByRole('button', { name: 'Vaciar' }));
    await user.click(screen.getByRole('button', { name: 'Añadir Shampoo' }));

    expect(ids()).toBe('1');
    expect(precio()).toBe(15.5);
  });
});

describe('useCart', () => {
  // Fallar con un mensaje claro al developer es mejor que devolver undefined
  // y reventar tres componentes más abajo con "cannot read property".
  it('lanza si se usa fuera del provider', () => {
    const Silencio = () => {
      useCart();
      return null;
    };

    expect(() => render(<Silencio />)).toThrow(
      /useCart debe ser usado dentro de un CartProvider/,
    );
  });
});