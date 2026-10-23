import { useCallback, useMemo, useState, type ReactNode } from 'react';
import { CartContext, type CartItem, type Producto } from './cart';

interface CartProviderProps {
  children: ReactNode;
}

/**
 * Proveedor del carrito. Sólo exporta el componente para no romper Fast Refresh;
 * los tipos y el hook `useCart` viven en `cart.ts`.
 *
 * Las acciones van en `useCallback` y el valor del contexto en `useMemo`: así su
 * identidad es estable entre renders y un `useEffect` que dependa de ellas no
 * entra en bucle.
 */
export const CartProvider = ({ children }: CartProviderProps) => {
  const [cart, setCart] = useState<CartItem[]>([]);

  const addToCart = useCallback((producto: Producto) => {
    setCart((prevCart) => {
      const itemExists = prevCart.find((item) => item.id === producto.id);
      if (itemExists) {
        return prevCart.map((item) =>
          item.id === producto.id
            ? { ...item, cantidad: item.cantidad + 1 }
            : item
        );
      }
      return [...prevCart, { ...producto, cantidad: 1 }];
    });
  }, []);

  const removeFromCart = useCallback((id: number) => {
    setCart((prevCart) => prevCart.filter((item) => item.id !== id));
  }, []);

  const clearCart = useCallback(() => setCart([]), []);

  const totalItems = cart.reduce((total, item) => total + item.cantidad, 0);
  const totalPrice = cart.reduce((total, item) => total + item.precio * item.cantidad, 0);

  const value = useMemo(
    () => ({ cart, addToCart, removeFromCart, clearCart, totalItems, totalPrice }),
    [cart, addToCart, removeFromCart, clearCart, totalItems, totalPrice]
  );

  return <CartContext.Provider value={value}>{children}</CartContext.Provider>;
};