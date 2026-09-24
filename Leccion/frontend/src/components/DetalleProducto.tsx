import { useParams, useNavigate } from 'react-router-dom';
import { useState, useEffect } from 'react';
import { type Producto } from '../data/productos';
import { getProductoById } from '../services/productosService';
import { useCart } from '../context/CartContext';

export default function DetalleProducto() {
  const { id } = useParams();
  const navigate = useNavigate();
  const { addToCart } = useCart();
  const [selectedImageIndex, setSelectedImageIndex] = useState(0);
  const [lightboxOpen, setLightboxOpen] = useState(false);
  const [lightboxIndex, setLightboxIndex] = useState(0);
  const [producto, setProducto] = useState<Producto | undefined>(undefined);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Cargar producto al montar
  useEffect(() => {
    const fetchProducto = async () => {
      try {
        setIsLoading(true);
        setError(null);
        const data = await getProductoById(Number(id));
        setProducto(data);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Error al cargar producto');
      } finally {
        setIsLoading(false);
      }
    };

    if (id) {
      fetchProducto();
    }
  }, [id]);

  if (isLoading) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <p className="text-gray-600 text-lg">Cargando producto...</p>
      </div>
    );
  }

  if (error || !producto) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <h1 className="text-3xl font-bold text-gray-900 mb-4">
            {error ? 'Error al cargar' : 'Producto no encontrado'}
          </h1>
          {error && <p className="text-red-600 mb-4">{error}</p>}
          <button
            onClick={() => navigate('/catalogo')}
            className="px-6 py-2 bg-rose-500 text-white rounded-lg hover:bg-rose-600 transition"
          >
            Volver al catálogo
          </button>
        </div>
      </div>
    );
  }

  const allImages = [producto.img, ...producto.galeria];
  const currentImage = allImages[selectedImageIndex];

  return (
    <div className="min-h-screen bg-gray-50 py-12 px-6">
      <div className="max-w-6xl mx-auto">
        {/* Botón atrás */}
        <button
          onClick={() => navigate('/catalogo')}
          className="mb-8 px-4 py-2 text-gray-600 hover:text-gray-900 font-semibold transition flex items-center gap-2"
        >
          ← Volver al catálogo
        </button>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-8 bg-white rounded-xl p-8 shadow-sm">
          {/* Galería */}
          <div>
            {/* Imagen principal */}
            <div
              className="mb-4 h-96 bg-gray-100 rounded-lg overflow-hidden cursor-zoom-in flex items-center justify-center"
              onClick={() => {
                setLightboxOpen(true);
                setLightboxIndex(selectedImageIndex);
              }}
            >
              <img
                src={currentImage}
                alt={producto.nombre}
                className="w-full h-full object-cover hover:scale-105 transition"
              />
            </div>

            {/* Thumbnails */}
            <div className="flex gap-2">
              {allImages.map((img, index) => (
                <button
                  key={index}
                  onClick={() => setSelectedImageIndex(index)}
                  className={`w-16 h-16 rounded-lg overflow-hidden border-2 transition ${
                    selectedImageIndex === index
                      ? 'border-rose-500'
                      : 'border-gray-200 hover:border-gray-300'
                  }`}
                >
                  <img
                    src={img}
                    alt={`Miniatura ${index + 1}`}
                    className="w-full h-full object-cover"
                  />
                </button>
              ))}
            </div>
          </div>

          {/* Información */}
          <div className="flex flex-col">
            <p className="text-sm text-rose-500 font-semibold uppercase mb-2">
              {producto.categoria}
            </p>

            <h1 className="text-4xl font-bold text-gray-900 mb-4">{producto.nombre}</h1>

            <div className="mb-6">
              <span className="text-4xl font-bold text-gray-900">
                ${producto.precio.toFixed(2)}
              </span>
            </div>

            <p className="text-gray-700 text-lg leading-relaxed mb-8">
              {producto.descripcion}
            </p>

            {/* Características */}
            <div className="mb-8 p-4 bg-gray-50 rounded-lg">
              <h3 className="font-bold text-gray-900 mb-3">Características</h3>
              <ul className="space-y-2 text-gray-700">
                <li>✓ Fórmula dermatológicamente probada</li>
                <li>✓ Ingredientes naturales de alta concentración</li>
                <li>✓ Apto para todo tipo de piel</li>
                <li>✓ Resultados visibles en 14 días</li>
              </ul>
            </div>

            {/* Botones */}
            <div className="flex gap-4 mt-auto">
              <button
                onClick={() => addToCart(producto)}
                className="flex-1 px-6 py-3 bg-rose-500 text-white rounded-lg hover:bg-rose-600 transition font-bold text-lg"
              >
                Añadir al Carrito
              </button>
              <button
                onClick={() => navigate('/carrito')}
                className="px-6 py-3 border-2 border-rose-500 text-rose-500 rounded-lg hover:bg-rose-50 transition font-bold"
              >
                Ver Carrito
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Lightbox Modal */}
      {lightboxOpen && (
        <div className="fixed inset-0 bg-black bg-opacity-90 flex items-center justify-center z-50">
          <button
            onClick={() => setLightboxOpen(false)}
            className="absolute top-6 right-6 text-white text-3xl hover:opacity-70 transition"
          >
            ✕
          </button>

          {/* Imagen grande */}
          <img
            src={allImages[lightboxIndex]}
            alt="Lightbox"
            className="max-h-96 max-w-2xl object-contain"
          />

          {/* Controles */}
          <button
            onClick={() =>
              setLightboxIndex((lightboxIndex - 1 + allImages.length) % allImages.length)
            }
            className="absolute left-6 text-white text-3xl hover:opacity-70 transition"
          >
            ‹
          </button>
          <button
            onClick={() => setLightboxIndex((lightboxIndex + 1) % allImages.length)}
            className="absolute right-6 text-white text-3xl hover:opacity-70 transition"
          >
            ›
          </button>

          {/* Indicador */}
          <p className="absolute bottom-6 text-white text-sm">
            {lightboxIndex + 1} / {allImages.length}
          </p>
        </div>
      )}
    </div>
  );
}
