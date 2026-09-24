import type { CartItem } from './CartContext';

const getCartKey = (userEmail: string | null) => {
    return userEmail ? `cart_${userEmail}` : null;
};

export const initializeCart = () => {
    const userEmail = localStorage.getItem("userEmail");
    const cartKey = getCartKey(userEmail);
    if (cartKey) {
        const saved = localStorage.getItem(cartKey);
        if (saved) {
            try {
                return JSON.parse(saved) as CartItem[];
            } catch {
                return [];
            }
        }
    }
    return [];
};

export { getCartKey };
