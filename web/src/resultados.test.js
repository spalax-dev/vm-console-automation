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

    const cuerpo = panel.querySelector('#result-body');

    expect(cuerpo.textContent).toContain('Se instaló nano');
    expect(panel.querySelector('#result-cuenta').textContent).toBe('(1)');

    // El historial no agranda la tarjeta: se desplaza dentro del panel,
    // con la barra de desplazamiento del mismo tono oscuro.
    expect(cuerpo.className).toContain('overflow-auto');
    expect(cuerpo.className).toContain('min-h-0');
    expect(cuerpo.className).toContain('scroll-oscuro');
    expect(panel.className).toContain('overflow-hidden');
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

  it('abre el detalle técnico solo en la última operación', () => {
    agregar({ ...EJEMPLO, mensaje: 'Primera' });
    agregar({ ...EJEMPLO, mensaje: 'Segunda' });

    const panel = ResultPanel();
    document.body.appendChild(panel);

    const detalles = [...panel.querySelectorAll('details')];

    expect(detalles).toHaveLength(2);
    expect(detalles[0].hasAttribute('open')).toBe(false);
    expect(detalles[1].hasAttribute('open')).toBe(true);
  });

  it('describe la orden cuando la operación no trae mensaje', () => {
    const panel = ResultPanel();
    document.body.appendChild(panel);

    showResult({ ok: true, operacion: 'apt-get remove -y nano' });

    const texto = panel.querySelector('#result-body').textContent;

    expect(texto).toContain('apt-get remove -y nano');
    expect(texto).not.toContain('Operación exitosa');
    expect(leer()[0].mensaje).toContain('apt-get remove -y nano');
  });

  it('pinta los iconos de la cabecera y el botón de limpiar', () => {
    const panel = ResultPanel();
    document.body.appendChild(panel);

    expect(panel.querySelector('header svg')).not.toBeNull();
    expect(panel.querySelector('#result-limpiar svg')).not.toBeNull();

    // El botón forma parte del panel oscuro, no del estilo claro general.
    expect(panel.querySelector('#result-limpiar').classList.contains('btn')).toBe(false);
  });

  it('no encoge las fichas cuando el historial crece', () => {
    agregar({ ...EJEMPLO, mensaje: 'Primera' });
    agregar({ ...EJEMPLO, mensaje: 'Segunda' });

    const panel = ResultPanel();
    document.body.appendChild(panel);

    for (const ficha of panel.querySelectorAll('#result-body > article')) {
      expect(ficha.classList.contains('shrink-0')).toBe(true);
    }
  });

  it('se desplaza siempre hasta la última operación', () => {
    agregar({ ...EJEMPLO, mensaje: 'Primera' });

    const panel = ResultPanel();
    document.body.appendChild(panel);

    const cuerpo = panel.querySelector('#result-body');

    Object.defineProperty(cuerpo, 'scrollHeight', { value: 640, configurable: true });

    showResult({ ...EJEMPLO, mensaje: 'Segunda' });

    expect(cuerpo.scrollTop).toBe(640);
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

describe('detalle técnico del resultado', () => {
  it('omits empty output blocks and explains when there is no output', () => {
    const panel = ResultPanel();
    document.body.appendChild(panel);

    showResult({ ok: true, mensaje: 'Hecho', operacion: 'systemctl start demo.service', codigo: 0 });

    const cuerpo = panel.querySelector('#result-body');

    expect(cuerpo.textContent).toContain('La orden no produjo salida.');
    expect(cuerpo.textContent).not.toContain('stderr:');
  });

  it('shows stdout and stderr when the command produces them', () => {
    const panel = ResultPanel();
    document.body.appendChild(panel);

    showResult({
      ok: true,
      mensaje: 'Hecho',
      operacion: 'systemctl status demo.service',
      codigo: 0,
      salida: 'active (running)',
      stderr: 'warning',
    });

    const cuerpo = panel.querySelector('#result-body');

    expect(cuerpo.textContent).toContain('active (running)');
    expect(cuerpo.textContent).toContain('stderr:');
    expect(cuerpo.textContent).toContain('warning');
  });
});

describe('formato de la salida de systemd', () => {
  it('highlights labels and states and escapes the output', () => {
    const salida = [
      '● getty-static.service - getty on tty2-tty6 if dbus and logind are not available',
      '     Loaded: loaded (/usr/lib/systemd/system/getty-static.service; static)',
      '     Active: inactive (dead)',
      '  Condition: start condition unmet at Mon 2026-10-26 18:22:10 -05; 30min ago',
      '',
      'Oct 26 18:22:10 base systemd[1]: getty-static.service skipped <script>alert(1)</script>',
    ].join('\n');

    const panel = ResultPanel();
    document.body.appendChild(panel);

    showResult({
      ok: true,
      mensaje: 'Estado de la unidad.',
      operacion: 'systemctl status getty-static.service',
      codigo: 0,
      salida,
    });

    const bloque = panel.querySelector('.salida-systemctl');

    expect(bloque).not.toBeNull();
    expect(bloque.querySelectorAll('span').length).toBeGreaterThan(3);
    expect(bloque.textContent).toContain('Active: inactive (dead)');
    expect(bloque.textContent).toContain('skipped');
    expect(bloque.querySelector('script')).toBeNull();
    expect(panel.innerHTML).not.toContain('<script>');
  });

  it('keeps ordinary output in a plain preformatted block', () => {
    const panel = ResultPanel();
    document.body.appendChild(panel);

    showResult({
      ok: true,
      mensaje: 'Hecho',
      operacion: 'echo hola',
      codigo: 0,
      salida: 'hola',
    });

    expect(panel.querySelector('.salida-systemctl')).toBeNull();
    expect(panel.querySelector('pre').textContent).toBe('hola');
  });
});
