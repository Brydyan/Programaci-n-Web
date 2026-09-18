import { Link } from "react-router-dom";
import { MdOutlineDashboard } from "react-icons/md";
import { GrCatalog } from "react-icons/gr";
import { FaNetworkWired } from "react-icons/fa";
import { useSidebar } from "../context/SidebarContext";

const Sidebar = () => {
    const { isCollapsed } = useSidebar();

    return (
        <aside
            className={`bg-slate-500 text-white flex flex-col transition-all duration-300 ${
                isCollapsed ? "w-20" : "w-64"
            }`}
        >
            <div
                className={`p-6 text-2xl font-bold border-b border-slate-700 flex items-center justify-center ${
                    isCollapsed ? "text-lg" : ""
                }`}
            >
                {isCollapsed ? "MC" : "MultiCatálogo"}
            </div>
            <nav className="flex-1 p-4 space-y-2">
                <Link
                    to="/"
                    className="flex items-center gap-3 p-3 rounded hover:bg-slate-800 transition"
                    title={isCollapsed ? "Dashboard" : ""}
                >
                    <MdOutlineDashboard className="text-xl flex-shrink-0" />
                    {!isCollapsed && <span>Dashboard</span>}
                </Link>
                <Link
                    to="/catalogo"
                    className="flex items-center gap-3 p-3 rounded hover:bg-slate-800 transition"
                    title={isCollapsed ? "Catálogo" : ""}
                >
                    <GrCatalog className="text-xl flex-shrink-0" />
                    {!isCollapsed && <span>Catálogo</span>}
                </Link>
                <Link
                    to="/mi-red"
                    className="flex items-center gap-3 p-3 rounded hover:bg-slate-800 transition"
                    title={isCollapsed ? "Mi Red" : ""}
                >
                    <FaNetworkWired className="text-xl flex-shrink-0" />
                    {!isCollapsed && <span>Mi Red</span>}
                </Link>
            </nav>
        </aside>
    );
};

export default Sidebar;