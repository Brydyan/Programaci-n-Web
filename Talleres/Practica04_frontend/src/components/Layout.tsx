import { Outlet } from "react-router-dom";
import Sidebar from "./Sidebar";
import Navbar from "./Navbar";
import { useSidebar } from "../context/SidebarContext";
const Layout = () => {
    const { isCollapsed, toggleCollapsed } = useSidebar();

    return (
        <div className="flex h-screen bg-slate-50">
            {/* Sidebar - drawer en móvil cuando expandido, static en desktop */}
            <div className={`${
                !isCollapsed ? "fixed md:static inset-y-0 left-0 z-40" : ""
            } transition-transform duration-300`}>
                <Sidebar />
            </div>

            {/* Overlay en móvil cuando sidebar está expandido - clickeable para cerrar */}
            {!isCollapsed && (
                <div
                    className="fixed inset-0 md:hidden z-20 cursor-pointer"
                    onClick={toggleCollapsed}
                />
            )}

            {/* Área de Contenido Principal */}
            <div className="flex-1 flex flex-col overflow-hidden relative md:z-50">
                <Navbar />
                <main className="flex-1 overflow-x-hidden overflow-y-auto p-4 md:p-8">
                    <Outlet />
                </main>
            </div>
        </div>
    );
};
export default Layout;