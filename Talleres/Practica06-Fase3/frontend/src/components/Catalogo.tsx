import { useCart } from '../context/CartContext';
import { useEffect, useState } from 'react';
import { getProductosAPI, type Producto } from '../services/api';
import { Link } from 'react-router-dom';

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
                    <div
                        key={prod.id}
                        className="bg-white rounded-lg overflow-hidden border border-slate-200 shadow-sm flex flex-col hover:shadow-md transition"
                    >
                        <Link to={`/producto/${prod.id}`} className="block overflow-hidden bg-slate-50">
                            <img
                                src={prod.img}
                                alt={prod.nombre}
                                className="w-full h-40 object-cover hover:scale-105 transition duration-300"
                            />
                        </Link>
                        <div className="p-4 flex flex-col flex-1">
                            <Link to={`/producto/${prod.id}`} className="hover:text-indigo-600 transition">
                                <h3 className="font-semibold text-slate-700">{prod.nombre}</h3>
                            </Link>
                            <p className="text-indigo-600 font-bold mt-2 mb-4">${prod.precio.toFixed(2)}</p>

                            <div className="mt-auto flex flex-col gap-2">
                                <button
                                    onClick={() => addToCart(prod)}
                                    className="w-full bg-slate-900 text-white py-2 rounded text-sm hover:bg-indigo-600 transition font-medium"
                                >
                                    Añadir al Carrito
                                </button>
                                <Link
                                    to={`/producto/${prod.id}`}
                                    className="w-full text-center border border-slate-200 text-slate-600 py-1.5 rounded text-xs hover:bg-slate-50 hover:text-indigo-600 transition font-medium"
                                >
                                    Ver Detalle
                                </Link>
                            </div>
                        </div>
                    </div>
                ))}
            </div>
        </div>
    );
};

export default Catalogo;