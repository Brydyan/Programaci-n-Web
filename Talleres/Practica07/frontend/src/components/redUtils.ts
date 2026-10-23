// Utilidades del árbol de red, separadas de RedArbol.tsx.
//
// El componente solo exporta componentes; si mezcla funciones, el
// eslint-plugin-react-refresh avisa que el Fast Refresh de Vite no funciona
// sobre ese archivo. Los helpers se prueban sin montar nada.

import { useState } from 'react';
import type { NodoRed } from '../types/api';

export type ResumenRed = { nodos: number; ventas: number; nivelMaximo: number };

/**
 * Suma una rama completa de forma recursiva.
 *
 * `nivelMaximo` se lleva aparte porque una sola pasada no puede saber cuál
 * fue la profundidad más profunda: se toma el máximo entre la profundidad
 * actual y la de cada hijo.
 */
export const resumenRed = (nodos: NodoRed[]): ResumenRed =>
  nodos.reduce<ResumenRed>(
    (acc, n) => {
      const hijos = resumenRed(n.hijos ?? []);
      return {
        nodos: acc.nodos + 1 + hijos.nodos,
        ventas: acc.ventas + (n.ventas ?? 0) + hijos.ventas,
        nivelMaximo: Math.max(acc.nivelMaximo, n.nivel, hijos.nivelMaximo),
      };
    },
    { nodos: 0, ventas: 0, nivelMaximo: 0 },
  );

/**
 * ¿Este nodo, o alguno de sus descendientes, coincide con la búsqueda?
 *
 * Normaliza la búsqueda aquí y no confiar en quien llame: se exporta aparte,
 * y buscar "ADMIN" contra "admin" devolvía `false` sin ningún error visible
 * cuando la normalización vivía en filtrarArbol.
 */
export const coincide = (n: NodoRed, busqueda: string): boolean => {
  const q = busqueda.trim().toLowerCase();
  if (!q) return true; // Sin búsqueda, todo coincide.
  return (
    n.nombre.toLowerCase().includes(q) ||
    n.codigo.toLowerCase().includes(q) ||
    (n.hijos ?? []).some((h) => coincide(h, q))
  );
};

/**
 * Filtra el árbol por nombre o código, conservando la rama entera de
 * cualquier nodo que coincide: si sólo se dejaran las coincidencias
 * aisladas, un hijo que casa bien podría aparecer sin el padre que lo
 * justifica.
 *
 * Es filtrado de cliente a propósito: el árbol completo cabe en una
 * respuesta y llega entero, así que no tiene sentido pedirlo otra vez por
 * cada tecla.
 */
export const filtrarArbol = (arbol: NodoRed[], texto: string): NodoRed[] => {
  const q = texto.trim();
  if (!q) return arbol;
  return arbol.filter((n) => coincide(n, q));
};

/** Estado del buscador del árbol. */
export const useFiltroArbol = (arbol: NodoRed[]) => {
  const [texto, setTexto] = useState('');
  return { texto, setTexto, filtrado: filtrarArbol(arbol, texto) };
};