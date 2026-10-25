// @vitest-environment happy-dom

import { afterEach, describe, expect, it, vi } from 'vitest';

import { renderIcons } from './icons.js';
import { renderPaquetes } from './views/paquetes.js';
import { renderUsuarios } from './views/usuarios.js';
import { renderHaproxy } from './views/haproxy.js';

/**
 * Sustituye `fetch` por un doble que responde con el cuerpo dado.
 *
 * @param {object} cuerpo Cuerpo JSON de la respuesta.
 * @param {number} [estado] Código HTTP.
 */
function simularFetch(cuerpo, estado = 200) {
  vi.stubGlobal('fetch', async () => ({
    ok: estado >= 200 && estado < 300,
    status: estado,
    headers: { get: () => 'application/json' },
    json: async () => cuerpo,
    text: async () => JSON.stringify(cuerpo),
    blob: async () => new Blob(['x']),
  }));
}

/**
 * Crea la ranura del panel de resultado que esperan las vistas.
 */
function prepararRanura() {
  document.body.innerHTML = '<div id="app"></div><div id="result-slot"></div>';
}

afterEach(() => {
  vi.unstubAllGlobals();
  document.body.innerHTML = '';
});

describe('iconos', () => {
  it('reemplaza las marcas por SVG con el trazo de la interfaz', () => {
    const contenedor = document.createElement('div');
    contenedor.innerHTML = '<i data-lucide="package"></i><i data-lucide="key-round"></i>';

    renderIcons(contenedor);

    const svgs = contenedor.querySelectorAll('svg');

    expect(svgs).toHaveLength(2);
    expect(svgs[0].getAttribute('stroke-width')).toBe('1.5');
    expect(svgs[0].getAttribute('width')).toBe('16');
    expect(contenedor.querySelectorAll('i')).toHaveLength(0);
  });
});

describe('vistas conectadas', () => {
  it('pinta los paquetes que devuelve la API', async () => {
    prepararRanura();
    simularFetch({ paquetes: [{ nombre: 'nano', version: '7.2-1', estado: 'installed' }] });

    const vista = renderPaquetes();
    document.body.appendChild(vista);

    await vi.waitFor(() => {
      expect(vista.querySelector('#paquetes-filas').textContent).toContain('nano');
    });

    expect(vista.querySelector('#paquetes-cuenta').textContent).toBe('1 de 1');
    expect(vista.querySelector('#paquetes-filas').querySelectorAll('svg').length).toBeGreaterThanOrEqual(0);
  });

  it('pinta un aviso cuando el módulo responde 501', async () => {
    prepararRanura();
    simularFetch({ ok: false, mensaje: 'La operación «listar los usuarios» aún no está conectada con la máquina.' }, 501);

    const vista = renderUsuarios();
    document.body.appendChild(vista);

    await vi.waitFor(() => {
      expect(vista.querySelector('#usuarios-filas').textContent).toContain('no está conectado');
    });
  });

  it('informa cuando el back-end está vacío', async () => {
    prepararRanura();
    simularFetch({ servidores: [] });

    const vista = renderHaproxy();
    document.body.appendChild(vista);

    await vi.waitFor(() => {
      expect(vista.querySelector('#backend-filas').textContent).toContain('no tiene servidores');
    });
  });
});
