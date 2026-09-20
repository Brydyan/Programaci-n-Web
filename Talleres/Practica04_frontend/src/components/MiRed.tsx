const MiRed = () => {
    const referidos = [
        { id: 1, nombre: "Ana García", nivel: "Nivel 1", ventas: "$1,200" },
        { id: 2, nombre: "Luis Poveda", nivel: "Nivel 1", ventas: "$850" },
        { id: 3, nombre: "Marta Sánchez", nivel: "Nivel 2", ventas: "$430" },
    ];
    return (
        <div>
            <h1 className="text-2xl font-bold text-slate-800 mb-6">
                Mi Red de Referidos
            </h1>

            {/* Vista Desktop - Tabla */}
            <div className="hidden md:block bg-white rounded-lg border border-slate-200 shadow-sm overflow-hidden">
                <table className="w-full text-left">
                    <thead className="bg-slate-50 border-b border-slate-200">
                        <tr>
                            <th className="p-4 font-semibold text-slate-600">Nombre</th>
                            <th className="p-4 font-semibold text-slate-600">Jerarquía</th>
                            <th className="p-4 font-semibold text-slate-600">
                                Ventas Mensuales
                            </th>
                        </tr>
                    </thead>
                    <tbody>
                        {referidos.map((ref) => (
                            <tr
                                key={ref.id}
                                className="border-b border-slate-100 hover:bg-slate-50 transition"
                            >
                                <td className="p-4 text-slate-700">{ref.nombre}</td>
                                <td className="p-4 text-slate-500">{ref.nivel}</td>
                                <td className="p-4 text-indigo-600 font-medium">
                                    {ref.ventas}
                                </td>
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
                            <h3 className="text-lg font-semibold text-slate-800">
                                {ref.nombre}
                            </h3>
                        </div>
                        <div className="space-y-2">
                            <div className="flex justify-between items-center">
                                <span className="text-sm text-slate-500">Jerarquía:</span>
                                <span className="text-sm font-medium text-slate-700">
                                    {ref.nivel}
                                </span>
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
        </div>
    );
};
export default MiRed;