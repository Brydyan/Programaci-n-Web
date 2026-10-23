import { Link } from "react-router-dom";
import { MdInventory2, MdOutlineDashboard } from "react-icons/md";
import { GrCatalog } from "react-icons/gr";
import { FaNetworkWired, FaReceipt, FaClipboardList, FaPiggyBank, FaMoneyBillWave, FaSearchDollar, FaSitemap } from "react-icons/fa";
import { useSidebar } from "../context/sidebar";
import { useAuth } from "../context/auth";

type Item = {
    to: string;
    label: string;
    Icon: React.ComponentType<{ className?: string }>;
    roles: Array<"admin" | "cliente">;
};

const items: Item[] = [
    { to: "/",                 label: "Dashboard",  Icon: MdOutlineDashboard, roles: ["admin"] },
    { to: "/catalogo",         label: "Catálogo",   Icon: GrCatalog,          roles: ["admin", "cliente"] },
    { to: "/mi-red",           label: "Mi Red",     Icon: FaNetworkWired,     roles: ["admin", "cliente"] },
    { to: "/mis-ganancias",    label: "Mis Ganancias", Icon: FaPiggyBank,      roles: ["admin", "cliente"] },
    { to: "/admin/productos",  label: "Productos",  Icon: MdInventory2,       roles: ["admin"] },
    { to: "/mis-pedidos",      label: "Mis Pedidos", Icon: FaReceipt,          roles: ["admin", "cliente"] },
    { to: "/admin/pedidos",    label: "Pedidos",    Icon: FaClipboardList,    roles: ["admin"] },
    { to: "/admin/liquidaciones", label: "Liquidaciones", Icon: FaMoneyBillWave, roles: ["admin"] },
    { to: "/admin/comisiones", label: "Auditoría", Icon: FaSearchDollar, roles: ["admin"] },
    { to: "/admin/red", label: "Red Global", Icon: FaSitemap, roles: ["admin"] },
];

// El aside iba en `bg-slate-500`, un gris medio que no aparecía en ninguna
// otra superficie de la aplicación y dejaba el menú como un bloque
// descolgado. Ahora la superficie es `bg-slate-800` —neutra y oscura— y el
// acento se reserva para la interacción: el item se ilumina con el tono de
// marca al pasar por encima.
const SIDEBAR_TENUE = "bg-marca-700 hover:bg-marca-600";

const Sidebar = () => {
    const { isCollapsed } = useSidebar();
    const { rol } = useAuth();

    // Sin rol conocido no se muestra NADA, tampoco lo de admin. La guarda
    // anterior era `!rol || ...`, y con rol null conjunctaba todos los items:
    // el menú prometía "Pedidos" y "Productos" a un cliente que al pulsarlos
    // rebotaba en ProtectedRoute. Las rutas ya estaban protegidas, así que no
    // era un agujero de permisos, solo una promesa incumplida.
    const visible = rol ? items.filter((it) => it.roles.includes(rol)) : [];

    return (
        <aside
            className={`bg-slate-800 text-white flex flex-col transition-all duration-300 h-screen ${
                isCollapsed ? "w-20" : "w-64"
            }`}
        >
            <div
                className={`p-4 md:p-6 text-lg md:text-2xl font-bold border-b border-slate-700 flex items-center justify-center transition-all ${
                    isCollapsed ? "text-lg" : ""
                }`}
            >
                {isCollapsed ? "MC" : "MultiCatálogo"}
            </div>
            <nav className="flex-1 p-2 md:p-4 space-y-1 md:space-y-2">
                {visible.length === 0 && (
                    <div className="text-slate-200 text-sm px-2 py-3 opacity-75">
                        {isCollapsed ? "…" : "Sin menús disponibles"}
                    </div>
                )}
                {visible.map(({ to, label, Icon }) => (
                    <Link
                        key={to}
                        to={to}
                        className={`flex items-center gap-2 md:gap-3 p-2 md:p-3 rounded transition text-sm md:text-base ${SIDEBAR_TENUE}`}
                        title={isCollapsed ? label : ""}
                    >
                        <Icon className="text-lg md:text-xl flex-shrink-0" />
                        {!isCollapsed && (
                            <span className="whitespace-nowrap">{label}</span>
                        )}
                    </Link>
                ))}
            </nav>
            {!isCollapsed && (
                <div className="p-4 border-t border-slate-700 text-xs text-slate-200 opacity-70">
                    {rol === "admin" ? "Modo administrador" : "Modo cliente"}
                </div>
            )}
        </aside>
    );
};

export default Sidebar;