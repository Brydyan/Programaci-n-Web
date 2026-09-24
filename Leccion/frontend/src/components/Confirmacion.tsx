import { Link, useLocation } from 'react-router-dom';

interface PedidoConfirmado {
  numero: string;
  items: { nombre: string; cantidad: number; precio: number }[];
  total: number;
  cliente: { nombre: string; email: string; ciudad: string };
  fecha: string;
}

const Confirmacion = () => {
  const location = useLocation();
  const pedido = (location.state as { pedido?: PedidoConfirmado } | null)?.pedido;

  // Si no hay datos de pedido (acceso directo a la URL), mostramos un mensaje genérico
  if (!pedido) {
    return (
      <div className="min-h-screen bg-gray-50 py-12 px-6">
        <div className="max-w-2xl mx-auto bg-white p-10 rounded-lg border border-slate-200 text-center">
          <div className="text-6xl mb-4">✅</div>
          <h1 className="text-3xl font-bold text-slate-800 mb-2">¡Gracias por tu compra!</h1>
          <p className="text-slate-500 mb-6">Tu pedido ha sido registrado correctamente.</p>
          <Link
            to="/tienda"
            className="inline-block bg-indigo-600 text-white font-semibold px-6 py-3 rounded-lg hover:bg-indigo-700 transition"
          >
            Volver a la Tienda
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50 py-12 px-6">
      <div className="max-w-2xl mx-auto">
        {/* Encabezado de confirmación */}
        <div className="bg-white p-8 rounded-lg border border-slate-200 shadow-sm text-center mb-8">
          <div className="text-6xl mb-4">✅</div>
          <h1 className="text-3xl font-bold text-slate-800 mb-2">¡Pedido Confirmado!</h1>
          <p className="text-slate-500 mb-4">Tu compra ha sido procesada exitosamente.</p>
          <p className="text-2xl font-bold text-indigo-600">Número de Orden: {pedido.numero}</p>
        </div>

        {/* Datos del cliente */}
        <div className="bg-white p-6 rounded-lg border border-slate-200 shadow-sm mb-6">
          <h2 className="text-lg font-bold text-slate-800 mb-4">Datos del Cliente</h2>
          <div className="space-y-2 text-slate-700">
            <p>
              <strong>Nombre:</strong> {pedido.cliente.nombre}
            </p>
            <p>
              <strong>Email:</strong> {pedido.cliente.email}
            </p>
            <p>
              <strong>Ciudad:</strong> {pedido.cliente.ciudad}
            </p>
            <p>
              <strong>Fecha del pedido:</strong> {pedido.fecha}
            </p>
          </div>
        </div>

        {/* Resumen de productos */}
        <div className="bg-white p-6 rounded-lg border border-slate-200 shadow-sm mb-6">
          <h2 className="text-lg font-bold text-slate-800 mb-4">Resumen de Compra</h2>
          <div className="space-y-3">
            {pedido.items.map((item, index) => (
              <div key={index} className="flex justify-between pb-3 border-b border-slate-100 last:border-0">
                <div>
                  <p className="font-semibold text-slate-800">{item.nombre}</p>
                  <p className="text-sm text-slate-500">Cantidad: {item.cantidad}</p>
                </div>
                <span className="font-semibold text-slate-800">
                  ${(item.precio * item.cantidad).toFixed(2)}
                </span>
              </div>
            ))}
          </div>

          {/* Total */}
          <div className="mt-6 pt-4 border-t border-slate-200">
            <div className="flex justify-between items-center">
              <span className="text-lg font-bold text-slate-800">Total Pagado</span>
              <span className="text-2xl font-bold text-indigo-600">
                ${pedido.total.toFixed(2)}
              </span>
            </div>
          </div>
        </div>

        {/* Botones de acción */}
        <div className="flex gap-4">
          <Link
            to="/tienda"
            className="flex-1 text-center bg-indigo-600 text-white font-semibold py-3 rounded-lg hover:bg-indigo-700 transition"
          >
            Volver a la Tienda
          </Link>
          <Link
            to="/"
            className="flex-1 text-center border-2 border-indigo-600 text-indigo-600 font-semibold py-3 rounded-lg hover:bg-indigo-50 transition"
          >
            Ir al Dashboard
          </Link>
        </div>
      </div>
    </div>
  );
};

export default Confirmacion;
