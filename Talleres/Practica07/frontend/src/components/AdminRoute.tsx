import { Navigate, Outlet } from 'react-router-dom';
import { useAuth } from '../context/auth';

export const AdminRoute = () => {
  const { isAuthenticated, rol } = useAuth();
  if (!isAuthenticated) return <Navigate to="/login" replace />;
  if (rol !== 'admin') return <Navigate to="/" replace />;
  return <Outlet />;
};