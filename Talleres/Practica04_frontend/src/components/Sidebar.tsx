import { Link } from "react-router-dom";
import { MdOutlineDashboard } from "react-icons/md";
import { GrCatalog } from "react-icons/gr";
import { FaNetworkWired } from "react-icons/fa";
import { useSidebar } from "../context/SidebarContext";

const Sidebar = () => {
    const { isCollapsed } = useSidebar();

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
                <Link
                    to="/"
                    className="flex items-center gap-2 md:gap-3 p-2 md:p-3 rounded hover:bg-slate-800 transition text-sm md:text-base"
                    title={isCollapsed ? "Dashboard" : ""}
                >
                    <MdOutlineDashboard className="text-lg md:text-xl flex-shrink-0" />
                    {!isCollapsed && <span className="whitespace-nowrap">Dashboard</span>}
                </Link>
                <Link
                    to="/catalogo"
                    className="flex items-center gap-2 md:gap-3 p-2 md:p-3 rounded hover:bg-slate-800 transition text-sm md:text-base"
                    title={isCollapsed ? "Catálogo" : ""}
                >
                    <GrCatalog className="text-lg md:text-xl flex-shrink-0" />
                    {!isCollapsed && <span className="whitespace-nowrap">Catálogo</span>}
                </Link>
                <Link
                    to="/mi-red"
                    className="flex items-center gap-2 md:gap-3 p-2 md:p-3 rounded hover:bg-slate-800 transition text-sm md:text-base"
                    title={isCollapsed ? "Mi Red" : ""}
                >
                    <FaNetworkWired className="text-lg md:text-xl flex-shrink-0" />
                    {!isCollapsed && <span className="whitespace-nowrap">Mi Red</span>}
                </Link>
            </nav>
        </aside>
    );
};

export default Sidebar;