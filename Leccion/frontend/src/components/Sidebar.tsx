import { Link } from "react-router-dom";
import { MdOutlineDashboard } from "react-icons/md";
import { GrCatalog } from "react-icons/gr";
import { FaNetworkWired } from "react-icons/fa";
import { MdStorefront, MdSettings, MdAnalytics } from "react-icons/md";
import { useSidebar } from "../context/SidebarContext";
import { useAuth } from "../context/AuthContext";

export default function Sidebar() {
  const { isCollapsed } = useSidebar();
  const { user } = useAuth();

  const isAdmin = user?.rol === 'admin';

  return (
    <aside
      className={`bg-gradient-to-b from-gray-900 to-gray-800 text-white flex flex-col transition-all duration-300 h-screen border-r border-gray-700 ${
        isCollapsed ? "w-20" : "w-64"
      }`}
    >
      {/* Logo */}
      <div
        className={`p-4 md:p-6 text-lg md:text-2xl font-bold border-b border-gray-700 flex items-center justify-center transition-all ${
          isCollapsed ? "text-lg" : ""
        }`}
      >
        {isCollapsed ? "MC" : "MultiCatálogo"}
      </div>

      {/* Rol badge */}
      {!isCollapsed && (
        <div className="px-4 py-2 mx-4 mt-4 bg-rose-500 text-white text-xs font-bold rounded-lg text-center uppercase">
          {isAdmin ? "Admin" : "Cliente"}
        </div>
      )}

      {/* Nav */}
      <nav className="flex-1 p-2 md:p-4 space-y-1 md:space-y-2">
        {/* Dashboard */}
        <Link
          to="/"
          className="flex items-center gap-2 md:gap-3 p-2 md:p-3 rounded hover:bg-gray-700 transition text-sm md:text-base"
          title={isCollapsed ? "Dashboard" : ""}
        >
          <MdOutlineDashboard className="text-lg md:text-xl flex-shrink-0" />
          {!isCollapsed && <span className="whitespace-nowrap">Dashboard</span>}
        </Link>

        {/* Tienda */}
        <Link
          to="/tienda"
          className="flex items-center gap-2 md:gap-3 p-2 md:p-3 rounded hover:bg-gray-700 transition text-sm md:text-base"
          title={isCollapsed ? "Tienda" : ""}
        >
          <MdStorefront className="text-lg md:text-xl flex-shrink-0" />
          {!isCollapsed && <span className="whitespace-nowrap">Tienda</span>}
        </Link>

        {/* Catálogo */}
        <Link
          to="/catalogo"
          className="flex items-center gap-2 md:gap-3 p-2 md:p-3 rounded hover:bg-gray-700 transition text-sm md:text-base"
          title={isCollapsed ? "Catálogo" : ""}
        >
          <GrCatalog className="text-lg md:text-xl flex-shrink-0" />
          {!isCollapsed && <span className="whitespace-nowrap">Catálogo</span>}
        </Link>

        {/* Mi Red (Cliente) */}
        {!isAdmin && (
          <Link
            to="/mi-red"
            className="flex items-center gap-2 md:gap-3 p-2 md:p-3 rounded hover:bg-gray-700 transition text-sm md:text-base"
            title={isCollapsed ? "Mi Red" : ""}
          >
            <FaNetworkWired className="text-lg md:text-xl flex-shrink-0" />
            {!isCollapsed && <span className="whitespace-nowrap">Mi Red</span>}
          </Link>
        )}

        {/* Separador admin */}
        {isAdmin && (
          <div className="my-4 border-t border-gray-700"></div>
        )}

        {/* Panel Admin */}
        {isAdmin && (
          <>
            <div className={`text-gray-400 text-xs font-bold uppercase px-3 py-2 ${isCollapsed ? 'hidden' : ''}`}>
              Administración
            </div>

            <Link
              to="/mi-red"
              className="flex items-center gap-2 md:gap-3 p-2 md:p-3 rounded hover:bg-gray-700 transition text-sm md:text-base"
              title={isCollapsed ? "Red MLM" : ""}
            >
              <FaNetworkWired className="text-lg md:text-xl flex-shrink-0" />
              {!isCollapsed && <span className="whitespace-nowrap">Red MLM</span>}
            </Link>

            <Link
              to="/reportes"
              className="flex items-center gap-2 md:gap-3 p-2 md:p-3 rounded hover:bg-gray-700 transition text-sm md:text-base"
              title={isCollapsed ? "Reportes" : ""}
            >
              <MdAnalytics className="text-lg md:text-xl flex-shrink-0" />
              {!isCollapsed && <span className="whitespace-nowrap">Reportes</span>}
            </Link>

            <Link
              to="/configuracion"
              className="flex items-center gap-2 md:gap-3 p-2 md:p-3 rounded hover:bg-gray-700 transition text-sm md:text-base"
              title={isCollapsed ? "Configuración" : ""}
            >
              <MdSettings className="text-lg md:text-xl flex-shrink-0" />
              {!isCollapsed && <span className="whitespace-nowrap">Configuración</span>}
            </Link>
          </>
        )}
      </nav>
    </aside>
  );
}