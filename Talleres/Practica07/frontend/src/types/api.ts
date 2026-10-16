// Tipos espejo del contrato OpenAPI (openapi.yaml).
// Cualquier cambio en openapi.yaml debe reflejarse aquí.

export type Rol = 'cliente' | 'admin';

export type ErrorCode =
  | 'VALIDATION_ERROR'
  | 'UNAUTHORIZED'
  | 'FORBIDDEN'
  | 'NOT_FOUND'
  | 'CONFLICT'
  | 'RATE_LIMITED'
  | 'INVALID_TRANSITION'
  | 'STOCK_INSUFICIENTE'
  | 'INTERNAL';

export interface APIError {
  status: number;
  code: ErrorCode;
  message: string;
  details?: Record<string, unknown>;
}

export class ApiError extends Error {
  public readonly apiError: APIError;
  constructor(apiError: APIError) {
    super(apiError.message);
    this.name = 'ApiError';
    this.apiError = apiError;
  }
  get status(): number {
    return this.apiError.status;
  }
  get code(): ErrorCode {
    return this.apiError.code;
  }
}

export function getErrorMessage(err: unknown): string {
  if (err instanceof ApiError) return err.apiError.message;
  if (err instanceof Error) return err.message;
  return 'Error desconocido';
}

export interface Pagina<T> {
  data: T[];
  page: number;
  limit: number;
  total: number;
}

// ── Auth ────────────────────────────────────────────────────
export interface LoginRequest {
  email: string;
  password: string;
}
export interface LoginResponse {
  token: string;
  email: string;
  rol: Rol;
}
export interface RegisterRequest {
  nombre: string;
  email: string;
  password: string;
  ref?: string;
}
export interface RegisterResponse {
  id: number;
  email: string;
  codigo_ref: string;
}

// ── Productos ───────────────────────────────────────────────
export interface Producto {
  id: number;
  nombre: string;
  precio: number;
  categoria: string;
  stock: number;
  activo: boolean;
  img: string;
}
export interface ProductoCreate {
  nombre: string;
  precio: number;
  categoria: string;
  stock: number;
  img?: string;
}
export interface ProductoUpdate {
  nombre?: string;
  precio?: number;
  categoria?: string;
  stock?: number;
  img?: string;
  activo?: boolean;
}

// ── Pedidos ─────────────────────────────────────────────────
export interface ItemCarrito {
  id: number;
  nombre: string;
  precio: number;
  cantidad: number;
}
export interface CheckoutRequest {
  nombre?: string;
  direccion: string;
  ciudad: string;
  telefono: string;
  metodo_pago: string;
  items: ItemCarrito[];
}
export interface PedidoItem {
  producto_id: number;
  nombre: string;
  precio: number;
  cantidad: number;
}
export interface Pedido {
  id: number;
  total: number;
  estado:
    | 'pendiente'
    | 'preparando'
    | 'enviado'
    | 'entregado'
    | 'cancelado';
  created_at: string;
  items?: PedidoItem[];
}
export interface PedidoResumen {
  id: number;
  total: number;
  estado: Pedido['estado'];
  created_at: string;
}

// ── Red / Comisiones ────────────────────────────────────────
export interface NodoRed {
  id: number;
  nombre: string;
  nivel: number;
  ventas: number;
  hijos?: NodoRed[];
}
export interface Comision {
  id: number;
  pedido_id: number;
  nivel: number;
  porcentaje: number;
  monto: number;
  estado: 'pendiente' | 'aprobada' | 'pagada' | 'revertida';
  creado_en: string;
}

// ── Liquidaciones ───────────────────────────────────────────
export interface Liquidacion {
  id: number;
  referido_id: number;
  total: number;
  periodo?: string;
  estado: 'pendiente' | 'pagada';
  creado_en: string;
  pagada_en?: string | null;
}

// ── Reportes ────────────────────────────────────────────────
export interface ReporteResumen {
  total_vendido: number;
  num_pedidos: number;
  ticket_promedio: number;
  comisiones_devengadas: number;
}
export interface TopProducto {
  producto_id: number;
  nombre: string;
  unidades: number;
  ingresos: number;
}

// ── Health ──────────────────────────────────────────────────
export interface Health {
  status: 'ok' | 'degraded';
  pg: 'up' | 'down';
  redis: 'up' | 'down';
  version?: string;
}