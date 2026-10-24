// Formatters compartidos por los widgets de reportes.
//
// Se viven en un .ts (no .tsx) a propósito: el eslint-plugin-react-refresh
// avisa cuando un archivo de componente exporta algo que no es un componente,
// y los helpers no lo son.

/** Monto en dólares con 2 decimales. */
export const formatCurrency = (n: number): string => `$${n.toFixed(2)}`;

/** Entero con separador de miles (es-EC usa punto como separador). */
export const formatNumero = (n: number): string =>
  new Intl.NumberFormat('es-EC').format(n);

/** Fecha corta. Si el ISO no parsea, se devuelve tal cual en vez de "Invalid Date". */
export const formatFecha = (iso: string): string => {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleDateString('es-EC', {
    year: 'numeric',
    month: 'short',
    day: '2-digit',
  });
};

/**
 * Ancho porcentual para una barra, acotado a [0, 100].
 *
 * Sin el acotado, un valor negativo produce un width CSS inválido y la barra
 * desaparece; uno mayor que el máximo se sale de la tarjeta.
 */
export const anchoBarra = (valor: number, maximo: number): number => {
  // La guarda tiene que comprobar AMBOS operandos con isFinite. `NaN <= 0`
  // es `false` en JS, así que validar sólo `maximo <= 0` deja pasar un
  // máximo NaN: la división da NaN, Math.min(100, NaN) da NaN, y el style
  // acaba siendo `width: "NaN%"`, que el navegador descarta en silencio y la
  // barra desaparece sin error en consola.
  if (!Number.isFinite(valor) || !Number.isFinite(maximo) || maximo <= 0) {
    return 0;
  }
  return Math.max(0, Math.min(100, (valor / maximo) * 100));
};