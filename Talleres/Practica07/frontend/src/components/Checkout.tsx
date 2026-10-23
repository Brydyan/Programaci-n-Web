import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useCart } from '../context/cart';
import { checkoutAPI } from '../services/api';

// Campos de envío que el backend exige (openapi.yaml → CheckoutRequest:
// required [direccion, ciudad, telefono, metodo_pago, items]).
//
// Los `minLength` del contrato están replicados aquí con `minLength` para que
// el navegador bloquee el envío antes de gastar un round-trip. NO son la
// validación: el servidor vuelve a validar con go-playground, porque un
// `minLength` en el cliente es una cortesía, no una garantía.
interface EnvioForm {
  nombre: string;
  direccion: string;
  ciudad: string;
  telefono: string;
  metodo_pago: string;
}

const VALIDACION = {
  direccion: { minLength: 5, maxLength: 300 },
  ciudad: { minLength: 2, maxLength: 120 },
  telefono: { minLength: 6, maxLength: 30 },
} as const;

const Checkout = () => {
  const { cart, totalPrice } = useCart();
  const navigate = useNavigate();

  const [form, setForm] = useState<EnvioForm>({
    nombre: '',
    direccion: '',
    ciudad: '',
    telefono: '',
    metodo_pago: 'tarjeta',
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
    setForm({ ...form, [e.target.name]: e.target.value });
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      const pedido = await checkoutAPI({
        ...form,
        items: cart.map((item) => ({
          id: item.id,
          nombre: item.nombre,
          precio: item.precio,
          cantidad: item.cantidad,
        })),
      });
      navigate('/confirmacion', { state: { pedido } });
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Error desconocido');
    } finally {
      setLoading(false);
    }
  };

  if (cart.length === 0) {
    return (
      <div className="text-center py-16">
        <p className="text-slate-500 text-lg">Tu carrito está vacío.</p>
      </div>
    );
  }

  return (
    <div className="max-w-2xl mx-auto">
      <h1 className="text-2xl font-bold text-slate-800 mb-6">Proceder al Pago</h1>

      {error && (
        <div className="bg-red-50 border border-red-300 text-red-700 px-4 py-3 rounded-lg mb-6">
          <strong>Error:</strong> {error}
        </div>
      )}

      <form onSubmit={handleSubmit} className="bg-white p-6 rounded-lg border border-slate-200 shadow-sm space-y-4">
        <div>
          <label
            htmlFor="nombre" className="block text-sm font-medium text-slate-700 mb-1">
            Nombre completo <span className="text-slate-400 font-normal">(opcional)</span>
          </label>
          <input
            type="text"
            id="nombre"
            name="nombre"
            value={form.nombre}
            onChange={handleChange}
            maxLength={120}
            className="w-full border border-slate-300 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-marca-500"
          />
        </div>
        <div>
          <label
            htmlFor="direccion" className="block text-sm font-medium text-slate-700 mb-1">
            Dirección de entrega
          </label>
          <input
            type="text"
            id="direccion"
            name="direccion"
            value={form.direccion}
            onChange={handleChange}
            required
            {...VALIDACION.direccion}
            className="w-full border border-slate-300 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-marca-500"
          />
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label
            htmlFor="ciudad" className="block text-sm font-medium text-slate-700 mb-1">Ciudad</label>
            <input
              type="text"
              id="ciudad"
            name="ciudad"
              value={form.ciudad}
              onChange={handleChange}
              required
              {...VALIDACION.ciudad}
              className="w-full border border-slate-300 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-marca-500"
            />
          </div>
          <div>
            <label
            htmlFor="telefono" className="block text-sm font-medium text-slate-700 mb-1">Teléfono</label>
            <input
              type="tel"
              id="telefono"
            name="telefono"
              value={form.telefono}
              onChange={handleChange}
              required
              {...VALIDACION.telefono}
              className="w-full border border-slate-300 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-marca-500"
            />
          </div>
        </div>
        <div>
          <label
            htmlFor="metodo_pago" className="block text-sm font-medium text-slate-700 mb-1">Método de pago</label>
          <select
            id="metodo_pago"
            name="metodo_pago"
            value={form.metodo_pago}
            onChange={handleChange}
            required
            className="w-full border border-slate-300 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-marca-500"
          >
            <option value="tarjeta">Tarjeta de crédito/débito</option>
            <option value="efectivo">Efectivo</option>
            <option value="transferencia">Transferencia bancaria</option>
          </select>
        </div>

        <div className="border-t border-slate-100 pt-4 mt-4">
          <div className="flex justify-between mb-2">
            <span className="text-slate-600">{cart.reduce((t, i) => t + i.cantidad, 0)} productos</span>
            <span className="font-semibold">${totalPrice.toFixed(2)}</span>
          </div>
          <button
            type="submit"
            disabled={loading}
            className="w-full bg-marca-600 text-white py-3 rounded-lg font-semibold hover:bg-marca-700 transition disabled:opacity-50 min-h-[44px]"
          >
            {loading ? 'Procesando...' : 'Confirmar Pedido'}
          </button>
        </div>
      </form>
    </div>
  );
};

export default Checkout;
