import { useNavigate } from 'react-router-dom';
import { productosMock, categorias } from '../data/productos';
import { useCart } from '../context/CartContext';

export default function Tienda() {
  const navigate = useNavigate();
  const { addToCart } = useCart();

  // Destacados: primeros 4 productos
  const destacados = productosMock.slice(0, 4);

  return (
    <div>
      {/* Hero full-screen */}
      <section className="h-screen w-full bg-gradient-to-br from-rose-100 via-pink-50 to-purple-100 flex items-center justify-center relative overflow-hidden">
        <div className="absolute inset-0 opacity-20">
          <div className="absolute top-10 left-10 w-64 h-64 bg-rose-300 rounded-full blur-3xl"></div>
          <div className="absolute bottom-10 right-10 w-80 h-80 bg-purple-300 rounded-full blur-3xl"></div>
        </div>

        <div className="relative z-10 text-center px-6 max-w-2xl">
          <h1 className="text-5xl md:text-6xl font-bold text-gray-900 mb-6">
            Cuida tu piel con <span className="text-rose-500">excelencia</span>
          </h1>
          <p className="text-lg md:text-xl text-gray-700 mb-8">
            Productos premium de skincare. Fórmulas que transforman. Resultados que hablan.
          </p>
          <div className="flex gap-4 justify-center flex-wrap">
            <button
              onClick={() => navigate('/catalogo')}
              className="px-8 py-3 bg-rose-500 text-white rounded-lg font-semibold hover:bg-rose-600 transition"
            >
              Explorar Catálogo
            </button>
            <button
              onClick={() => document.getElementById('destacados')?.scrollIntoView({ behavior: 'smooth' })}
              className="px-8 py-3 bg-white text-rose-500 border-2 border-rose-500 rounded-lg font-semibold hover:bg-rose-50 transition"
            >
              Ver Destacados
            </button>
          </div>
        </div>
      </section>

      {/* Sección Destacados */}
      <section id="destacados" className="py-16 px-6 bg-white">
        <div className="max-w-6xl mx-auto">
          <div className="text-center mb-12">
            <h2 className="text-4xl font-bold text-gray-900 mb-4">Nuestros Destacados</h2>
            <p className="text-gray-600 text-lg">Productos estrella seleccionados para ti</p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
            {destacados.map((producto) => (
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
                <div className="p-4">
                  <p className="text-sm text-rose-500 font-semibold uppercase mb-1">
                    {producto.categoria}
                  </p>
                  <h3
                    className="text-lg font-bold text-gray-900 mb-2 cursor-pointer hover:text-rose-500 transition line-clamp-2"
                    onClick={() => navigate(`/producto/${producto.id}`)}
                  >
                    {producto.nombre}
                  </h3>
                  <p className="text-gray-600 text-sm line-clamp-2 mb-4">
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

          <div className="text-center mt-12">
            <button
              onClick={() => navigate('/catalogo')}
              className="px-8 py-3 bg-gray-900 text-white rounded-lg font-semibold hover:bg-gray-800 transition"
            >
              Ver todos los productos
            </button>
          </div>
        </div>
      </section>

      {/* Sección Categorías */}
      <section className="py-16 px-6 bg-gray-50">
        <div className="max-w-6xl mx-auto">
          <div className="text-center mb-12">
            <h2 className="text-4xl font-bold text-gray-900 mb-4">Explora por Categoría</h2>
            <p className="text-gray-600 text-lg">Encuentra exactamente lo que buscas</p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-4">
            {categorias.map((categoria) => {
              const count = productosMock.filter((p) => p.categoria === categoria).length;
              return (
                <button
                  key={categoria}
                  onClick={() => navigate('/catalogo')}
                  className="p-6 bg-white border border-gray-200 rounded-xl hover:border-rose-500 hover:shadow-md transition text-center group"
                >
                  <div className="text-3xl mb-3 text-rose-500 group-hover:scale-110 transition">
                    📦
                  </div>
                  <h3 className="text-lg font-bold text-gray-900 mb-1">{categoria}</h3>
                  <p className="text-sm text-gray-600">{count} productos</p>
                </button>
              );
            })}
          </div>
        </div>
      </section>

      {/* Sección Info */}
      <section className="py-16 px-6 bg-white">
        <div className="max-w-6xl mx-auto">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
            <div className="text-center">
              <div className="text-4xl mb-4">✨</div>
              <h3 className="text-xl font-bold text-gray-900 mb-2">Calidad Premium</h3>
              <p className="text-gray-600">
                Ingredientes puros y fórmulas probadas dermatológicamente
              </p>
            </div>
            <div className="text-center">
              <div className="text-4xl mb-4">🚚</div>
              <h3 className="text-xl font-bold text-gray-900 mb-2">Envío Rápido</h3>
              <p className="text-gray-600">
                Entrega en 24-48 horas a cualquier parte del país
              </p>
            </div>
            <div className="text-center">
              <div className="text-4xl mb-4">💚</div>
              <h3 className="text-xl font-bold text-gray-900 mb-2">Garantía</h3>
              <p className="text-gray-600">
                Satisfacción 100% o devolvemos tu dinero
              </p>
            </div>
          </div>
        </div>
      </section>
    </div>
  );
}
