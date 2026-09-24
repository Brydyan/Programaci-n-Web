import { Link, useNavigate } from "react-router-dom";
import { useCart } from "../context/CartContext";
import { useAuth } from "../context/AuthContext";
import { useSidebar } from "../context/SidebarContext";
import { GiHamburgerMenu } from "react-icons/gi";

export default function Navbar() {
  const { totalItems } = useCart();
  const { logout, user } = useAuth();
  const { toggleCollapsed } = useSidebar();
  const navigate = useNavigate();

  const handleLogout = () => {
    logout();
    navigate("/login");
  };

  const userInitials = user?.email ? user.email.substring(0, 2).toUpperCase() : "U";

  return (
    <header className="h-16 bg-white border-b border-gray-200 flex items-center justify-between px-4 md:px-8 gap-4 shadow-sm">
      <div className="flex items-center gap-2 md:gap-4 min-w-0">
        <button
          onClick={toggleCollapsed}
          className="p-2 hover:bg-gray-100 rounded-lg transition flex-shrink-0"
          title="Toggle Sidebar"
        >
          <GiHamburgerMenu className="text-xl md:text-2xl text-gray-600" />
        </button>
        <h2 className="text-gray-700 font-semibold text-sm md:text-lg truncate">
          MultiCatálogo — {user?.rol === 'admin' ? 'Panel Admin' : 'Tienda'}
        </h2>
      </div>

      <div className="flex items-center gap-3 md:gap-6 flex-shrink-0">
        {/* Carrito */}
        <Link
          to="/carrito"
          className="relative p-2 hover:bg-gray-100 rounded-full transition"
          title="Carrito"
        >
          <span className="text-2xl">🛒</span>
          {totalItems > 0 && (
            <span className="absolute top-0 right-0 bg-rose-500 text-white text-xs font-bold w-5 h-5 flex items-center justify-center rounded-full transform translate-x-1 -translate-y-1">
              {totalItems}
            </span>
          )}
        </Link>

        {/* Divider */}
        <div className="hidden md:block w-px h-6 bg-gray-200"></div>

        {/* User profile dropdown */}
        <div className="relative group cursor-pointer">
          <div className="flex items-center gap-3">
            {/* Avatar */}
            <div className="w-10 h-10 rounded-full bg-gradient-to-br from-rose-400 to-pink-500 flex items-center justify-center text-white font-bold border-2 border-gray-100 overflow-hidden">
              {userInitials}
            </div>

            {/* User info visible on md+ */}
            <div className="hidden md:flex flex-col text-right">
              <span className="text-xs md:text-sm text-gray-900 font-semibold truncate max-w-40">
                {user?.email?.split('@')[0]}
              </span>
              <span className={`text-xs font-bold uppercase ${
                user?.rol === 'admin' ? 'text-amber-600' : 'text-blue-600'
              }`}>
                {user?.rol === 'admin' ? 'Administrador' : 'Cliente'}
              </span>
            </div>
          </div>

          {/* Dropdown menu */}
          <div className="absolute right-0 top-full mt-2 w-48 bg-white border border-gray-200 rounded-xl shadow-xl opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all duration-200 z-50">
            {/* Header */}
            <div className="px-4 py-3 border-b border-gray-100">
              <p className="text-sm font-semibold text-gray-900">{user?.email}</p>
              <p className="text-xs text-gray-600 mt-1">
                {user?.rol === 'admin' ? 'Administrador' : 'Cliente'}
              </p>
            </div>

            {/* Menu items */}
            <div className="py-2">
              <Link
                to="/"
                className="block px-4 py-2 text-sm text-gray-700 hover:bg-gray-50 transition"
              >
                📊 Dashboard
              </Link>
              <Link
                to="/tienda"
                className="block px-4 py-2 text-sm text-gray-700 hover:bg-gray-50 transition"
              >
                🏪 Tienda
              </Link>
              <Link
                to="/catalogo"
                className="block px-4 py-2 text-sm text-gray-700 hover:bg-gray-50 transition"
              >
                📋 Catálogo
              </Link>
              <Link
                to="/carrito"
                className="block px-4 py-2 text-sm text-gray-700 hover:bg-gray-50 transition"
              >
                🛒 Carrito ({totalItems})
              </Link>

              {user?.rol !== 'admin' && (
                <Link
                  to="/mi-red"
                  className="block px-4 py-2 text-sm text-gray-700 hover:bg-gray-50 transition"
                >
                  🌳 Mi Red
                </Link>
              )}
            </div>

            {/* Logout */}
            <div className="border-t border-gray-100 py-2">
              <button
                onClick={handleLogout}
                className="w-full text-left px-4 py-2 text-sm text-red-600 font-semibold hover:bg-red-50 transition"
              >
                🚪 Cerrar Sesión
              </button>
            </div>
          </div>
        </div>
      </div>
    </header>
  );
}