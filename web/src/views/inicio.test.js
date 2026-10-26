// @vitest-environment happy-dom

import { afterEach, describe, expect, it, vi } from 'vitest';

import { renderInicio } from './inicio.js';

/**
 * Builds a JSON response for the fetch stub.
 *
 * @param {object} body Response body.
 * @returns {object} Minimal Response-like object.
 */
function jsonResponse(body) {
  return {
    ok: true,
    status: 200,
    headers: { get: (name) => (name.toLowerCase() === 'content-type' ? 'application/json' : '') },
    json: async () => body,
    text: async () => JSON.stringify(body),
    blob: async () => new Blob(['x']),
  };
}

afterEach(() => {
  vi.unstubAllGlobals();
  document.body.innerHTML = '';
});

describe('home view registration', () => {
  it('shows the register button and opens the dialog only on demand', async () => {
    const fetchMock = vi.fn(async (url) => {
      if (url === '/api/session') {
        return jsonResponse({
          vm: 'debian-1',
          conector: 'ssh',
          ip: '192.168.1.50',
          estado: 'running',
          registrada: false,
        });
      }

      if (url === '/api/vms') {
        return jsonResponse({
          maquinas: [{
            nombre: 'debian-1',
            uuid: 'uuid-1',
            estado: 'running',
            activa: true,
            registrada: false,
          }],
        });
      }

      return jsonResponse({});
    });

    vi.stubGlobal('fetch', fetchMock);

    document.body.innerHTML = '<div id="app"></div><div id="result-slot"></div>';

    const view = renderInicio();
    document.body.appendChild(view);

    await vi.waitFor(() => {
      expect(view.querySelector('#registrar-sesion')).not.toBeNull();
    });

    expect(document.querySelector('#registro-form')).toBeNull();

    view.querySelector('#registrar-sesion').click();

    await vi.waitFor(() => {
      expect(document.querySelector('#registro-form')).not.toBeNull();
    });

    expect(document.querySelector('#registro-diagnostico')).toBeNull();
  });
});
