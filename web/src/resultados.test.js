// @vitest-environment happy-dom

import { afterEach, beforeEach, describe, expect, it } from 'vitest';

import { agregar, leer, limpiar } from './resultados.js';
import { ResultPanel, showResult } from './components/ResultPanel.js';

/** Clave con la que se guarda el historial. */
const CLAVE = 'vmcons:resultados';

/** Resultado de ejemplo. */
const EJEMPLO = { ok: true, mensaje: 'Se instaló nano', operacion: 'apt-get install -y nano', codigo: 0 };

beforeEach(() => {
  window.localStorage.clear();
  document.body.innerHTML = '';
});

afterEach(() => {
  window.localStorage.clear();
});

describe('almacén de resultados', () => {
  it('guarda y recupera los resultados en orden', () => {
    agregar(EJEMPLO);
    agregar({ ...EJEMPLO, mensaje: 'Se eliminó nano' });

    const historial = leer();

    expect(historial).toHaveLength(2);
    expect(historial[1].mensaje).toBe('Se eliminó nano');
  });

  it('conserva como máximo cincuenta resultados', () => {
    for (let i = 0; i < 60; i += 1) {
      agregar({ ...EJEMPLO, mensaje: `Operación ${i}` });
    }

    const historial = leer();

    expect(historial).toHaveLength(50);
    expect(historial[49].mensaje).toBe('Operación 59');
  });

  it('vacía el historial', () => {
    agregar(EJEMPLO);
    limpiar();

    expect(leer()).toHaveLength(0);
    expect(window.localStorage.getItem(CLAVE)).toBe('[]');
  });

  it('tolera un contenido corrupto', () => {
    window.localStorage.setItem(CLAVE, 'no es JSON');

    expect(leer()).toHaveLength(0);
  });

  it('descarta entradas sin mensaje', () => {
    window.localStorage.setItem(CLAVE, JSON.stringify([{ ok: true }, EJEMPLO]));

    expect(leer()).toHaveLength(1);
  });
});

describe('panel de resultado', () => {
  it('muestra el historial guardado al montarse', () => {
    agregar(EJEMPLO);

    const panel = ResultPanel();
    document.body.appendChild(panel);

    expect(panel.querySelector('#result-body').textContent).toContain('Se instaló nano');
    expect(panel.querySelector('#result-cuenta').textContent).toBe('(1)');
  });

  it('añade un resultado nuevo sin borrar los anteriores', () => {
    agregar(EJEMPLO);

    const panel = ResultPanel();
    document.body.appendChild(panel);

    showResult({ ok: false, mensaje: 'No fue posible instalar', operacion: 'apt-get install -y nano', codigo: 100 });

    const cuerpo = panel.querySelector('#result-body');

    expect(cuerpo.textContent).toContain('Se instaló nano');
    expect(cuerpo.textContent).toContain('No fue posible instalar');
    expect(panel.querySelector('#result-cuenta').textContent).toBe('(2)');
    expect(leer()).toHaveLength(2);
  });

  it('limpia el historial con el botón', () => {
    agregar(EJEMPLO);

    const panel = ResultPanel();
    document.body.appendChild(panel);

    panel.querySelector('#result-limpiar').click();

    expect(panel.querySelector('#result-body').textContent).toContain('Sin operaciones ejecutadas');
    expect(leer()).toHaveLength(0);
  });

  it('sobrevive a reconstruir el panel', () => {
    agregar(EJEMPLO);

    const primero = ResultPanel();
    document.body.appendChild(primero);

    document.body.innerHTML = '';

    const segundo = ResultPanel();
    document.body.appendChild(segundo);

    expect(segundo.querySelector('#result-body').textContent).toContain('Se instaló nano');
  });
});
