import { useEffect, useState } from 'react';
import { getRedAPI, type Referido } from '../services/api';

const MiRed = () => {
  const [referidos, setReferidos] = useState<Referido[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchRed = async () => {
    try {
      setIsLoading(true);
      setError(null);
      const data = await getRedAPI();
      setReferidos(data);
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : 'Error al conectar con el servidor para obtener la red de referidos'
      );
      // Fallback a arreglo vacío para prevenir bloqueos en UI
      setReferidos([]);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchRed();
  }, []);

  if (isLoading) {
    return (
      <div className="flex justify-center items-center min-h-96">
        <p className="text-slate-600 font-medium">Cargando red de referidos...</p>
      </div>
    );
  }

  return (
    <div>
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between mb-6 gap-4">
        <h1 className="text-2xl font-bold text-slate-800">Mi Red de Referidos</h1>
        {error && (
          <button
            onClick={fetchRed}
            className="text-xs bg-indigo-50 text-indigo-600 hover:bg-indigo-100 px-3 py-1.5 rounded-md font-medium transition self-start"
          >
            Reintentar conexión
          </button>
        )}
      </div>

      {error && (
        <div className="bg-red-50 text-red-600 p-4 rounded-lg border border-red-200 mb-6 flex items-center justify-between">
          <p className="text-sm font-medium">{error}</p>
        </div>
      )}

      {referidos.length === 0 && !error ? (
        <div className="bg-white rounded-lg border border-slate-200 p-8 text-center text-slate-500">
          No hay miembros registrados en tu red actualmente.
        </div>
      ) : (
        <>
          {/* Vista Desktop - Tabla */}
          <div className="hidden md:block bg-white rounded-lg border border-slate-200 shadow-sm overflow-hidden">
            <table className="w-full text-left">
              <thead className="bg-slate-50 border-b border-slate-200">
                <tr>
                  <th className="p-4 font-semibold text-slate-600">Nombre</th>
                  <th className="p-4 font-semibold text-slate-600">Jerarquía</th>
                  <th className="p-4 font-semibold text-slate-600">Ventas Mensuales</th>
                </tr>
              </thead>
              <tbody>
                {referidos.map((ref) => (
                  <tr
                    key={ref.id}
                    className="border-b border-slate-100 hover:bg-slate-50 transition"
                  >
                    <td className="p-4 text-slate-700 font-medium">{ref.nombre}</td>
                    <td className="p-4 text-slate-500">{ref.nivel}</td>
                    <td className="p-4 text-indigo-600 font-semibold">{ref.ventas}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* Vista Móvil - Cards */}
          <div className="md:hidden grid grid-cols-1 gap-4">
            {referidos.map((ref) => (
              <div
                key={ref.id}
                className="bg-white rounded-lg border border-slate-200 p-4 shadow-sm"
              >
                <div className="mb-3 pb-3 border-b border-slate-100">
                  <h3 className="text-lg font-semibold text-slate-800">{ref.nombre}</h3>
                </div>
                <div className="space-y-2">
                  <div className="flex justify-between items-center">
                    <span className="text-sm text-slate-500">Jerarquía:</span>
                    <span className="text-sm font-medium text-slate-700">{ref.nivel}</span>
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-sm text-slate-500">Ventas Mensuales:</span>
                    <span className="text-sm font-semibold text-indigo-600">
                      {ref.ventas}
                    </span>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </>
      )}
    </div>
  );
};

export default MiRed;