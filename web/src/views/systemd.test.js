// @vitest-environment happy-dom

import { afterEach, describe, expect, it, vi } from 'vitest';

import { ResultPanel } from '../components/ResultPanel.js';
import { renderSystemd } from './systemd.js';

/**
 * Builds a JSON response for the fetch stub.
 *
 * @param {object} body Response body.
 * @param {number} [status] HTTP status.
 * @returns {object} Minimal Response-like object.
 */
function jsonResponse(body, status = 200) {
  return {
    ok: status >= 200 && status < 300,
    status,
    headers: { get: (name) => (name.toLowerCase() === 'content-type' ? 'application/json' : '') },
    json: async () => body,
    text: async () => JSON.stringify(body),
    blob: async () => new Blob(['x']),
  };
}

/**
 * Mounts the systemd view with its result panel.
 *
 * @returns {{view: HTMLElement, result: HTMLElement}} Mounted elements.
 */
function mountView() {
  document.body.innerHTML = '<div id="app"></div>';

  const view = renderSystemd();
  document.body.appendChild(view);

  const result = ResultPanel();
  result.id = 'resultado-global';
  view.querySelector('#result-slot').appendChild(result);

  return { view, result };
}

afterEach(() => {
  window.localStorage.clear();
  vi.unstubAllGlobals();
  document.body.innerHTML = '';
});

describe('systemd unit control', () => {
  it('selects the clicked table row in the service control', async () => {
    vi.stubGlobal('fetch', vi.fn(async (url) => {
      if (url === '/api/units') {
        return jsonResponse({
          unidades: [
            { unidad: 'anacron.service', activo: true, habilitado: true, estado: 'active', detalle: 'running' },
            { unidad: 'cron.service', activo: true, habilitado: true, estado: 'active', detalle: 'running' },
          ],
        });
      }

      if (url === '/api/units/cron.service') {
        return jsonResponse({ unidad: 'cron.service', activo: true, habilitado: true, estado: 'active', detalle: 'running' });
      }

      return jsonResponse({});
    }));

    const { view } = mountView();

    await vi.waitFor(() => {
      expect(view.querySelectorAll('tr[data-unidad]')).toHaveLength(2);
    });

    const select = view.querySelector('#unidad-seleccion');
    const row = view.querySelector('tr[data-unidad="cron.service"]');

    row.click();

    expect(select.value).toBe('cron.service');

    await vi.waitFor(() => {
      expect(view.querySelector('#unidad-estado').textContent).toBe('activa');
    });
  });

  it('shows the real systemctl status output', async () => {
    const statusOutput = '● anacron.service - Anacron\n     Active: active (running)\n';

    vi.stubGlobal('fetch', vi.fn(async (url) => {
      if (url === '/api/units') {
        return jsonResponse({
          unidades: [
            { unidad: 'anacron.service', activo: true, habilitado: true, estado: 'active', detalle: 'running' },
          ],
        });
      }

      if (url === '/api/units/anacron.service') {
        return jsonResponse({ unidad: 'anacron.service', activo: true, habilitado: true, estado: 'active', detalle: 'running' });
      }

      if (url === '/api/units/anacron.service/status') {
        return jsonResponse({
          ok: true,
          mensaje: 'Estado de la unidad «anacron.service».',
          operacion: 'systemctl status anacron.service --no-pager --full',
          codigo: 0,
          salida: statusOutput,
          stderr: '',
        });
      }

      return jsonResponse({});
    }));

    const { view, result } = mountView();

    await vi.waitFor(() => {
      expect(view.querySelectorAll('tr[data-unidad]')).toHaveLength(1);
    });

    view.querySelector('[data-accion="estado"]').click();

    await vi.waitFor(() => {
      expect(result.textContent).toContain('Active: active (running)');
    });

    expect(result.textContent).toContain('systemctl status anacron.service');
    expect(result.textContent).not.toContain('systemctl estado');
  });
});

describe('systemd control results', () => {
  it('shows the status output after a control action and uses the real verb', async () => {
    const statusOutput = '● anacron.service - Anacron\n     Active: active (running)\n';
    const calls = [];

    vi.stubGlobal('fetch', vi.fn(async (url, options = {}) => {
      const method = options.method ?? 'GET';
      calls.push(`${method} ${url}`);

      if (url === '/api/units') {
        return jsonResponse({
          unidades: [
            { unidad: 'anacron.service', activo: true, habilitado: true, estado: 'active', detalle: 'running' },
          ],
        });
      }

      if (url === '/api/units/anacron.service' && method === 'PATCH') {
        return jsonResponse({ unidad: 'anacron.service', activo: true, habilitado: true, estado: 'active', detalle: 'running' });
      }

      if (url === '/api/units/anacron.service/status') {
        return jsonResponse({
          ok: true,
          mensaje: 'Estado de la unidad «anacron.service».',
          operacion: 'systemctl status anacron.service --no-pager --full',
          codigo: 0,
          salida: statusOutput,
          stderr: '',
        });
      }

      return jsonResponse({});
    }));

    const { view, result } = mountView();

    await vi.waitFor(() => {
      expect(view.querySelectorAll('tr[data-unidad]')).toHaveLength(1);
    });

    view.querySelector('[data-accion="iniciar"]').click();

    await vi.waitFor(() => {
      expect(result.textContent).toContain('Active: active (running)');
    });

    expect(calls).toContain('PATCH /api/units/anacron.service');
    expect(result.textContent).toContain('systemctl start anacron.service');
    expect(result.textContent).not.toContain('systemctl iniciar');
  });
});
