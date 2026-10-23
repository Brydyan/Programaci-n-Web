import { createContext, useContext } from 'react';

/**
 * Tipos, objeto de contexto y hook del carrito.
 *
 * Vive separado de `CartContext.tsx` (que sólo exporta el Provider) para que
 * el Fast Refresh de Vite funcione: un archivo que exporta componentes y hooks
 * obliga a recargar la página entera en cada edición.
 *
 * @see CartContext.tsx
 */

export interface Producto {
  id: number;
  nombre: string;
  precio: number;
  img: string;
}

export interface CartItem extends Producto {
  cantidad: number;
}

export interface CartContextType {
  cart: CartItem[];
  addToCart: (producto: Producto) => void;
  removeFromCart: (id: number) => void;
  clearCart: () => void;
  totalItems: number;
  totalPrice: number;
}

export const CartContext = createContext<CartContextType | undefined>(undefined);

/** Acceso al carrito. Lanza si se usa fuera del `CartProvider`. */
export const useCart = (): CartContextType => {
  const context = useContext(CartContext);
  if (!context) {
    throw new Error('useCart debe ser usado dentro de un CartProvider');
  }
  return context;
};