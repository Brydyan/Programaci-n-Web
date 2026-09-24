import { useCart } from '../context/CartContext';
import { useNavigate } from 'react-router-dom';

export default function Carrito() {
  const navigate = useNavigate();
  const { cart, removeFromCart, updateQuantity, totalPrice, totalItems, clearCart } = useCart();

  return (
    <div className="min-h-screen bg-gray-50 py-12 px-6">
      <div className="max-w-6xl mx-auto">
        <h1 className="text-4xl font-bold text-gray-900 mb-2">Tu Carrito de Compras</h1>
        <p className="text-gray-600 mb-8">{totalItems} artículos en el carrito</p>

        {cart.length === 0 ? (
          <div className="bg-white p-12 rounded-xl border border-gray-200 text-center">
            <div className="text-6xl mb-4">🛒</div>
            <p className="text-gray-600 text-lg mb-6">Tu carrito está vacío</p>
            <button
              onClick={() => navigate('/catalogo')}
              className="px-6 py-3 bg-rose-500 text-white rounded-lg hover:bg-rose-600 transition font-semibold"
            >
              Continuar comprando
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
            {/* Lista de productos */}
            <div className="lg:col-span-2">
              <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
                {cart.map((item, index) => (
                  <div
                    key={item.id}
                    className={`p-6 flex gap-6 ${index > 0 ? 'border-t border-gray-200' : ''}`}
                  >
                    {/* Imagen */}
                    <div className="w-24 h-24 flex-shrink-0 rounded-lg overflow-hidden bg-gray-100">
                      <img
                        src={item.img}
                        alt={item.nombre}
                        className="w-full h-full object-cover cursor-pointer hover:scale-105 transition"
                        onClick={() => navigate(`/producto/${item.id}`)}
                      />
                    </div>

                    {/* Información */}
                    <div className="flex-grow">
                      <h3
                        className="text-lg font-bold text-gray-900 cursor-pointer hover:text-rose-500 transition"
                        onClick={() => navigate(`/producto/${item.id}`)}
                      >
                        {item.nombre}
                      </h3>
                      <p className="text-sm text-gray-600">{item.categoria}</p>
                      <p className="text-sm text-gray-600 mt-1">
                        Precio unitario: ${item.precio.toFixed(2)}
                      </p>
                    </div>

                    {/* Cantidad y precio */}
                    <div className="flex flex-col items-end gap-4">
                      {/* Controles de cantidad */}
                      <div className="flex items-center gap-3 border border-gray-200 rounded-lg p-1">
                        <button
                          onClick={() => updateQuantity(item.id, item.cantidad - 1)}
                          className="w-8 h-8 flex items-center justify-center text-gray-600 hover:bg-gray-100 transition"
                        >
                          −
                        </button>
                        <span className="w-8 text-center font-semibold">{item.cantidad}</span>
                        <button
                          onClick={() => updateQuantity(item.id, item.cantidad + 1)}
                          className="w-8 h-8 flex items-center justify-center text-gray-600 hover:bg-gray-100 transition"
                        >
                          +
                        </button>
                      </div>

                      {/* Total del producto */}
                      <div className="text-right">
                        <p className="text-2xl font-bold text-gray-900">
                          ${(item.precio * item.cantidad).toFixed(2)}
                        </p>
                      </div>

                      {/* Botón eliminar */}
                      <button
                        onClick={() => removeFromCart(item.id)}
                        className="text-red-500 hover:text-red-700 transition text-sm font-semibold"
                      >
                        Eliminar
                      </button>
                    </div>
                  </div>
                ))}
              </div>

              {/* Botón limpiar carrito */}
              <button
                onClick={clearCart}
                className="mt-4 text-red-500 hover:text-red-700 transition font-semibold text-sm"
              >
                Vaciar carrito
              </button>
            </div>

            {/* Resumen de pago */}
            <div className="lg:col-span-1">
              <div className="bg-white border border-gray-200 rounded-xl p-6 sticky top-6">
                <h2 className="text-2xl font-bold text-gray-900 mb-6">Resumen del Pedido</h2>

                <div className="space-y-4 mb-6 pb-6 border-b border-gray-200">
                  <div className="flex justify-between">
                    <span className="text-gray-600">Subtotal</span>
                    <span className="font-semibold text-gray-900">
                      ${totalPrice.toFixed(2)}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-600">Envío</span>
                    <span className="font-semibold text-gray-900">Gratis</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-600">Descuento</span>
                    <span className="font-semibold text-green-600">$0.00</span>
                  </div>
                </div>

                <div className="mb-6">
                  <div className="flex justify-between items-center">
                    <span className="text-gray-900 font-bold">Total</span>
                    <span className="text-3xl font-bold text-gray-900">
                      ${totalPrice.toFixed(2)}
                    </span>
                  </div>
                </div>

                <button
                  onClick={() => navigate('/checkout')}
                  className="w-full px-6 py-3 bg-rose-500 text-white rounded-lg hover:bg-rose-600 transition font-bold mb-3"
                >
                  Proceder al Pago
                </button>

                <button
                  onClick={() => navigate('/catalogo')}
                  className="w-full px-6 py-3 border-2 border-gray-200 text-gray-900 rounded-lg hover:bg-gray-50 transition font-semibold"
                >
                  Continuar comprando
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}