import { useAuth } from '../context/AuthContext';
import { useCart } from '../context/CartContext';
import { redInicial, contarRed, sumarVentasRed, sumarComisiones, nivelAlcanzado } from '../data/red';
import { productosMock } from '../data/productos';
import { useNavigate } from 'react-router-dom';

export default function Dashboard() {
  const navigate = useNavigate();
  const { user } = useAuth();
  const { totalItems, totalPrice } = useCart();

  // Cálculos red MLM (funciones puras)
  const totalReferidos = contarRed(redInicial);
  const totalVentasRed = sumarVentasRed(redInicial);
  const totalComisiones = sumarComisiones(redInicial);
  const referidosDirectos = (redInicial.hijos ?? []).length;

  // Nivel alcanzado según referidos directos
  const nivelActual = nivelAlcanzado(referidosDirectos);

  return (
    <div className="min-h-screen bg-gray-50 py-12 px-6">
      <div className="max-w-7xl mx-auto">
        {/* Bienvenida */}
        <div className="mb-12">
          <h1 className="text-4xl font-bold text-gray-900 mb-2">
            Hola, {user?.email?.split('@')[0]}
          </h1>
          <p className="text-gray-600">
            Rol: <span className="font-semibold text-rose-500">{user?.rol === 'admin' ? 'Administrador' : 'Cliente'}</span>
          </p>
        </div>

        {/* KPIs principales */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-12">
          {/* Carrito actual */}
          <div className="bg-white rounded-xl border border-gray-200 p-6 hover:shadow-lg transition">
            <p className="text-gray-600 text-sm font-semibold uppercase mb-2">Carrito Actual</p>
            <h3 className="text-4xl font-bold text-gray-900">${totalPrice.toFixed(2)}</h3>
            <p className="text-sm text-gray-600 mt-2">{totalItems} artículos</p>
            <button
              onClick={() => navigate('/carrito')}
              className="mt-4 w-full px-4 py-2 bg-rose-500 text-white rounded-lg hover:bg-rose-600 transition text-sm font-semibold"
            >
              Ver carrito
            </button>
          </div>

          {/* Ventas Red */}
          <div className="bg-white rounded-xl border border-gray-200 p-6 hover:shadow-lg transition">
            <p className="text-gray-600 text-sm font-semibold uppercase mb-2">Ventas en Red</p>
            <h3 className="text-4xl font-bold text-blue-600">${totalVentasRed.toFixed(2)}</h3>
            <p className="text-sm text-gray-600 mt-2">De tu red completa</p>
            <button
              onClick={() => navigate('/mi-red')}
              className="mt-4 w-full px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 transition text-sm font-semibold"
            >
              Ver red
            </button>
          </div>

          {/* Comisiones */}
          <div className="bg-white rounded-xl border border-gray-200 p-6 hover:shadow-lg transition">
            <p className="text-gray-600 text-sm font-semibold uppercase mb-2">Comisiones</p>
            <h3 className="text-4xl font-bold text-green-600">${totalComisiones.toFixed(2)}</h3>
            <p className="text-sm text-gray-600 mt-2">Ganadas este período</p>
          </div>

          {/* Referidos */}
          <div className="bg-white rounded-xl border border-gray-200 p-6 hover:shadow-lg transition">
            <p className="text-gray-600 text-sm font-semibold uppercase mb-2">Referidos</p>
            <h3 className="text-4xl font-bold text-purple-600">{totalReferidos}</h3>
            <p className="text-sm text-gray-600 mt-2">En toda la red</p>
          </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
          {/* Columna izquierda: Info usuario y nivel */}
          <div className="lg:col-span-2 space-y-8">
            {/* Nivel y progreso */}
            <div className="bg-gradient-to-br from-rose-50 to-pink-50 rounded-xl border border-rose-200 p-8">
              <div className="flex items-center justify-between mb-6">
                <div>
                  <p className="text-rose-600 text-sm font-semibold uppercase mb-2">Nivel Actual</p>
                  <h3 className="text-4xl font-bold text-gray-900">{nivelActual}</h3>
                </div>
                <div className="text-6xl">👑</div>
              </div>

              <div className="mb-4">
                <div className="flex justify-between items-center mb-2">
                  <span className="text-sm font-semibold text-gray-700">Progreso hacia Diamante</span>
                  <span className="text-sm font-bold text-gray-900">
                    {referidosDirectos} / 6 referidos
                  </span>
                </div>
                <div className="w-full bg-gray-200 rounded-full h-3">
                  <div
                    className="bg-gradient-to-r from-rose-400 to-pink-500 h-3 rounded-full transition-all duration-500"
                    style={{ width: `${Math.min(100, (referidosDirectos / 6) * 100)}%` }}
                  ></div>
                </div>
              </div>

              <p className="text-sm text-gray-700">
                Necesitas {Math.max(0, 6 - referidosDirectos)} referido{Math.max(0, 6 - referidosDirectos) !== 1 ? 's' : ''} directo{Math.max(0, 6 - referidosDirectos) !== 1 ? 's' : ''} más para alcanzar Diamante
              </p>
            </div>

            {/* Opciones rápidas */}
            <div className="bg-white rounded-xl border border-gray-200 p-8">
              <h3 className="text-2xl font-bold text-gray-900 mb-6">Acciones Rápidas</h3>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <button
                  onClick={() => navigate('/tienda')}
                  className="px-6 py-3 bg-blue-500 text-white rounded-lg hover:bg-blue-600 transition font-semibold flex items-center justify-center gap-2"
                >
                  🏪 Ir a Tienda
                </button>
                <button
                  onClick={() => navigate('/catalogo')}
                  className="px-6 py-3 bg-purple-500 text-white rounded-lg hover:bg-purple-600 transition font-semibold flex items-center justify-center gap-2"
                >
                  📋 Ver Catálogo
                </button>
                <button
                  onClick={() => navigate('/mi-red')}
                  className="px-6 py-3 bg-green-500 text-white rounded-lg hover:bg-green-600 transition font-semibold flex items-center justify-center gap-2"
                >
                  🌳 Mi Red
                </button>
                <button
                  onClick={() => navigate('/carrito')}
                  className="px-6 py-3 bg-rose-500 text-white rounded-lg hover:bg-rose-600 transition font-semibold flex items-center justify-center gap-2"
                >
                  🛒 Carrito ({totalItems})
                </button>
              </div>
            </div>
          </div>

          {/* Columna derecha: Resumen stats */}
          <div className="space-y-6">
            {/* Productos disponibles */}
            <div className="bg-white rounded-xl border border-gray-200 p-6">
              <p className="text-gray-600 text-sm font-semibold uppercase mb-4">Catálogo</p>
              <h3 className="text-4xl font-bold text-gray-900 mb-2">{productosMock.length}</h3>
              <p className="text-sm text-gray-600 mb-4">Productos disponibles</p>
              <button
                onClick={() => navigate('/catalogo')}
                className="w-full px-4 py-2 bg-gray-900 text-white rounded-lg hover:bg-gray-800 transition text-sm font-semibold"
              >
                Explorar
              </button>
            </div>

            {/* Info rápida */}
            <div className="bg-blue-50 border border-blue-200 rounded-xl p-6">
              <h4 className="font-bold text-blue-900 mb-3">💡 Tips</h4>
              <ul className="text-sm text-blue-800 space-y-2">
                <li>✓ Completa tu perfil para más beneficios</li>
                <li>✓ Invita amigos y gana comisiones</li>
                <li>✓ Revisa tu red regularmente</li>
              </ul>
            </div>

            {/* Admin only */}
            {user?.rol === 'admin' && (
              <div className="bg-amber-50 border border-amber-200 rounded-xl p-6">
                <h4 className="font-bold text-amber-900 mb-3">🔐 Panel Admin</h4>
                <p className="text-sm text-amber-800 mb-4">
                  Como administrador, puedes gestionar usuarios y productos
                </p>
                <button className="w-full px-4 py-2 bg-amber-600 text-white rounded-lg hover:bg-amber-700 transition text-sm font-semibold">
                  Panel de Control
                </button>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}