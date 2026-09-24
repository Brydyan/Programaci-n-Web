import { useState } from 'react';
import { redInicial, contarRed, sumarVentasRed, sumarComisiones } from '../data/red';
import { productosMock } from '../data/productos';

export default function Reportes() {
  const totalReferidos = contarRed(redInicial);
  const totalVentasRed = sumarVentasRed(redInicial);
  const totalComisiones = sumarComisiones(redInicial);

  // Mock data: reportes de ventas
  const reporteVentas = [
    { mes: 'Enero', ventas: 4200, comisiones: 210 },
    { mes: 'Febrero', ventas: 5100, comisiones: 255 },
    { mes: 'Marzo', ventas: 6800, comisiones: 340 },
    { mes: 'Abril', ventas: 7200, comisiones: 360 },
  ];

  // Mock data: top productos (useState initializer para generar valores aleatorios solo una vez)
  const [topProductos] = useState(() =>
    productosMock.slice(0, 5).map((p) => ({
      ...p,
      unidadesVendidas: Math.floor(Math.random() * 100) + 10,
      ingresos: p.precio * (Math.floor(Math.random() * 100) + 10),
    }))
  );

  return (
    <div className="min-h-screen bg-gray-50 py-12 px-6">
      <div className="max-w-7xl mx-auto">
        {/* Encabezado */}
        <div className="mb-12">
          <h1 className="text-4xl font-bold text-gray-900 mb-2">📊 Reportes Administrativos</h1>
          <p className="text-gray-600">Análisis de ventas, red multinivel y productos</p>
        </div>

        {/* KPIs generales */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-12">
          <div className="bg-white rounded-xl border border-gray-200 p-6">
            <p className="text-gray-600 text-sm font-semibold uppercase mb-2">Total Usuarios</p>
            <h3 className="text-4xl font-bold text-blue-600">{totalReferidos + 1}</h3>
            <p className="text-sm text-gray-600 mt-2">En la plataforma</p>
          </div>
          <div className="bg-white rounded-xl border border-gray-200 p-6">
            <p className="text-gray-600 text-sm font-semibold uppercase mb-2">Ventas Red</p>
            <h3 className="text-4xl font-bold text-green-600">${totalVentasRed.toFixed(2)}</h3>
            <p className="text-sm text-gray-600 mt-2">Acumuladas</p>
          </div>
          <div className="bg-white rounded-xl border border-gray-200 p-6">
            <p className="text-gray-600 text-sm font-semibold uppercase mb-2">Comisiones Pagadas</p>
            <h3 className="text-4xl font-bold text-purple-600">${totalComisiones.toFixed(2)}</h3>
            <p className="text-sm text-gray-600 mt-2">Distribuidas</p>
          </div>
          <div className="bg-white rounded-xl border border-gray-200 p-6">
            <p className="text-gray-600 text-sm font-semibold uppercase mb-2">Productos Activos</p>
            <h3 className="text-4xl font-bold text-orange-600">{productosMock.length}</h3>
            <p className="text-sm text-gray-600 mt-2">En catálogo</p>
          </div>
        </div>

        {/* Ventas por mes */}
        <div className="bg-white rounded-xl border border-gray-200 p-8 mb-8">
          <h2 className="text-2xl font-bold text-gray-900 mb-6">Ventas Mensuales</h2>
          <div className="space-y-4">
            {reporteVentas.map((reporte) => (
              <div key={reporte.mes} className="flex items-center gap-4">
                <div className="w-24 text-sm font-semibold text-gray-700">{reporte.mes}</div>
                <div className="flex-grow">
                  <div className="bg-gray-200 rounded-full h-8 flex items-center relative">
                    <div
                      className="bg-gradient-to-r from-blue-400 to-blue-600 h-8 rounded-full flex items-center justify-end pr-4"
                      style={{ width: `${(reporte.ventas / 8000) * 100}%` }}
                    >
                      <span className="text-white text-xs font-bold">
                        ${reporte.ventas}
                      </span>
                    </div>
                  </div>
                </div>
                <div className="w-24 text-right">
                  <p className="text-sm font-semibold text-gray-900">${reporte.comisiones}</p>
                  <p className="text-xs text-gray-600">comisiones</p>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Top productos */}
        <div className="bg-white rounded-xl border border-gray-200 p-8">
          <h2 className="text-2xl font-bold text-gray-900 mb-6">Top 5 Productos</h2>
          <div className="overflow-x-auto">
            <table className="w-full text-left">
              <thead className="bg-gray-50 border-b border-gray-200">
                <tr>
                  <th className="px-6 py-3 font-semibold text-gray-700">Producto</th>
                  <th className="px-6 py-3 font-semibold text-gray-700">Unidades Vendidas</th>
                  <th className="px-6 py-3 font-semibold text-gray-700">Ingresos</th>
                  <th className="px-6 py-3 font-semibold text-gray-700">Categoría</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200">
                {topProductos.map((producto) => (
                  <tr key={producto.id} className="hover:bg-gray-50 transition">
                    <td className="px-6 py-4 font-medium text-gray-900">{producto.nombre}</td>
                    <td className="px-6 py-4 text-gray-600">{producto.unidadesVendidas}</td>
                    <td className="px-6 py-4 font-semibold text-green-600">
                      ${producto.ingresos.toFixed(2)}
                    </td>
                    <td className="px-6 py-4 text-gray-600">
                      <span className="bg-blue-100 text-blue-700 px-3 py-1 rounded-full text-sm font-medium">
                        {producto.categoria}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  );
}
