import { useCart } from '../context/CartContext';
import { useEffect, useState } from 'react';
import { getProductosAPI } from '../services/api';

interface Producto {
    id: number;
    nombre: string;
    precio: number;
    img: string;
}

const Catalogo = () => {
    const { addToCart } = useCart();
    const [productos, setProductos] = useState<Producto[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        const fetchProductos = async () => {
            try {
                setIsLoading(true);
                setError(null);
                const data = await getProductosAPI();
                setProductos(data);
            } catch (err) {
                setError(err instanceof Error ? err.message : 'Error al cargar productos');
            } finally {
                setIsLoading(false);
            }
        };

        fetchProductos();
    }, []);
    if (isLoading) {
        return (
            <div className="flex justify-center items-center min-h-96">
                <p className="text-slate-600">Cargando productos...</p>
            </div>
        );
    }

    if (error) {
        return (
            <div className="bg-red-50 text-red-600 p-4 rounded-lg border border-red-200">
                Error: {error}
            </div>
        );
    }

    return (
        <div>
            <h1 className="text-2xl font-bold text-slate-800 mb-6">Catálogo de
                Productos</h1>
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
                {productos.map((prod) => (
                    <div key={prod.id} className="bg-white rounded-lg overflow-hidden
border border-slate-200 shadow-sm flex flex-col">
                        <img src={prod.img} alt={prod.nombre} className="w-full h-40
object-cover" />
                        <div className="p-4 flex flex-col flex-1">
                            <h3 className="font-semibold text-slate-700">{prod.nombre}</h3>
                            <p className="text-indigo-600 font-bold mt-2 mb4">${prod.precio.toFixed(2)}</p>

                            <button
                                onClick={() => addToCart(prod)}
                                className="mt-auto w-full bg-slate-900 text-white py-2 rounded
text-sm hover:bg-indigo-600 transition"
                            >
                                Añadir al Carrito
                            </button>
                        </div>
                    </div>
                ))}
            </div>
        </div>
    );
};

export default Catalogo;