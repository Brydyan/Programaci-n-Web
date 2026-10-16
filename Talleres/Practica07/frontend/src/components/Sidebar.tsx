import { Link } from "react-router-dom";
import { MdOutlineDashboard } from "react-icons/md";
import { GrCatalog } from "react-icons/gr";
import { FaNetworkWired } from "react-icons/fa";
import { useSidebar } from "../context/SidebarContext";
import { useAuth } from "../context/AuthContext";

type Item = {
    to: string;
    label: string;
    Icon: React.ComponentType<{ className?: string }>;
    roles: Array<"admin" | "cliente">;
};

const items: Item[] = [
    { to: "/",           label: "Dashboard", Icon: MdOutlineDashboard, roles: ["admin"] },
    { to: "/catalogo",   label: "Catálogo",  Icon: GrCatalog,          roles: ["admin", "cliente"] },
    { to: "/mi-red",     label: "Mi Red",    Icon: FaNetworkWired,     roles: ["admin"] },
];

const Sidebar = () => {
    const { isCollapsed } = useSidebar();
    const { rol } = useAuth();

    const visible = items.filter((it) => !rol || it.roles.includes(rol));

    return (
        <aside
            className={`bg-slate-500 text-white flex flex-col transition-all duration-300 h-screen ${
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
                        className="flex items-center gap-2 md:gap-3 p-2 md:p-3 rounded hover:bg-slate-800 transition text-sm md:text-base"
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