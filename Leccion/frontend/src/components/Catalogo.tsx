import { useCart } from '../context/CartContext';
import { useState, useMemo, useEffect } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { categorias, type Producto } from '../data/productos';
import { getProductos } from '../services/productosService';

export default function Catalogo() {
  const { addToCart } = useCart();
  const [searchParams, setSearchParams] = useSearchParams();
  const [busqueda, setBusqueda] = useState<string>("");
  const [categoria, setCategoria] = useState<string>(searchParams.get("categoria") ?? "Todas");
  const [productos, setProductos] = useState<Producto[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Cargar productos al montar
  useEffect(() => {
    const fetchProductos = async () => {
      try {
        setIsLoading(true);
        setError(null);
        const data = await getProductos();
        setProductos(data);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Error al cargar productos');
      } finally {
        setIsLoading(false);
      }
    };

    fetchProductos();
  }, []);

  // Filtrado con useMemo: solo se recalcula cuando cambian busqueda/categoria/productos
  const productosFiltrados = useMemo(() => {
    const texto = busqueda.trim().toLowerCase();
    return productos.filter((prod) => {
      const coincideCategoria = categoria === "Todas" || prod.categoria === categoria;
      const coincideTexto =
        texto === "" ||
        prod.nombre.toLowerCase().includes(texto) ||
        prod.descripcion.toLowerCase().includes(texto);
      return coincideCategoria && coincideTexto;
    });
  }, [productos, busqueda, categoria]);

  const handleCategoriaChange = (value: string) => {
    setCategoria(value);
    // Sincronizamos el query string para que la URL sea compartible
    if (value === "Todas") {
      setSearchParams({});
    } else {
      setSearchParams({ categoria: value });
    }
  };

  if (isLoading) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <p className="text-gray-600 text-lg">Cargando productos...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center px-6">
        <div className="bg-red-50 border border-red-200 rounded-xl p-6 max-w-md text-center">
          <p className="text-red-700 font-semibold mb-4">Error al cargar productos</p>
          <p className="text-red-600 text-sm mb-4">{error}</p>
          <button
            onClick={() => window.location.reload()}
            className="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition"
          >
            Reintentar
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <div className="max-w-7xl mx-auto px-6 py-12">
        <h1 className="text-2xl font-bold text-slate-800 mb-2">Catálogo de Productos</h1>
        <p className="text-slate-500 mb-6">
          {productosFiltrados.length} producto(s) encontrado(s)
        </p>

        {/* ===== BARRA DE BÚSQUEDA Y FILTROS ===== */}
        <div className="flex flex-col md:flex-row gap-4 mb-8">
          <input
            type="text"
            value={busqueda}
            onChange={(e) => setBusqueda(e.target.value)}
            placeholder="Buscar por nombre o descripción..."
            className="flex-1 px-4 py-3 rounded-lg border border-slate-300 focus:ring-2 focus:ring-indigo-600 focus:border-indigo-600 outline-none transition"
          />
          <select
            value={categoria}
            onChange={(e) => handleCategoriaChange(e.target.value)}
            className="px-4 py-3 rounded-lg border border-slate-300 bg-white focus:ring-2 focus:ring-indigo-600 focus:border-indigo-600 outline-none transition"
          >
            <option value="Todas">Todas las categorías</option>
            {categorias.map((cat) => (
              <option key={cat} value={cat}>
                {cat}
              </option>
            ))}
          </select>
        </div>

        {/* Grid de productos */}
        {productosFiltrados.length > 0 ? (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
            {productosFiltrados.map((prod) => (
              <Link
                key={prod.id}
                to={`/producto/${prod.id}`}
                className="bg-white rounded-xl overflow-hidden border border-slate-200 shadow-sm hover:shadow-lg hover:-translate-y-1 transition-all group"
              >
                <img
                  src={prod.img}
                  alt={prod.nombre}
                  className="w-full h-48 object-cover group-hover:scale-105 transition-transform"
                />
                <div className="p-4">
                  <p className="text-xs uppercase tracking-wide text-indigo-500 font-semibold">
                    {prod.categoria}
                  </p>
                  <h3 className="font-semibold text-slate-800 mt-1">{prod.nombre}</h3>
                  <p className="text-indigo-600 font-bold mt-2">${prod.precio.toFixed(2)}</p>
                  <button
                    onClick={(e) => {
                      e.preventDefault();
                      addToCart(prod);
                    }}
                    className="mt-3 w-full px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition text-sm font-semibold"
                  >
                    Añadir al Carrito
                  </button>
                </div>
              </Link>
            ))}
          </div>
        ) : (
          <div className="text-center py-12">
            <p className="text-slate-500 mb-4">No encontramos productos que coincidan con tu búsqueda.</p>
            <button
              onClick={() => {
                setBusqueda('');
                handleCategoriaChange('Todas');
              }}
              className="px-6 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition font-semibold"
            >
              Limpiar filtros
            </button>
          </div>
        )}
      </div>
    </div>
  );
}