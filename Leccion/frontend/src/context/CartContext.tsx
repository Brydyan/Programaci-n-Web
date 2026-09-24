import { createContext, useContext, useState, useEffect, type ReactNode } from 'react';
import type { Producto } from '../data/productos';
import { initializeCart, getCartKey } from './cartHelpers';

export interface CartItem extends Producto {
    cantidad: number;
}

interface CartContextType {
    cart: CartItem[];
    addToCart: (producto: Producto) => void;
    removeFromCart: (id: number) => void;
    updateQuantity: (id: number, cantidad: number) => void;
    clearCart: () => void;
    totalItems: number;
    totalPrice: number;
}

const CartContext = createContext<CartContextType | undefined>(undefined);

// eslint-disable-next-line react-refresh/only-export-components
export const useCart = () => {
    const context = useContext(CartContext);
    if (!context) {
        throw new Error("useCart debe ser usado dentro de un CartProvider");
    }
    return context;
};

interface CartProviderProps {
    children: ReactNode;
}

export const CartProvider = ({ children }: CartProviderProps) => {
    const [cart, setCart] = useState<CartItem[]>(() => initializeCart());
    const userEmail = localStorage.getItem("userEmail");

    // Guardar carrito en localStorage cuando cambia
    useEffect(() => {
        const cartKey = getCartKey(userEmail);
        if (cartKey) {
            localStorage.setItem(cartKey, JSON.stringify(cart));
        }
    }, [cart, userEmail]);

    const addToCart = (producto: Producto) => {
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
    };

    const removeFromCart = (id: number) => {
        setCart((prevCart) => prevCart.filter((item) => item.id !== id));
    };

    const updateQuantity = (id: number, cantidad: number) => {
        if (cantidad <= 0) {
            removeFromCart(id);
        } else {
            setCart((prevCart) =>
                prevCart.map((item) =>
                    item.id === id ? { ...item, cantidad } : item
                )
            );
        }
    };

    const clearCart = () => {
        setCart([]);
    };

    const totalItems = cart.reduce((total, item) => total + item.cantidad, 0);
    const totalPrice = cart.reduce((total, item) => total + item.precio * item.cantidad, 0);

    return (
        <CartContext.Provider
            value={{ cart, addToCart, removeFromCart, updateQuantity, clearCart, totalItems, totalPrice }}
        >
            {children}
        </CartContext.Provider>
    );
};