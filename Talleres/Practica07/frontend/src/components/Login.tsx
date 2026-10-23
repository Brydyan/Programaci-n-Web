import { useState, useEffect, type FormEvent } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useAuth } from '../context/auth';

const Login = () => {
    const [email, setEmail] = useState<string>('');
    const [password, setPassword] = useState<string>('');
    const [error, setError] = useState<string>('');

    const navigate = useNavigate();
    const { login, isLoading, isAuthenticated, rol } = useAuth();

    // Si ya está autenticado, redirigir según su rol
    useEffect(() => {
        if (isAuthenticated) {
            navigate(rol === 'admin' ? '/' : '/catalogo', { replace: true });
        }
    }, [isAuthenticated, rol, navigate]);

    const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
        e.preventDefault();
        setError('');

        try {
            const data = await login(email, password);
            navigate(data.rol === 'admin' ? '/' : '/catalogo', { replace: true });
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Error al iniciar sesión');
        }
    };
    return (
        <div className="min-h-screen flex items-center justify-center bg-slate-100
px-4">
            <div className="max-w-md w-full bg-white rounded-lg shadow-md p-8
border border-slate-200">
                <div className="text-center mb-8">
                    <h2 className="text-3xl font-bold text-slate-900">Andy Alejandro</h2>
                    <p className="text-sm font-medium text-marca-600 mt-1">7mo semestre</p>
                    <p className="text-slate-500 mt-2">Ingresa a tu cuenta para
                        continuar</p>
                </div>
                {error && (
                    <div className="bg-red-50 text-red-600 p-3 rounded-lg text-sm mb-6
text-center border border-red-200">
                        {error}
                    </div>
                )}
                <form onSubmit={handleSubmit} className="space-y-6">
                    <div>
                        <label className="block text-sm font-medium text-slate-700 mb2">Correo Electrónico</label>
                        <input
                            type="email"
                            value={email}
                            onChange={(e) => setEmail(e.target.value)}
                            className="w-full px-4 py-3 rounded-lg border border-slate-300
focus:ring-2 focus:ring-marca-600 focus:border-marca-600 outline-none
transition"
                            placeholder="admin@upse.edu.ec"
                            required
                        />
                    </div>
                    <div>
                        <label className="block text-sm font-medium text-slate-700 mb2">Contraseña</label>
                        <input
                            type="password"
                            value={password}
                            onChange={(e) => setPassword(e.target.value)}
                            className="w-full px-4 py-3 rounded-lg border border-slate-300
focus:ring-2 focus:ring-marca-600 focus:border-marca-600 outline-none
transition"
                            placeholder="••••••"
                            required
                        />
                    </div>
                    <button
                        type="submit"
                        disabled={isLoading}
                        className="w-full bg-marca-600 text-white font-bold py-3 min-h-[44px] rounded-lg
hover:bg-marca-700 transition disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                        {isLoading ? 'Iniciando...' : 'Iniciar Sesión'}
                    </button>
                </form>
                <p className="text-center text-sm text-slate-600 mt-4">
                    ¿No tienes cuenta?{' '}
                    <Link to="/registro" className="text-slate-900 font-semibold hover:underline">
                        Crear cuenta
                    </Link>
                </p>
            </div>
        </div>
    );
};
export default Login;