import { describe, expect, it } from 'vitest';
import { readFileSync, readdirSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';
import { ESTADO_COMISION, ESTADO_LIQUIDACION, ESTADO_PEDIDO } from './tokens';
import type { EstadoComision, EstadoLiquidacion } from '../types/api';
import type { EstadoPedido } from '../services/pedidosService';

// Estos tokens son la razón de que el tema siga siendo coherente, y nada en
// el type-checker avisa si se rompen: `bg-estado-exito-bg` es una clase de
// Tailwind que solo existe si el @theme la declara, y si no existe el
// elemento se queda sin fondo sin ningún error en consola.
//
// Estos tests leen el archivo de verdad, no una copia.

const raizCss = resolve(
  dirname(fileURLToPath(import.meta.url)),
  '..',
  'index.css',
);
const css = readFileSync(raizCss, 'utf8');

describe('tokens de index.css', () => {
  it('declara cada color de estado', () => {
    const requeridos = [
      'estado-neutro-bg', 'estado-neutro-fg',
      'estado-aviso-bg', 'estado-aviso-fg',
      'estado-info-bg', 'estado-info-fg',
      'estado-exito-bg', 'estado-exito-fg',
      'estado-error-bg', 'estado-error-fg',
    ];
    for (const t of requeridos) {
      expect(css, `falta --color-${t}`).toContain(`--color-${t}:`);
    }
  });

  it('declara los tonos de marca que usa el proyecto', () => {
    for (const tono of [50, 100, 200, 300, 400, 500, 600, 700]) {
      expect(css, `falta --color-marca-${tono}`).toContain(`--color-marca-${tono}:`);
    }
  });

  it('no queda ninguna clase indigo-* en los componentes', () => {
    // El acento del proyecto se llama marca. Un indigo-* pegado de otro
    // proyecto seguiría funcionando (Tailwind lo trae de serie) pero
    // escaparía del token: cambiar el color de acento no lo tocaría, y
    // volvería a haber dos verdades.
    const dir = dirname(fileURLToPath(import.meta.url));
    const offenders: string[] = [];
    for (const archivo of readdirSync(dir)) {
      if (!archivo.endsWith('.tsx')) continue;
      const contenido = readFileSync(resolve(dir, archivo), 'utf8');
      if (/\bindigo-\d/.test(contenido)) offenders.push(archivo);
    }
    expect(offenders, `clases indigo-* sin tokenizar en: ${offenders.join(', ')}`)
      .toEqual([]);
  });

  it('declara el radio de control que usa el anillo de foco', () => {
    expect(css).toContain('--radius-control:');
  });
});

describe('semantica de estados', () => {
  it('cubre todos los estados de comision', () => {
    const estados: EstadoComision[] = ['pendiente', 'aprobada', 'pagada', 'revertida'];
    for (const e of estados) expect(ESTADO_COMISION[e]).toBeDefined();
  });

  it('cubre todos los estados de pedido', () => {
    const estados: EstadoPedido[] = [
      'pendiente', 'preparando', 'enviado', 'entregado', 'cancelado',
    ];
    for (const e of estados) expect(ESTADO_PEDIDO[e]).toBeDefined();
  });

  it('cubre los dos estados de liquidacion', () => {
    const estados: EstadoLiquidacion[] = ['pendiente', 'pagada'];
    for (const e of estados) expect(ESTADO_LIQUIDACION[e]).toBeDefined();
  });

  it('pinta revertida como neutro, no como error', () => {
    // Una comision revertida es un pedido cancelado con el dinero devuelto:
    // no es un fallo del sistema, asi que no debe salir en rojo.
    expect(ESTADO_COMISION.revertida.tono).toBe('neutro');
    expect(ESTADO_PEDIDO.cancelado.tono).toBe('error');
  });

  it('distingue pendiente de pagado en los tres dominios', () => {
    expect(ESTADO_COMISION.pendiente.tono).toBe('aviso');
    expect(ESTADO_PEDIDO.pendiente.tono).toBe('neutro');
    expect(ESTADO_LIQUIDACION.pendiente.tono).toBe('aviso');
    // El mismo nombre puede tener distinto peso según el dominio, asi que
    // el texto nunca debe suponerse a partir de la clave.
    expect(ESTADO_COMISION.pagada.texto).toBe('Pagada');
  });
});