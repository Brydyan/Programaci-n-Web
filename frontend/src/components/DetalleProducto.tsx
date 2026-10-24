import { useEffect, useState } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { getProductoPorIdAPI, type Producto } from '../services/api';
import { useCart } from '../context/cart';

const DetalleProducto = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { addToCart } = useCart();

  const [producto, setProducto] = useState<Producto | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [added, setAdded] = useState(false);

  useEffect(() => {
    const fetchProducto = async () => {
      if (!id) {
        setError('El ID del producto es requerido');
        setIsLoading(false);
        return;
      }

      try {
        setIsLoading(true);
        setError(null);
        const data = await getProductoPorIdAPI(id);
        setProducto(data);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Error al cargar el producto');
      } finally {
        setIsLoading(false);
      }
    };

    fetchProducto();
  }, [id]);

  const handleAddToCart = () => {
    if (producto) {
      addToCart(producto);
      setAdded(true);
      setTimeout(() => setAdded(false), 2000);
    }
  };

  if (isLoading) {
    return (
      <div className="flex justify-center items-center min-h-96">
        <p className="text-slate-600 font-medium">Cargando producto...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="max-w-2xl mx-auto my-8">
        <div className="bg-red-50 border border-red-200 text-red-700 px-6 py-4 rounded-lg shadow-sm mb-6 flex flex-col gap-2">
          <div className="flex items-center gap-2">
            <span className="font-semibold text-lg">Error al cargar producto</span>
          </div>
          <p className="text-sm font-medium">{error}</p>
        </div>
        <button
          onClick={() => navigate('/catalogo')}
          className="inline-flex items-center px-4 py-2 bg-slate-900 text-white rounded-lg text-sm hover:bg-marca-600 transition font-medium min-h-[44px]"
>
          &larr; Volver al Catálogo
        </button>
      </div>
    );
  }

  if (!producto) {
    return null;
  }

  return (
    <div className="max-w-4xl mx-auto">
      {/* Botón Volver */}
      <div className="mb-6">
        <Link
          to="/catalogo"
          className="inline-flex items-center text-slate-600 hover:text-marca-600 transition text-sm font-medium gap-1"
        >
          &larr; Volver al Catálogo
        </Link>
      </div>

      {/* Tarjeta de Detalle del Producto */}
      <div className="bg-white rounded-lg border border-slate-200 shadow-sm overflow-hidden grid grid-cols-1 md:grid-cols-2">
        <div className="h-72 md:h-full bg-slate-100 flex items-center justify-center overflow-hidden">
          <img
            src={producto.img}
            alt={producto.nombre}
            referrerPolicy="origin"
            className="w-full h-full object-cover"
          />
        </div>

        <div className="p-6 md:p-8 flex flex-col justify-between">
          <div>
            <div className="flex items-center justify-between mb-2">
              <span className="text-xs font-semibold uppercase tracking-wider text-slate-400 bg-slate-100 px-2 py-1 rounded">
                ID #{producto.id}
              </span>
              <span className="text-xs font-medium text-emerald-600 bg-emerald-50 px-2 py-1 rounded">
                En Stock
              </span>
            </div>
            <h1 className="text-2xl md:text-3xl font-bold text-slate-800 mb-4">
              {producto.nombre}
            </h1>
            <p className="text-marca-600 text-3xl font-bold mb-6">
              ${producto.precio.toFixed(2)}
            </p>
            <div className="text-slate-600 text-sm leading-relaxed mb-6 space-y-2">
              <p>
                Producto de alta calidad dermatológica formulado para el cuidado diario y
                revitalización facial.
              </p>
              <p className="text-slate-500">
                Garantía de satisfacción y entrega inmediata a través de nuestra red de distribuidores.
              </p>
            </div>
          </div>

          <div className="pt-4 border-t border-slate-100 flex flex-col gap-3">
            <button
              onClick={handleAddToCart}
              className={`w-full py-3 px-4 min-h-[44px] rounded-lg font-medium transition text-center ${
                added
                  ? 'bg-emerald-600 text-white'
                  : 'bg-slate-900 text-white hover:bg-marca-600'
              }`}
            >
              {added ? '✓ ¡Añadido al Carrito!' : 'Añadir al Carrito'}
            </button>
            <button
              onClick={() => navigate('/catalogo')}
              className="w-full py-2.5 px-4 rounded-lg border border-slate-300 text-slate-700 text-sm hover:bg-slate-50 transition min-h-[44px]"
            >
              Seguir Comprando
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};

export default DetalleProducto;
