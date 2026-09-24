import { useState } from 'react';
import {
  redInicial,
  contarRed,
  sumarVentasRed,
  sumarComisiones,
  comisionDeReferido,
  TASA_COMISION,
  type Referido,
} from '../data/red';

export default function MiRed() {
  const [expandedReferidos, setExpandedReferidos] = useState<Set<number>>(new Set());

  const toggleExpand = (id: number) => {
    const newExpanded = new Set(expandedReferidos);
    if (newExpanded.has(id)) {
      newExpanded.delete(id);
    } else {
      newExpanded.add(id);
    }
    setExpandedReferidos(newExpanded);
  };

  // Datos derivados de la red
  const totalReferidos = contarRed(redInicial);
  const totalVentas = sumarVentasRed(redInicial);
  const totalComisiones = sumarComisiones(redInicial);

  const ReferidoNode = ({ referido }: { referido: Referido }) => {
    const hasChildren = referido.hijos && referido.hijos.length > 0;
    const isExpanded = expandedReferidos.has(referido.id);
    const comisionLocal = comisionDeReferido(referido);
    const tasaLocal = TASA_COMISION[referido.nivel] ?? 0;

    return (
      <div>
        <div
          className="bg-white border border-gray-200 rounded-lg p-4 mb-2 hover:shadow-md transition"
          onClick={() => hasChildren && toggleExpand(referido.id)}
        >
          <div className="flex items-start gap-4">
            {/* Avatar */}
            <div className="w-10 h-10 rounded-full bg-gradient-to-br from-rose-400 to-pink-500 flex items-center justify-center text-white font-bold flex-shrink-0 mt-1">
              {referido.nombre.charAt(0)}
            </div>

            {/* Información */}
            <div className="flex-grow">
              <div className="flex items-center gap-2 mb-2">
                {hasChildren && (
                  <button
                    className="text-gray-600 hover:text-gray-900 font-bold transition cursor-pointer"
                    onClick={(e) => {
                      e.stopPropagation();
                      toggleExpand(referido.id);
                    }}
                  >
                    {isExpanded ? '▼' : '▶'}
                  </button>
                )}
                <h3 className="font-bold text-gray-900">{referido.nombre}</h3>
                <span className="text-xs bg-rose-100 text-rose-700 px-2 py-1 rounded-full font-semibold">
                  Nivel {referido.nivel}
                </span>
              </div>

              {/* Métricas */}
              <div className="grid grid-cols-3 gap-4 text-sm">
                <div>
                  <span className="text-gray-600">Ventas</span>
                  <p className="font-bold text-gray-900">${referido.ventas.toFixed(2)}</p>
                </div>
                <div>
                  <span className="text-gray-600">Tasa Comisión</span>
                  <p className="font-bold text-gray-900">{(tasaLocal * 100).toFixed(0)}%</p>
                </div>
                <div>
                  <span className="text-gray-600">Tu Comisión</span>
                  <p className="font-bold text-green-600">${comisionLocal.toFixed(2)}</p>
                </div>
              </div>
            </div>

            {/* Contador de hijos */}
            {hasChildren && (
              <div className="text-right text-xs text-gray-600 bg-gray-50 px-3 py-1 rounded">
                {referido.hijos!.length} hijo{referido.hijos!.length !== 1 ? 's' : ''}
              </div>
            )}
          </div>
        </div>

        {/* Hijos expandibles */}
        {hasChildren && isExpanded && (
          <div className="ml-4 border-l-2 border-gray-200 pl-4 space-y-2">
            {referido.hijos!.map((hijo) => (
              <ReferidoNode key={hijo.id} referido={hijo} />
            ))}
          </div>
        )}
      </div>
    );
  };

  return (
    <div className="min-h-screen bg-gray-50 py-12 px-6">
      <div className="max-w-6xl mx-auto">
        {/* Encabezado */}
        <div className="mb-12">
          <h1 className="text-4xl font-bold text-gray-900 mb-2">Mi Red de Referidos</h1>
          <p className="text-gray-600">
            Estructura jerárquica multinivel con comisiones por nivel
          </p>
        </div>

        {/* KPIs */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-12">
          <div className="bg-white rounded-xl border border-gray-200 p-6">
            <p className="text-gray-600 text-sm font-semibold uppercase mb-2">Total Referidos</p>
            <h3 className="text-4xl font-bold text-gray-900">{totalReferidos}</h3>
            <p className="text-sm text-gray-600 mt-2">En toda tu red</p>
          </div>
          <div className="bg-white rounded-xl border border-gray-200 p-6">
            <p className="text-gray-600 text-sm font-semibold uppercase mb-2">Ventas Totales Red</p>
            <h3 className="text-4xl font-bold text-gray-900">${totalVentas.toFixed(2)}</h3>
            <p className="text-sm text-gray-600 mt-2">De tu red completa</p>
          </div>
          <div className="bg-white rounded-xl border border-gray-200 p-6">
            <p className="text-gray-600 text-sm font-semibold uppercase mb-2">Comisiones Totales</p>
            <h3 className="text-4xl font-bold text-green-600">${totalComisiones.toFixed(2)}</h3>
            <p className="text-sm text-gray-600 mt-2">Acumuladas esta temporada</p>
          </div>
        </div>

        {/* Árbol de referidos */}
        <div className="bg-white rounded-xl border border-gray-200 p-8 mb-8">
          <h2 className="text-2xl font-bold text-gray-900 mb-6">Estructura de tu Red (Expandible)</h2>

          {(redInicial.hijos ?? []).length > 0 ? (
            <div className="space-y-4">
              {(redInicial.hijos ?? []).map((referido) => (
                <ReferidoNode key={referido.id} referido={referido} />
              ))}
            </div>
          ) : (
            <div className="text-center py-12">
              <p className="text-gray-600 text-lg">Aún no tienes referidos directos</p>
            </div>
          )}
        </div>

        {/* Información sobre tasas */}
        <div className="bg-blue-50 border border-blue-200 rounded-xl p-6">
          <h3 className="font-bold text-blue-900 mb-4">💡 Tasas de Comisión por Nivel</h3>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="bg-white p-4 rounded-lg border border-blue-100">
              <p className="text-sm font-semibold text-gray-600 uppercase mb-2">Nivel 1 (Directo)</p>
              <p className="text-3xl font-bold text-blue-600">{(TASA_COMISION[1] * 100).toFixed(0)}%</p>
              <p className="text-xs text-gray-600 mt-2">Tus referidos directos</p>
            </div>
            <div className="bg-white p-4 rounded-lg border border-blue-100">
              <p className="text-sm font-semibold text-gray-600 uppercase mb-2">Nivel 2 (Indirecto)</p>
              <p className="text-3xl font-bold text-blue-600">{(TASA_COMISION[2] * 100).toFixed(0)}%</p>
              <p className="text-xs text-gray-600 mt-2">Referidos de tus referidos</p>
            </div>
            <div className="bg-white p-4 rounded-lg border border-blue-100">
              <p className="text-sm font-semibold text-gray-600 uppercase mb-2">Nivel 3+</p>
              <p className="text-3xl font-bold text-blue-600">{(TASA_COMISION[3] * 100).toFixed(0)}%</p>
              <p className="text-xs text-gray-600 mt-2">Estructura multinivel completa</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}