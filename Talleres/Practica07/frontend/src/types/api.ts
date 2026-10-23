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
  email_usuario?: string;
  usuario_id?: number;
}
export interface PedidoDetalle {
  id: number;
  usuario_id?: number;
  email_usuario: string;
  total: number;
  metodo_pago: string;
  estado: Pedido['estado'];
  created_at: string;
  items?: PedidoItem[];
}

export const ESTADOS_PEDIDO = [
  'pendiente',
  'preparando',
  'enviado',
  'entregado',
  'cancelado',
] as const;
export type EstadoPedidoFE = (typeof ESTADOS_PEDIDO)[number];

// FSM local (mirror del backend; el server es autoridad)
export const TRANSICIONES_VALIDAS: Record<EstadoPedidoFE, EstadoPedidoFE[]> = {
  pendiente: ['preparando', 'cancelado'],
  preparando: ['enviado', 'cancelado'],
  enviado: ['entregado', 'cancelado'],
  entregado: [],
  cancelado: [],
};

// ── Red / Comisiones ────────────────────────────────────────
export interface NodoRed {
  id: number;
  codigo: string;
  nombre: string;
  nivel: number;
  ventas: number;
  hijos?: NodoRed[];
}
export type EstadoComision = 'pendiente' | 'aprobada' | 'pagada' | 'revertida';

/** Fila del ledger: quién cobró, cuánto, en qué nivel y por qué pedido. */
export interface Comision {
  id: number;
  pedido_id?: number;
  referido_id?: number;
  nivel: number;
  porcentaje: number;
  monto: number;
  estado: EstadoComision;
  creado_en: string;
}

/** Agregado por estado — responde "¿cuánto me deben?". */
export interface ComisionTotal {
  estado: EstadoComision;
  total: number;
  cantidad: number;
}

/** Respuesta de /api/comisiones/mias y /api/admin/comisiones. */
export interface ComisionResumen {
  data: Comision[];
  page: number;
  limit: number;
  total: number;
  totales: ComisionTotal[];
}

// ── Liquidaciones ───────────────────────────────────────────
export type EstadoLiquidacion = 'pendiente' | 'pagada';

export interface Liquidacion {
  id: number;
  referido_id: number;
  total: number;
  periodo?: string;
  estado: EstadoLiquidacion;
  creado_en: string;
  pagada_en?: string | null;
}

/**
 * Fila de la pantalla de liquidación: cuánto se le debe a cada distribuidor
 * y cuántas comisiones componen esa deuda. `ultima_pago` es "" cuando
 * nunca se le pagó, no null.
 */
export interface DeudaDistribuidor {
  referido_id: number;
  codigo: string;
  nombre: string;
  total: number;
  comisiones: number;
  ultima_pago?: string;
}

// ── Reportes ────────────────────────────────────────────────
// Mirror exacto de openapi.yaml → components.schemas.*Reportes*.
// Si un campo pasa a camelCase aquí, el backend ya no casa y la tarjeta
// del dashboard muestra `undefined` en vez de fallar.

/** KPIs del periodo. `total_vendido` y `num_pedidos` excluyen cancelados. */
export interface ReporteResumen {
  total_vendido: number;
  num_pedidos: number;
  ticket_promedio: number;
  comisiones_devengadas: number;
}

/** Fila del ranking de productos. `nombre` es la foto histórica del pedido. */
export interface TopProducto {
  producto_id: number;
  nombre: string;
  unidades: number;
  ingresos: number;
}

/**
 * Fila del ranking de distribuidores.
 *
 * `ventas` viene del contador acumulado `referidos.ventas`, que el backend
 * actualiza best-effort tras el commit del checkout: puede ir un instante por
 * detrás de `comisiones`, que sí se lee del ledger. Para cifras de auditoría
 * usar `/api/admin/comisiones`.
 */
export interface TopReferido {
  referido_id: number;
  codigo: string;
  nombre: string;
  ventas: number;
  comisiones: number;
}

// ── Health ──────────────────────────────────────────────────
export interface Health {
  status: 'ok' | 'degraded';
  pg: 'up' | 'down';
  redis: 'up' | 'down';
  version?: string;
}