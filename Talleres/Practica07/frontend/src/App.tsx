import { BrowserRouter, Routes, Route, Navigate, Outlet } from 'react-router-dom';
import Layout from './components/Layout';
import Dashboard from './components/Dashboard';
import Catalogo from './components/Catalogo';
import MiRed from './components/MiRed';
import MisGanancias from './components/MisGanancias';
import Carrito from './components/Carrito';
import Login from './components/Login';
import Registro from './components/Registro';
import DetalleProducto from './components/DetalleProducto';
import Checkout from './components/Checkout';
import Confirmacion from './components/Confirmacion';
import AdminProductos from './components/AdminProductos';
import MisPedidos from './components/MisPedidos';
import AdminPedidos from './components/AdminPedidos';
import Liquidaciones from './components/Liquidaciones';
import { CartProvider } from './context/CartContext';
import { useAuth } from './context/auth';
import { AuthProvider } from './context/AuthContext';
import { SidebarProvider } from './context/SidebarContext';
import { AdminRoute } from './components/AdminRoute';
import InstallPrompt from './components/InstallPrompt';
import OfflineBanner from './components/OfflineBanner';
import UpdatePrompt from './components/UpdatePrompt';

// Componente para proteger las rutas privadas
const ProtectedRoute = () => {
  const { isAuthenticated } = useAuth();

  // Si no está autenticado, lo enviamos al login
  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  // Si está autenticado, renderiza las rutas hijas (Outlet)
  return <Outlet />;
};
function App() {
  return (
    <AuthProvider> {/* Proveedor de Autenticación */}
      <CartProvider> {/* Proveedor del Carrito */}
        <SidebarProvider> {/* Proveedor del Sidebar */}
          {/* Avisos de PWA: se montan fuera de los providers para que
             ufliquen sobre cualquier pantalla. Ambos devuelven null cuando
              no hay nada que mostrar. */}
          <OfflineBanner />
          <UpdatePrompt />
          <InstallPrompt />
          <BrowserRouter>
          <Routes>
            {/* Ruta pública */}
            <Route path="/login" element={<Login />} />
            <Route path="/registro" element={<Registro />} />
            {/* Rutas protegidas (cualquier usuario autenticado) */}
            <Route element={<ProtectedRoute />}>
              <Route path="/" element={<Layout />}>
                <Route path="catalogo" element={<Catalogo />} />
                <Route path="carrito" element={<Carrito />} />
                <Route path="producto/:id" element={<DetalleProducto />} />
                <Route path="productos/:id" element={<DetalleProducto />} />
                <Route path="checkout" element={<Checkout />} />
                <Route path="confirmacion" element={<Confirmacion />} />
                <Route path="mis-pedidos" element={<MisPedidos />} />
                {/* Cada usuario tiene su propia red: no es una vista de admin. */}
                <Route path="mi-red" element={<MiRed />} />
                {/* El lado del dinero: ledger devengado y cobros. */}
                <Route path="mis-ganancias" element={<MisGanancias />} />
              </Route>
            </Route>

            {/* Rutas solo admin */}
            <Route element={<AdminRoute />}>
              <Route path="/" element={<Layout />}>
                <Route index element={<Dashboard />} />
                <Route path="admin/productos" element={<AdminProductos />} />
                <Route path="admin/pedidos" element={<AdminPedidos />} />
                {/* Cobro de comisiones: consultar deuda y pagarla. */}
                <Route path="admin/liquidaciones" element={<Liquidaciones />} />
              </Route>
            </Route>

            {/* Ruta comodín para capturar 404 y redirigir */}
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </BrowserRouter>
        </SidebarProvider>
      </CartProvider>
    </AuthProvider>
  );
}
export default App;