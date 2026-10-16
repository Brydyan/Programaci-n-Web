import { useLocation, useNavigate, Link } from 'react-router-dom';
import { useCart } from '../context/CartContext';
import { useEffect } from 'react';

interface PedidoState {
  id: number;
  total: number;
  estado: string;
}

const Confirmacion = () => {
  const location = useLocation();
  const navigate = useNavigate();
  const { clearCart } = useCart();
  const pedido = location.state?.pedido as PedidoState | undefined;

  useEffect(() => {
    if (!pedido) {
      navigate('/', { replace: true });
      return;
    }
    clearCart();
  }, []);

  if (!pedido) return null;

  return (
    <div className="max-w-lg mx-auto text-center py-16">
      <div className="bg-white p-8 rounded-lg border border-slate-200 shadow-sm">
        <div className="text-6xl mb-4">✅</div>
        <h1 className="text-2xl font-bold text-green-700 mb-2">¡Compra confirmada!</h1>
        <p className="text-slate-500 mb-6">Tu pedido ha sido procesado exitosamente.</p>
        <div className="bg-green-50 border border-green-200 rounded-lg p-4 mb-6 text-left">
          <p className="text-sm text-slate-600"><span className="font-semibold">N° de orden:</span> #{pedido.id}</p>
          <p className="text-sm text-slate-600"><span className="font-semibold">Total:</span> ${pedido.total.toFixed(2)}</p>
          <p className="text-sm text-slate-600"><span className="font-semibold">Estado:</span> {pedido.estado}</p>
        </div>
        <Link
          to="/catalogo"
          className="inline-block bg-indigo-600 text-white px-6 py-3 rounded-lg font-semibold hover:bg-indigo-700 transition"
        >
          Seguir comprando
        </Link>
      </div>
    </div>
  );
};

export default Confirmacion;
