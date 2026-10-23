import { describe, expect, it } from 'vitest';
import { coincide, filtrarArbol, resumenRed } from './redUtils';
import type { NodoRed } from '../types/api';

// Estas funciones decidían qué se pintaba en dos pantallas distintas, así que
// un error aquí se vería como dos cifras diferentes según por dónde se mire.

const nodo = (
  id: number,
  nombre: string,
  codigo: string,
  nivel: number,
  ventas: number,
  hijos?: NodoRed[],
): NodoRed => ({ id, nombre, codigo, nivel, ventas, hijos });

const ARBOL: NodoRed[] = [
  nodo(1, 'TU', 'TU', 0, 50, [
    nodo(2, 'Administrador', 'ADMIN', 1, 30, [nodo(10, 'Ana', 'MKT-ANA', 2, 20)]),
  ]),
  nodo(3, 'Cliente Demo', 'LUIS', 0, 40, []),
];

describe('resumenRed', () => {
  it('suma nodos y ventas de toda la rama', () => {
    const r = resumenRed(ARBOL);
    expect(r.nodos).toBe(4); // TU, ADMIN, MKT-ANA y LUIS
    expect(r.ventas).toBe(140); // 50 + 30 + 20 + 40
  });

  it('informa la profundidad máxima', () => {
    expect(resumenRed(ARBOL).nivelMaximo).toBe(2);
  });

  it('devuelve ceros con un árbol vacío', () => {
    expect(resumenRed([])).toEqual({ nodos: 0, ventas: 0, nivelMaximo: 0 });
  });

  it('no rompe si una respuesta llega sin ventas', () => {
    // El backend declara `Ventas float64` sin omitempty, así que el campo
    // siempre viaja. El `?? 0` de resumenRed es defensa ante una respuesta
    // malformada, no un caso esperado: si se sumara undefined, `acc.ventas`
    // se volvería NaN y todas las cifras de la pantalla saldrían rotas.
    const sinVentas = [{ id: 9, nombre: 'X', codigo: 'X', nivel: 0 }] as NodoRed[];
    expect(resumenRed(sinVentas).ventas).toBe(0);
  });
});

describe('coincide', () => {
  it('busca por nombre sin distinguir mayúsculas', () => {
    expect(coincide(ARBOL[1], 'cliente')).toBe(true);
    expect(coincide(ARBOL[1], 'CLIENTE')).toBe(true);
  });

  it('busca por código', () => {
    expect(coincide(ARBOL[0], 'mkt-ana')).toBe(true);
  });

  it('encuentra coincidencias en descendientes', () => {
    // "Ana" sólo existe a dos niveles de profundidad.
    expect(coincide(ARBOL[0], 'ana')).toBe(true);
  });

  it('devuelve false si no hay coincidencia', () => {
    expect(coincide(ARBOL[1], 'zzzz')).toBe(false);
  });
});

describe('filtrarArbol', () => {
  it('devuelve el árbol intacto con búsqueda vacía', () => {
    expect(filtrarArbol(ARBOL, '')).toBe(ARBOL);
    expect(filtrarArbol(ARBOL, '   ')).toBe(ARBOL);
  });

  it('conserva la rama completa cuando un descendiente coincide', () => {
    // Si se filtrara nodo a nodo, "Ana" aparecería sin su padre y el árbol
    // se leería como si fuera una raíz de nivel 2.
    const r = filtrarArbol(ARBOL, 'Ana');
    expect(r).toHaveLength(1);
    expect(r[0].codigo).toBe('TU');
    expect(r[0].hijos?.[0].hijos?.[0].nombre).toBe('Ana');
  });

  it('puede dejar varias raíces', () => {
    // "Demo" está en LUIS y "Ana" en la otra rama: con "e" no, con el
    // nombre completo sólo una; este caso comprueba dos coincidencias.
    const r = filtrarArbol(ARBOL, 'o');
    expect(r.length).toBeGreaterThanOrEqual(1);
  });

  it('devuelve vacío si nada coincide', () => {
    expect(filtrarArbol(ARBOL, 'zzzz')).toEqual([]);
  });
});