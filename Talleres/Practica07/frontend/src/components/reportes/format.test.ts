import { describe, expect, it } from 'vitest';
import {
  anchoBarra,
  formatCurrency,
  formatFecha,
  formatNumero,
} from './format';

// Funciones puras: son el mejor punto de entrada para probar porque no hay
// DOM, ni red, ni estado. Y son las que más bugs silenciosos acumulan: un
// `toFixed` mal colocado produce "12.3450" en la tabla de ventas y nadie se
// da cuenta hasta que un cliente lo reporta.

describe('formatCurrency', () => {
  it('antepone el símbolo y fija dos decimales', () => {
    expect(formatCurrency(0)).toBe('$0.00');
    expect(formatCurrency(12.5)).toBe('$12.50');
    expect(formatCurrency(12450)).toBe('$12450.00');
  });

  it('redondea a dos decimales', () => {
    expect(formatCurrency(12.345)).toBe('$12.35');
    expect(formatCurrency(12.344)).toBe('$12.34');
  });

  // Un total de carrito se calcula sumando floats. Sin el toFixed, 0.1+0.2
  // se muestra como 0.30000000000000004.
  it('absorbe el error de punto flotante de las sumas', () => {
    const suma = 0.1 + 0.2;
    expect(suma).not.toBe(0.3);
    expect(formatCurrency(suma)).toBe('$0.30');
  });
});

describe('formatNumero', () => {
  it('agrupa miles', () => {
    expect(formatNumero(0)).toBe('0');
    expect(formatNumero(1000)).toBe('1000'.replace(/(\d)(?=(\d{3})+$)/g, '$1.'));
  });

  it('no rompe con negativos', () => {
    expect(typeof formatNumero(-5)).toBe('string');
    expect(formatNumero(-5)).toContain('5');
  });
});

describe('formatFecha', () => {
  it('formatea un ISO válido', () => {
    const salida = formatFecha('2026-01-15T10:30:00Z');
    expect(salida).not.toBe('Invalid Date');
    expect(salida).toContain('2026');
  });

  // Devolver "Invalid Date" en la tabla de pedidos es peor que devolver el
  // input crudo: al menos el crudo se puede leer.
  it('devuelve el input tal cual si no parsea', () => {
    expect(formatFecha('no-es-fecha')).toBe('no-es-fecha');
    expect(formatFecha('')).toBe('');
  });
});

describe('anchoBarra', () => {
  it('escala respecto al máximo', () => {
    expect(anchoBarra(50, 100)).toBe(50);
    expect(anchoBarra(100, 100)).toBe(100);
    expect(anchoBarra(0, 100)).toBe(0);
    expect(anchoBarra(25, 50)).toBe(50);
  });

  // Sin el acotado superior, un valor por encima del máximo produce un
  // width CSS inválido y la barra desaparece.
  it('acota a 100 cuando el valor supera el máximo', () => {
    expect(anchoBarra(150, 100)).toBe(100);
  });

  it('nunca devuelve negativo', () => {
    expect(anchoBarra(-10, 100)).toBe(0);
  });

  // La、最大=0 hace que cualquier valor dé Infinity o NaN en el width.
  it('devuelve 0 con máximo 0 o negativo, en vez de dividir por cero', () => {
    expect(anchoBarra(10, 0)).toBe(0);
    expect(anchoBarra(10, -5)).toBe(0);
  });

  it('devuelve 0 con valores no finitos', () => {
    expect(anchoBarra(NaN, 100)).toBe(0);
    expect(anchoBarra(Infinity, 100)).toBe(0);
    expect(anchoBarra(10, NaN)).toBe(0);
  });

  // Es el caso real de una lista vacía: datos = [], máximo = 0.
  it('aguanta la lista vacía', () => {
    const filas: number[] = [];
    const max = filas.reduce((m, v) => Math.max(m, v), 0);
    expect(anchoBarra(10, max)).toBe(0);
  });
});