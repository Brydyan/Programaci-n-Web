import { BrowserRouter, Routes, Route, Navigate, Outlet } from 'react-router-dom';
import Layout from './components/Layout';
import Dashboard from './components/Dashboard';
import Catalogo from './components/Catalogo';
import MiRed from './components/MiRed';
import Carrito from './components/Carrito';
import Login from './components/Login';
import Tienda from './components/Tienda';
import DetalleProducto from './components/DetalleProducto';
import Checkout from './components/Checkout';
import Reportes from './components/Reportes';
import Configuracion from './components/Configuracion';
import { CartProvider } from './context/CartContext';
import { AuthProvider, useAuth } from './context/AuthContext';
import { SidebarProvider } from './context/SidebarContext';

// Componente para proteger las rutas privadas
const ProtectedRoute = () => {
  const { isAuthenticated } = useAuth();

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  return <Outlet />;
};

// Componente para proteger rutas por rol
interface RoleProtectedRouteProps {
  allowedRoles: Array<"admin" | "cliente">;
}

const RoleProtectedRoute = ({ allowedRoles }: RoleProtectedRouteProps) => {
  const { isAuthenticated, userRole } = useAuth();

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  if (userRole && !allowedRoles.includes(userRole)) {
    return <Navigate to="/" replace />;
  }

  return <Outlet />;
};

function App() {
  return (
    <AuthProvider>
      <CartProvider>
        <SidebarProvider>
          <BrowserRouter>
            <Routes>
              {/* Ruta pública */}
              <Route path="/login" element={<Login />} />

              {/* Rutas protegidas generales */}
              <Route element={<ProtectedRoute />}>
                <Route path="/" element={<Layout />}>
                  <Route index element={<Dashboard />} />
                  <Route path="catalogo" element={<Catalogo />} />
                  <Route path="tienda" element={<Tienda />} />
                  <Route path="producto/:id" element={<DetalleProducto />} />
                  <Route path="carrito" element={<Carrito />} />
                  <Route path="checkout" element={<Checkout />} />
                  <Route path="mi-red" element={<MiRed />} />
                </Route>
              </Route>

              {/* Rutas por rol (ejemplo para admin) */}
              <Route element={<RoleProtectedRoute allowedRoles={["admin"]} />}>
                <Route path="/" element={<Layout />}>
                  <Route path="reportes" element={<Reportes />} />
                  <Route path="configuracion" element={<Configuracion />} />
                </Route>
              </Route>

              {/* Ruta comodín */}
              <Route path="*" element={<Navigate to="/" replace />} />
            </Routes>
          </BrowserRouter>
        </SidebarProvider>
      </CartProvider>
    </AuthProvider>
  );
}

export default App;