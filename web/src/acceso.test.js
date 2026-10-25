// @vitest-environment happy-dom

import { afterEach, describe, expect, it, vi } from 'vitest';

import { renderAcceso } from './views/acceso.js';

/**
 * Sustituye `fetch` por un doble que registra las llamadas.
 *
 * @param {object} cuerpo Cuerpo JSON de la respuesta.
 * @param {number} [estado] Código HTTP.
 * @returns {Array<{url: string, opciones: object}>} Llamadas registradas.
 */
function simularFetch(cuerpo, estado = 200) {
  const llamadas = [];

  vi.stubGlobal('fetch', async (url, opciones) => {
    llamadas.push({ url, opciones });

    return {
      ok: estado >= 200 && estado < 300,
      status: estado,
      headers: { get: (nombre) => (nombre === 'content-type' ? 'application/json' : '') },
      json: async () => cuerpo,
      text: async () => JSON.stringify(cuerpo),
    };
  });

  return llamadas;
}

/**
 * Rellena un campo del formulario.
 *
 * @param {HTMLElement} pantalla Pantalla de acceso.
 * @param {string} id Identificador del campo.
 * @param {string} valor Valor a escribir.
 */
function escribir(pantalla, id, valor) {
  const campo = pantalla.querySelector(`#${id}`);
  campo.value = valor;
}

afterEach(() => {
  vi.unstubAllGlobals();
  document.body.innerHTML = '';
});

describe('pantalla de acceso', () => {
  it('crea la cuenta cuando no existe y avisa al entrar', async () => {
    const llamadas = simularFetch({ configurada: true, autenticado: true, usuario: 'jorge' });
    const alEntrar = vi.fn();

    const pantalla = renderAcceso({ configurada: false, alEntrar });
    document.body.appendChild(pantalla);

    escribir(pantalla, 'acceso-usuario', 'jorge');
    escribir(pantalla, 'acceso-password', 'una-clave-larga');
    escribir(pantalla, 'acceso-repetir', 'una-clave-larga');

    pantalla.querySelector('#acceso-form').dispatchEvent(
      new Event('submit', { bubbles: true, cancelable: true }),
    );

    await vi.waitFor(() => {
      expect(alEntrar).toHaveBeenCalled();
    });

    expect(llamadas[0].url).toBe('/api/auth/setup');
    expect(llamadas[0].opciones.method).toBe('POST');
    expect(JSON.parse(llamadas[0].opciones.body)).toEqual({
      usuario: 'jorge',
      password: 'una-clave-larga',
    });
  });

  it('no envía el formulario si el usuario no cumple las reglas', () => {
    const llamadas = simularFetch({});
    const pantalla = renderAcceso({ configurada: true });
    document.body.appendChild(pantalla);

    escribir(pantalla, 'acceso-usuario', 'Jorge');
    escribir(pantalla, 'acceso-password', 'una-clave-larga');

    pantalla.querySelector('#acceso-form').dispatchEvent(
      new Event('submit', { bubbles: true, cancelable: true }),
    );

    expect(llamadas).toHaveLength(0);
    expect(pantalla.querySelector('#acceso-aviso').textContent).toContain('usuario');
  });

  it('exige que las contraseñas coincidan al crear la cuenta', () => {
    const llamadas = simularFetch({});
    const pantalla = renderAcceso({ configurada: false });
    document.body.appendChild(pantalla);

    escribir(pantalla, 'acceso-usuario', 'jorge');
    escribir(pantalla, 'acceso-password', 'una-clave-larga');
    escribir(pantalla, 'acceso-repetir', 'otra-clave-larga');

    pantalla.querySelector('#acceso-form').dispatchEvent(
      new Event('submit', { bubbles: true, cancelable: true }),
    );

    expect(llamadas).toHaveLength(0);
    expect(pantalla.querySelector('#acceso-aviso').textContent).toContain('no coinciden');
  });

  it('muestra el mensaje del servidor cuando las credenciales fallan', async () => {
    simularFetch({ ok: false, mensaje: 'Usuario o contraseña incorrectos.' }, 401);

    const pantalla = renderAcceso({ configurada: true });
    document.body.appendChild(pantalla);

    escribir(pantalla, 'acceso-usuario', 'jorge');
    escribir(pantalla, 'acceso-password', 'clave-equivocada');

    pantalla.querySelector('#acceso-form').dispatchEvent(
      new Event('submit', { bubbles: true, cancelable: true }),
    );

    await vi.waitFor(() => {
      expect(pantalla.querySelector('#acceso-aviso').textContent).toContain(
        'Usuario o contraseña incorrectos.',
      );
    });
  });
});
