import { useCart } from '../context/CartContext';
import { useState, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { productosMock, categorias } from '../data/productos';

export default function Catalogo() {
  const navigate = useNavigate();
  const { addToCart } = useCart();
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedCategory, setSelectedCategory] = useState<string | null>(null);

  // useMemo: filtra productos solo cuando searchTerm o selectedCategory cambian
  const filteredProductos = useMemo(() => {
    return productosMock.filter((producto) => {
      const matchSearch =
        producto.nombre.toLowerCase().includes(searchTerm.toLowerCase()) ||
        producto.descripcion.toLowerCase().includes(searchTerm.toLowerCase());

      const matchCategory = selectedCategory === null || producto.categoria === selectedCategory;

      return matchSearch && matchCategory;
    });
  }, [searchTerm, selectedCategory]);

  return (
    <div className="min-h-screen bg-gray-50">
      <div className="max-w-7xl mx-auto px-6 py-12">
        {/* Encabezado */}
        <div className="mb-12">
          <h1 className="text-4xl font-bold text-gray-900 mb-2">Catálogo de Productos</h1>
          <p className="text-gray-600">
            {filteredProductos.length} productos encontrados
          </p>
        </div>

        {/* Filtros */}
        <div className="grid grid-cols-1 lg:grid-cols-4 gap-6 mb-12">
          {/* Búsqueda */}
          <div className="lg:col-span-2">
            <label className="block text-sm font-semibold text-gray-700 mb-2">
              Buscar
            </label>
            <input
              type="text"
              placeholder="Serum, Crema, Mascarilla..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:border-rose-500 focus:ring-2 focus:ring-rose-200"
            />
          </div>

          {/* Categorías */}
          <div className="lg:col-span-2">
            <label className="block text-sm font-semibold text-gray-700 mb-2">
              Categoría
            </label>
            <select
              value={selectedCategory || ''}
              onChange={(e) => setSelectedCategory(e.target.value || null)}
              className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:border-rose-500 focus:ring-2 focus:ring-rose-200"
            >
              <option value="">Todas las categorías</option>
              {categorias.map((cat) => (
                <option key={cat} value={cat}>
                  {cat}
                </option>
              ))}
            </select>
          </div>
        </div>

        {/* Grid de productos */}
        {filteredProductos.length > 0 ? (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
            {filteredProductos.map((producto) => (
              <div
                key={producto.id}
                className="bg-white border border-gray-200 rounded-xl overflow-hidden hover:shadow-lg transition group"
              >
                {/* Imagen */}
                <div
                  className="h-48 bg-gray-100 overflow-hidden cursor-pointer group-hover:scale-105 transition"
                  onClick={() => navigate(`/producto/${producto.id}`)}
                >
                  <img
                    src={producto.img}
                    alt={producto.nombre}
                    className="w-full h-full object-cover"
                  />
                </div>

                {/* Contenido */}
                <div className="p-4 flex flex-col">
                  <p className="text-sm text-rose-500 font-semibold uppercase mb-1">
                    {producto.categoria}
                  </p>
                  <h3
                    className="text-lg font-bold text-gray-900 mb-2 cursor-pointer hover:text-rose-500 transition line-clamp-2"
                    onClick={() => navigate(`/producto/${producto.id}`)}
                  >
                    {producto.nombre}
                  </h3>
                  <p className="text-gray-600 text-sm line-clamp-2 mb-4 flex-grow">
                    {producto.descripcion}
                  </p>

                  <div className="flex justify-between items-center">
                    <span className="text-2xl font-bold text-gray-900">
                      ${producto.precio.toFixed(2)}
                    </span>
                    <button
                      onClick={() => addToCart(producto)}
                      className="px-4 py-2 bg-rose-500 text-white rounded-lg hover:bg-rose-600 transition text-sm font-semibold"
                    >
                      + Carrito
                    </button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="text-center py-12">
            <p className="text-gray-600 text-lg mb-4">
              No encontramos productos que coincidan con tu búsqueda.
            </p>
            <button
              onClick={() => {
                setSearchTerm('');
                setSelectedCategory(null);
              }}
              className="px-6 py-2 bg-rose-500 text-white rounded-lg hover:bg-rose-600 transition font-semibold"
            >
              Limpiar filtros
            </button>
          </div>
        )}
      </div>
    </div>
  );
}