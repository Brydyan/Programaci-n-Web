import { useEffect, useMemo, useState, type FormEvent } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { useAuth } from '../context/auth';
import { ApiError } from '../types/api';

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
const REF_RE = /^[A-Z0-9]+$/;

type FieldErrors = Partial<
  Record<'nombre' | 'email' | 'password' | 'confirmPassword' | 'ref', string>
>;

const Registro = () => {
  const [searchParams] = useSearchParams();
  const initialRef = useMemo(
    () => searchParams.get('ref')?.toUpperCase() ?? '',
    [searchParams],
  );

  const [nombre, setNombre] = useState<string>('');
  const [email, setEmail] = useState<string>('');
  const [password, setPassword] = useState<string>('');
  const [confirmPassword, setConfirmPassword] = useState<string>('');
  const [ref, setRef] = useState<string>(initialRef);

  const [showPassword, setShowPassword] = useState<boolean>(false);
  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({});
  const [emailConflictHelp, setEmailConflictHelp] = useState<boolean>(false);

  const navigate = useNavigate();
  const { register, isAuthenticated, rol } = useAuth();

  // Si ya está autenticado, redirigir a su home según rol
  useEffect(() => {
    if (isAuthenticated) {
      navigate(rol === 'admin' ? '/' : '/catalogo', { replace: true });
    }
  }, [isAuthenticated, rol, navigate]);

  const validate = (): FieldErrors => {
    const next: FieldErrors = {};
    if (nombre.trim().length < 2) {
      next.nombre = 'Ingresa tu nombre (mínimo 2 caracteres).';
    }
    if (!EMAIL_RE.test(email.trim())) {
      next.email = 'Ingresa un correo electrónico válido.';
    }
    if (password.length < 8) {
      next.password = 'La contraseña debe tener al menos 8 caracteres.';
    }
    if (confirmPassword !== password) {
      next.confirmPassword = 'Las contraseñas no coinciden.';
    }
    const refTrim = ref.trim();
    if (refTrim.length > 0) {
      const len = refTrim.length;
      if (len < 4 || len > 8 || !REF_RE.test(refTrim)) {
        next.ref = 'Código inválido (4-8 caracteres alfanuméricos).';
      }
    }
    return next;
  };

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setError(null);
    setSuccess(null);
    setEmailConflictHelp(false);

    const next = validate();
    setFieldErrors(next);
    if (Object.keys(next).length > 0) return;

    setIsLoading(true);
    try {
      await register(
        nombre.trim(),
        email.trim(),
        password,
        ref.trim() ? ref.trim().toUpperCase() : undefined,
      );
      setSuccess('Cuenta creada. Iniciando sesión...');
      navigate('/', { replace: true });
    } catch (err) {
      if (err instanceof ApiError && err.code === 'CONFLICT') {
        setEmailConflictHelp(true);
        setError('Este correo ya está registrado. Inicia sesión o usa otro correo.');
      } else {
        setError(
          err instanceof Error ? err.message : 'No pudimos crear tu cuenta. Intenta de nuevo.',
        );
      }
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-slate-100 px-4 py-8">
      <div className="max-w-md w-full bg-white rounded-xl shadow-md p-8 border border-slate-200">
        <div className="text-center mb-8">
          <h2 className="text-3xl font-bold text-slate-900">Crear cuenta</h2>
          <p className="text-slate-500 mt-2">Únete a MultiCatálogo en segundos</p>
        </div>

        {success && (
          <div className="bg-green-50 text-green-700 p-3 rounded-lg text-sm mb-6 text-center border border-green-200">
            {success}
          </div>
        )}
        {error && (
          <div className="bg-red-50 text-red-600 p-3 rounded-lg text-sm mb-6 text-center border border-red-200">
            <p>{error}</p>
            {emailConflictHelp && (
              <Link
                to="/login"
                className="block mt-2 text-slate-900 font-semibold hover:underline"
              >
                Ir a iniciar sesión
              </Link>
            )}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-5" noValidate>
          <div>
            <label
              htmlFor="registro-nombre"
              className="block text-sm font-medium text-slate-700 mb-2"
            >
              Nombre
            </label>
            <input
              id="registro-nombre"
              type="text"
              autoComplete="name"
              value={nombre}
              onChange={(e) => setNombre(e.target.value)}
              className="w-full min-h-[44px] px-4 py-3 rounded-lg border border-slate-300 focus:ring-2 focus:ring-indigo-600 focus:border-indigo-600 outline-none transition"
              placeholder="Tu nombre"
              aria-invalid={Boolean(fieldErrors.nombre)}
              required
            />
            {fieldErrors.nombre && (
              <p className="mt-1 text-xs text-red-600">{fieldErrors.nombre}</p>
            )}
          </div>

          <div>
            <label
              htmlFor="registro-email"
              className="block text-sm font-medium text-slate-700 mb-2"
            >
              Correo Electrónico
            </label>
            <input
              id="registro-email"
              type="email"
              autoComplete="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full min-h-[44px] px-4 py-3 rounded-lg border border-slate-300 focus:ring-2 focus:ring-indigo-600 focus:border-indigo-600 outline-none transition"
              placeholder="tu@correo.com"
              aria-invalid={Boolean(fieldErrors.email)}
              required
            />
            {fieldErrors.email && (
              <p className="mt-1 text-xs text-red-600">{fieldErrors.email}</p>
            )}
          </div>

          <div>
            <label
              htmlFor="registro-password"
              className="block text-sm font-medium text-slate-700 mb-2"
            >
              Contraseña
            </label>
            <div className="relative">
              <input
                id="registro-password"
                type={showPassword ? 'text' : 'password'}
                autoComplete="new-password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full min-h-[44px] px-4 py-3 pr-24 rounded-lg border border-slate-300 focus:ring-2 focus:ring-indigo-600 focus:border-indigo-600 outline-none transition"
                placeholder="Mínimo 8 caracteres"
                aria-invalid={Boolean(fieldErrors.password)}
                required
              />
              <button
                type="button"
                onClick={() => setShowPassword((v) => !v)}
                className="absolute right-2 top-1/2 -translate-y-1/2 min-h-[36px] px-3 text-sm font-medium text-slate-600 hover:text-slate-900"
                aria-label={showPassword ? 'Ocultar contraseña' : 'Mostrar contraseña'}
              >
                {showPassword ? 'Ocultar' : 'Mostrar'}
              </button>
            </div>
            {fieldErrors.password && (
              <p className="mt-1 text-xs text-red-600">{fieldErrors.password}</p>
            )}
          </div>

          <div>
            <label
              htmlFor="registro-confirm"
              className="block text-sm font-medium text-slate-700 mb-2"
            >
              Confirmar contraseña
            </label>
            <input
              id="registro-confirm"
              type={showPassword ? 'text' : 'password'}
              autoComplete="new-password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              className="w-full min-h-[44px] px-4 py-3 rounded-lg border border-slate-300 focus:ring-2 focus:ring-indigo-600 focus:border-indigo-600 outline-none transition"
              placeholder="Repite tu contraseña"
              aria-invalid={Boolean(fieldErrors.confirmPassword)}
              required
            />
            {fieldErrors.confirmPassword && (
              <p className="mt-1 text-xs text-red-600">{fieldErrors.confirmPassword}</p>
            )}
          </div>

          <div>
            <label
              htmlFor="registro-ref"
              className="block text-sm font-medium text-slate-700 mb-2"
            >
              Código de referido <span className="text-slate-400">(opcional)</span>
            </label>
            <input
              id="registro-ref"
              type="text"
              inputMode="text"
              maxLength={8}
              value={ref}
              onChange={(e) => setRef(e.target.value.toUpperCase())}
              className="w-full min-h-[44px] px-4 py-3 rounded-lg border border-slate-300 focus:ring-2 focus:ring-indigo-600 focus:border-indigo-600 outline-none transition uppercase tracking-wider"
              placeholder="ABCD12"
              aria-invalid={Boolean(fieldErrors.ref)}
            />
            {fieldErrors.ref && (
              <p className="mt-1 text-xs text-red-600">{fieldErrors.ref}</p>
            )}
          </div>

          <button
            type="submit"
            disabled={isLoading}
            className="w-full min-h-[44px] bg-indigo-600 text-white font-bold py-3 rounded-lg hover:bg-indigo-700 transition disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isLoading ? 'Creando cuenta...' : 'Crear cuenta'}
          </button>
        </form>

        <p className="text-center text-sm text-slate-600 mt-6">
          ¿Ya tienes cuenta?{' '}
          <Link to="/login" className="text-slate-900 font-semibold hover:underline">
            Inicia sesión
          </Link>
        </p>
      </div>
    </div>
  );
};

export default Registro;
