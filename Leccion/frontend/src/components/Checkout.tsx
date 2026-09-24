import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useCart } from '../context/CartContext';
import { useAuth } from '../context/AuthContext';

type CheckoutStep = 'resumen' | 'pago' | 'confirmacion';

export default function Checkout() {
  const navigate = useNavigate();
  const { cart, totalPrice, clearCart } = useCart();
  const { userEmail } = useAuth();
  const [step, setStep] = useState<CheckoutStep>('resumen');
  const [formData, setFormData] = useState({
    nombre: '',
    apellido: '',
    email: userEmail || '',
    telefono: '',
    direccion: '',
    ciudad: '',
    codigoPostal: '',
    numeroTarjeta: '',
    mes: '',
    anio: '',
    cvv: '',
  });
  const [orderNumber] = useState(`ORD-${Date.now()}`);

  // Volver atrás
  if (cart.length === 0 && step === 'resumen') {
    return (
      <div className="min-h-screen bg-gray-50 py-12 px-6">
        <div className="max-w-md mx-auto bg-white p-8 rounded-xl border border-gray-200 text-center">
          <p className="text-gray-600 mb-6">Tu carrito está vacío</p>
          <button
            onClick={() => navigate('/catalogo')}
            className="px-6 py-2 bg-rose-500 text-white rounded-lg hover:bg-rose-600 transition"
          >
            Volver a catalogo
          </button>
        </div>
      </div>
    );
  }

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setFormData({
      ...formData,
      [e.target.name]: e.target.value,
    });
  };

  const handleNextStep = () => {
    if (step === 'resumen') {
      setStep('pago');
    } else if (step === 'pago') {
      setStep('confirmacion');
    }
  };

  const handlePrevStep = () => {
    if (step === 'pago') {
      setStep('resumen');
    } else if (step === 'confirmacion') {
      setStep('pago');
    }
  };

  const handleFinish = () => {
    // Limpiar carrito
    clearCart();
    navigate('/');
  };

  return (
    <div className="min-h-screen bg-gray-50 py-12 px-6">
      <div className="max-w-4xl mx-auto">
        {/* Progreso */}
        <div className="mb-12">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-8">
              <div
                className={`text-center ${
                  ['resumen', 'pago', 'confirmacion'].indexOf(step) >= 0
                    ? 'text-rose-500'
                    : 'text-gray-400'
                }`}
              >
                <div className="w-10 h-10 rounded-full bg-rose-500 text-white flex items-center justify-center font-bold mx-auto mb-2">
                  1
                </div>
                <p className="text-sm font-semibold">Resumen</p>
              </div>

              <div
                className={`w-12 h-1 ${
                  ['pago', 'confirmacion'].indexOf(step) >= 0 ? 'bg-rose-500' : 'bg-gray-300'
                }`}
              ></div>

              <div
                className={`text-center ${
                  ['pago', 'confirmacion'].indexOf(step) >= 0
                    ? 'text-rose-500'
                    : 'text-gray-400'
                }`}
              >
                <div
                  className={`w-10 h-10 rounded-full flex items-center justify-center font-bold mx-auto mb-2 ${
                    ['pago', 'confirmacion'].indexOf(step) >= 0
                      ? 'bg-rose-500 text-white'
                      : 'bg-gray-300 text-gray-600'
                  }`}
                >
                  2
                </div>
                <p className="text-sm font-semibold">Pago</p>
              </div>

              <div
                className={`w-12 h-1 ${
                  step === 'confirmacion' ? 'bg-rose-500' : 'bg-gray-300'
                }`}
              ></div>

              <div
                className={`text-center ${
                  step === 'confirmacion' ? 'text-rose-500' : 'text-gray-400'
                }`}
              >
                <div
                  className={`w-10 h-10 rounded-full flex items-center justify-center font-bold mx-auto mb-2 ${
                    step === 'confirmacion'
                      ? 'bg-rose-500 text-white'
                      : 'bg-gray-300 text-gray-600'
                  }`}
                >
                  3
                </div>
                <p className="text-sm font-semibold">Confirmación</p>
              </div>
            </div>
          </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
          {/* Contenido principal */}
          <div className="lg:col-span-2 bg-white rounded-xl border border-gray-200 p-8">
            {step === 'resumen' && (
              <div>
                <h2 className="text-2xl font-bold text-gray-900 mb-6">Resumen del Pedido</h2>
                <div className="space-y-4 mb-8">
                  {cart.map((item) => (
                    <div key={item.id} className="flex justify-between items-center pb-4 border-b border-gray-200">
                      <div>
                        <h3 className="font-semibold text-gray-900">{item.nombre}</h3>
                        <p className="text-sm text-gray-600">Cantidad: {item.cantidad}</p>
                      </div>
                      <span className="font-bold text-gray-900">
                        ${(item.precio * item.cantidad).toFixed(2)}
                      </span>
                    </div>
                  ))}
                </div>
                <button
                  onClick={handleNextStep}
                  className="w-full px-6 py-3 bg-rose-500 text-white rounded-lg hover:bg-rose-600 transition font-bold"
                >
                  Continuar a Pago
                </button>
              </div>
            )}

            {step === 'pago' && (
              <div>
                <h2 className="text-2xl font-bold text-gray-900 mb-6">Información de Pago</h2>

                {/* Datos personales */}
                <div className="mb-8">
                  <h3 className="font-bold text-gray-900 mb-4">Datos Personales</h3>
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <input
                      type="text"
                      name="nombre"
                      placeholder="Nombre"
                      value={formData.nombre}
                      onChange={handleInputChange}
                      className="px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:border-rose-500"
                    />
                    <input
                      type="text"
                      name="apellido"
                      placeholder="Apellido"
                      value={formData.apellido}
                      onChange={handleInputChange}
                      className="px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:border-rose-500"
                    />
                    <input
                      type="email"
                      name="email"
                      placeholder="Email"
                      value={formData.email}
                      onChange={handleInputChange}
                      className="px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:border-rose-500"
                    />
                    <input
                      type="tel"
                      name="telefono"
                      placeholder="Teléfono"
                      value={formData.telefono}
                      onChange={handleInputChange}
                      className="px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:border-rose-500"
                    />
                  </div>
                </div>

                {/* Dirección */}
                <div className="mb-8">
                  <h3 className="font-bold text-gray-900 mb-4">Dirección de Entrega</h3>
                  <div className="grid grid-cols-1 gap-4">
                    <input
                      type="text"
                      name="direccion"
                      placeholder="Dirección completa"
                      value={formData.direccion}
                      onChange={handleInputChange}
                      className="px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:border-rose-500"
                    />
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      <input
                        type="text"
                        name="ciudad"
                        placeholder="Ciudad"
                        value={formData.ciudad}
                        onChange={handleInputChange}
                        className="px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:border-rose-500"
                      />
                      <input
                        type="text"
                        name="codigoPostal"
                        placeholder="Código Postal"
                        value={formData.codigoPostal}
                        onChange={handleInputChange}
                        className="px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:border-rose-500"
                      />
                    </div>
                  </div>
                </div>

                {/* Tarjeta */}
                <div className="mb-8">
                  <h3 className="font-bold text-gray-900 mb-4">Información de Tarjeta</h3>
                  <div className="space-y-4">
                    <input
                      type="text"
                      name="numeroTarjeta"
                      placeholder="Número de tarjeta (16 dígitos)"
                      value={formData.numeroTarjeta}
                      onChange={handleInputChange}
                      className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:border-rose-500"
                    />
                    <div className="grid grid-cols-3 gap-4">
                      <input
                        type="text"
                        name="mes"
                        placeholder="MM"
                        value={formData.mes}
                        onChange={handleInputChange}
                        className="px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:border-rose-500"
                      />
                      <input
                        type="text"
                        name="anio"
                        placeholder="AA"
                        value={formData.anio}
                        onChange={handleInputChange}
                        className="px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:border-rose-500"
                      />
                      <input
                        type="text"
                        name="cvv"
                        placeholder="CVV"
                        value={formData.cvv}
                        onChange={handleInputChange}
                        className="px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:border-rose-500"
                      />
                    </div>
                  </div>
                </div>

                <div className="flex gap-4">
                  <button
                    onClick={handlePrevStep}
                    className="flex-1 px-6 py-3 border-2 border-gray-300 text-gray-900 rounded-lg hover:bg-gray-50 transition font-bold"
                  >
                    Atrás
                  </button>
                  <button
                    onClick={handleNextStep}
                    className="flex-1 px-6 py-3 bg-rose-500 text-white rounded-lg hover:bg-rose-600 transition font-bold"
                  >
                    Confirmar Pago
                  </button>
                </div>
              </div>
            )}

            {step === 'confirmacion' && (
              <div className="text-center">
                <div className="text-6xl mb-6">✅</div>
                <h2 className="text-3xl font-bold text-gray-900 mb-4">¡Pedido Confirmado!</h2>
                <p className="text-gray-600 mb-2">Tu pedido ha sido procesado exitosamente.</p>
                <p className="text-2xl font-bold text-rose-500 mb-8">Número de orden: {orderNumber}</p>

                <div className="bg-gray-50 p-6 rounded-lg mb-8 text-left">
                  <h3 className="font-bold text-gray-900 mb-4">Resumen de tu compra:</h3>
                  <div className="space-y-2 text-sm text-gray-700">
                    <p>
                      <strong>Nombre:</strong> {formData.nombre} {formData.apellido}
                    </p>
                    <p>
                      <strong>Email:</strong> {formData.email}
                    </p>
                    <p>
                      <strong>Entrega en:</strong> {formData.direccion}, {formData.ciudad}
                    </p>
                    <p>
                      <strong>Total pagado:</strong> ${totalPrice.toFixed(2)}
                    </p>
                  </div>
                </div>

                <button
                  onClick={handleFinish}
                  className="w-full px-6 py-3 bg-rose-500 text-white rounded-lg hover:bg-rose-600 transition font-bold"
                >
                  Volver al inicio
                </button>
              </div>
            )}
          </div>

          {/* Resumen lateral */}
          <div className="bg-white rounded-xl border border-gray-200 p-6 h-fit sticky top-6">
            <h3 className="text-xl font-bold text-gray-900 mb-6">Total del Pedido</h3>
            <div className="space-y-4 pb-6 border-b border-gray-200 mb-6">
              {cart.map((item) => (
                <div key={item.id} className="flex justify-between text-sm">
                  <span className="text-gray-600">
                    {item.nombre} x {item.cantidad}
                  </span>
                  <span className="font-semibold text-gray-900">
                    ${(item.precio * item.cantidad).toFixed(2)}
                  </span>
                </div>
              ))}
            </div>
            <div className="mb-4 pb-4 border-b border-gray-200">
              <div className="flex justify-between mb-2">
                <span className="text-gray-600">Subtotal</span>
                <span className="font-semibold">${totalPrice.toFixed(2)}</span>
              </div>
              <div className="flex justify-between mb-2">
                <span className="text-gray-600">Envío</span>
                <span className="font-semibold">Gratis</span>
              </div>
            </div>
            <div className="flex justify-between text-lg">
              <span className="font-bold text-gray-900">Total</span>
              <span className="font-bold text-rose-500">${totalPrice.toFixed(2)}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
