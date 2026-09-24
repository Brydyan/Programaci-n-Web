import { useState } from 'react';

export default function Configuracion() {
  const [settings, setSettings] = useState({
    nombreTienda: 'MultiCatálogo',
    email: 'admin@upse.edu.ec',
    telefono: '+593 9 9999-9999',
    direccion: 'Av. Principal 123, Guayaquil',
    comisionLevel1: 10,
    comisionLevel2: 5,
    comisionLevel3: 2,
    notificacionesEmail: true,
    notificacionesSMS: false,
    mantenimiento: false,
  });

  const [guardado, setGuardado] = useState(false);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
    const { name, value, type } = e.target;
    setSettings({
      ...settings,
      [name]: type === 'checkbox' ? (e.target as HTMLInputElement).checked : value,
    });
  };

  const handleGuardar = () => {
    setGuardado(true);
    setTimeout(() => setGuardado(false), 2000);
  };

  return (
    <div className="min-h-screen bg-gray-50 py-12 px-6">
      <div className="max-w-4xl mx-auto">
        {/* Encabezado */}
        <div className="mb-12">
          <h1 className="text-4xl font-bold text-gray-900 mb-2">⚙️ Configuración</h1>
          <p className="text-gray-600">Administra la configuración de la tienda y comisiones</p>
        </div>

        {/* Alert guardado */}
        {guardado && (
          <div className="mb-6 p-4 bg-green-50 border border-green-200 rounded-lg text-green-700 font-semibold">
            ✓ Configuración guardada exitosamente
          </div>
        )}

        {/* Secciones */}
        <div className="space-y-8">
          {/* 1. Información General */}
          <div className="bg-white rounded-xl border border-gray-200 p-8">
            <h2 className="text-2xl font-bold text-gray-900 mb-6 flex items-center gap-2">
              <span>🏪</span> Información General
            </h2>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-semibold text-gray-700 mb-2">
                  Nombre de la Tienda
                </label>
                <input
                  type="text"
                  name="nombreTienda"
                  value={settings.nombreTienda}
                  onChange={handleChange}
                  className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-200"
                />
              </div>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-semibold text-gray-700 mb-2">
                    Email Contacto
                  </label>
                  <input
                    type="email"
                    name="email"
                    value={settings.email}
                    onChange={handleChange}
                    className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-200"
                  />
                </div>
                <div>
                  <label className="block text-sm font-semibold text-gray-700 mb-2">
                    Teléfono
                  </label>
                  <input
                    type="tel"
                    name="telefono"
                    value={settings.telefono}
                    onChange={handleChange}
                    className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-200"
                  />
                </div>
              </div>
              <div>
                <label className="block text-sm font-semibold text-gray-700 mb-2">
                  Dirección Oficial
                </label>
                <input
                  type="text"
                  name="direccion"
                  value={settings.direccion}
                  onChange={handleChange}
                  className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-200"
                />
              </div>
            </div>
          </div>

          {/* 2. Tasas de Comisión */}
          <div className="bg-white rounded-xl border border-gray-200 p-8">
            <h2 className="text-2xl font-bold text-gray-900 mb-6 flex items-center gap-2">
              <span>💰</span> Tasas de Comisión Multinivel
            </h2>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
              <div className="bg-blue-50 p-6 rounded-lg border border-blue-200">
                <label className="block text-sm font-semibold text-gray-700 mb-4">
                  Nivel 1 (Directo)
                </label>
                <div className="flex items-center gap-2">
                  <input
                    type="number"
                    name="comisionLevel1"
                    value={settings.comisionLevel1}
                    onChange={handleChange}
                    min="0"
                    max="100"
                    className="w-20 px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:border-blue-500"
                  />
                  <span className="text-xl font-bold text-gray-700">%</span>
                </div>
              </div>
              <div className="bg-purple-50 p-6 rounded-lg border border-purple-200">
                <label className="block text-sm font-semibold text-gray-700 mb-4">
                  Nivel 2 (Indirecto)
                </label>
                <div className="flex items-center gap-2">
                  <input
                    type="number"
                    name="comisionLevel2"
                    value={settings.comisionLevel2}
                    onChange={handleChange}
                    min="0"
                    max="100"
                    className="w-20 px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:border-blue-500"
                  />
                  <span className="text-xl font-bold text-gray-700">%</span>
                </div>
              </div>
              <div className="bg-green-50 p-6 rounded-lg border border-green-200">
                <label className="block text-sm font-semibold text-gray-700 mb-4">
                  Nivel 3+
                </label>
                <div className="flex items-center gap-2">
                  <input
                    type="number"
                    name="comisionLevel3"
                    value={settings.comisionLevel3}
                    onChange={handleChange}
                    min="0"
                    max="100"
                    className="w-20 px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:border-blue-500"
                  />
                  <span className="text-xl font-bold text-gray-700">%</span>
                </div>
              </div>
            </div>
          </div>

          {/* 3. Notificaciones */}
          <div className="bg-white rounded-xl border border-gray-200 p-8">
            <h2 className="text-2xl font-bold text-gray-900 mb-6 flex items-center gap-2">
              <span>🔔</span> Notificaciones
            </h2>
            <div className="space-y-4">
              <label className="flex items-center gap-3 cursor-pointer">
                <input
                  type="checkbox"
                  name="notificacionesEmail"
                  checked={settings.notificacionesEmail}
                  onChange={handleChange}
                  className="w-5 h-5 rounded border-gray-300"
                />
                <span className="text-gray-700 font-medium">Recibir notificaciones por Email</span>
              </label>
              <label className="flex items-center gap-3 cursor-pointer">
                <input
                  type="checkbox"
                  name="notificacionesSMS"
                  checked={settings.notificacionesSMS}
                  onChange={handleChange}
                  className="w-5 h-5 rounded border-gray-300"
                />
                <span className="text-gray-700 font-medium">Recibir notificaciones por SMS</span>
              </label>
            </div>
          </div>

          {/* 4. Estado de Tienda */}
          <div className="bg-white rounded-xl border border-gray-200 p-8">
            <h2 className="text-2xl font-bold text-gray-900 mb-6 flex items-center gap-2">
              <span>🔧</span> Estado de Tienda
            </h2>
            <label className="flex items-center gap-3 cursor-pointer p-4 bg-yellow-50 border border-yellow-200 rounded-lg">
              <input
                type="checkbox"
                name="mantenimiento"
                checked={settings.mantenimiento}
                onChange={handleChange}
                className="w-5 h-5 rounded border-gray-300"
              />
              <span className="text-gray-700 font-medium">
                Modo Mantenimiento (usuarios verán mensaje)
              </span>
            </label>
          </div>

          {/* Botón guardar */}
          <div className="flex gap-4">
            <button
              onClick={handleGuardar}
              className="flex-1 px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition font-bold text-lg"
            >
              💾 Guardar Configuración
            </button>
            <button className="px-6 py-3 border-2 border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition font-bold">
              ↺ Restaurar Valores Predeterminados
            </button>
          </div>
        </div>

        {/* Info */}
        <div className="mt-12 p-6 bg-blue-50 border border-blue-200 rounded-xl">
          <h3 className="font-bold text-blue-900 mb-2">ℹ️ Información</h3>
          <p className="text-blue-800 text-sm">
            Los cambios en tasas de comisión se aplicarán a nuevas transacciones.
            Los usuarios actuales mantendrán sus tasas hasta la próxima revisión trimestral.
          </p>
        </div>
      </div>
    </div>
  );
}
