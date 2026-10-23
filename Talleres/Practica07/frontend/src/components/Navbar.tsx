import { Link, useNavigate } from "react-router-dom";
import { useState } from "react";
import { useCart } from "../context/cart";
import { useAuth } from "../context/auth";
import { useSidebar } from "../context/sidebar";
import { GiHamburgerMenu } from "react-icons/gi";

const Navbar = () => {
    const { totalItems } = useCart();
    const { logout, userEmail, rol } = useAuth();
    const { toggleCollapsed } = useSidebar();
    const [menuOpen, setMenuOpen] = useState(false);
    const navigate = useNavigate();

    const handleLogout = () => {
        logout();
        setMenuOpen(false);
        navigate("/login", { replace: true });
    };
    return (
        <header className="h-16 bg-white border-b border-slate-200 flex items-center justify-between px-4 md:px-8 gap-4">
            <div className="flex items-center gap-2 md:gap-4 min-w-0">
                <button
                    onClick={toggleCollapsed}
                    className="p-2 hover:bg-slate-100 rounded-lg transition flex-shrink-0"
                    title="Toggle Sidebar"
                    aria-label="Abrir/cerrar menú lateral"
                >
                    <GiHamburgerMenu className="text-xl md:text-2xl text-slate-600" />
                </button>
                {/* El título decía "Panel de Administración" fijo, así que un
                    cliente leía exactamente lo mismo que el admin. */}
                <h2 className="text-slate-600 font-medium text-sm md:text-lg truncate">
                    {rol === "admin" ? "Panel de Administración" : "Tienda"}
                </h2>
            </div>
            <div className="flex items-end gap-3 md:gap-6 flex-shrink-0">
                <Link
                    to="/carrito"
                    className="relative p-2 hover:bg-slate-100 rounded-full transition"
                    aria-label="Carrito"
                >
                    <span className="text-xl">🛒</span>
                    {totalItems > 0 && (
                        <span className="absolute top-0 right-0 bg-indigo-600 text-white
text-xs font-bold w-5 h-5 flex items-center justify-center rounded-full
transform translate-x-1 -translate-y-1">
                            {totalItems}
                        </span>
                    )}
                </Link>
                <div className="flex items-center gap-2 md:gap-4">
                    <span className="text-xs md:text-sm text-slate-500 hidden sm:inline truncate">{userEmail}</span>
                    {rol && (
                        <span
                            className={`hidden sm:inline-flex items-center px-2 py-0.5 rounded-full text-xs font-semibold ${
                                rol === "admin"
                                    ? "bg-indigo-100 text-indigo-700"
                                    : "bg-emerald-100 text-emerald-700"
                            }`}
                        >
                            {rol === "admin" ? "Admin" : "Cliente"}
                        </span>
                    )}
                    {/* Menú usuario: click-toggle (funciona en mobile) + hover en desktop */}
                    <div className="relative">
                        <button
                            type="button"
                            onClick={() => setMenuOpen((v) => !v)}
                            onBlur={() => setTimeout(() => setMenuOpen(false), 150)}
                            className="w-8 h-8 rounded-full bg-slate-200 overflow-hidden
border border-slate-300 flex items-center justify-center hover:ring-2 hover:ring-indigo-300 transition"
                            aria-label="Abrir menú de usuario"
                            aria-haspopup="menu"
                            aria-expanded={menuOpen}
                        >
                            <img
                                src="https://fastly.picsum.photos/id/64/4326/2884.jpg?hmac=9_SzX666YRpR_fOyYStXpfSiJ_edO3ghlSRnH2w09Kg"
                                alt="Avatar del usuario"
                                className="w-full h-full object-cover"
                            />
                        </button>
                        {menuOpen && (
                            <div
                                role="menu"
                                className="absolute right-0 top-full mt-1 w-44 bg-white border
border-slate-200 rounded-lg shadow-lg z-50 overflow-hidden"
                            >
                                <div className="px-4 py-2 text-xs text-slate-500 border-b border-slate-100">
                                    Sesión: {userEmail}
                                </div>
                                <button
                                    type="button"
                                    onClick={handleLogout}
                                    role="menuitem"
                                    className="w-full text-left px-4 py-2 text-sm text-red-600 font-semibold hover:bg-red-50 transition-colors flex items-center gap-2"
                                >
                                    <span aria-hidden>↪</span> Cerrar Sesión
                                </button>
                            </div>
                        )}
                    </div>
                </div>
            </div>
        </header>
    );
};
export default Navbar;