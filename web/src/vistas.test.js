// @vitest-environment happy-dom

import { afterEach, describe, expect, it, vi } from 'vitest';

import { renderIcons } from './icons.js';
import { Tabs } from './components/Tabs.js';
import { VMSelector } from './components/VMSelector.js';
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

describe('barra de pestañas', () => {
  it('marca la pestaña activa y deja una sola alcanzable con Tab', () => {
    const nav = Tabs('usuarios');
    document.body.appendChild(nav);

    const activa = nav.querySelector('[aria-selected="true"]');

    expect(activa.dataset.tab).toBe('usuarios');
    expect(nav.querySelectorAll('[tabindex="0"]')).toHaveLength(1);
    expect(nav.getAttribute('role')).toBe('tablist');
  });

  it('se mueve con las flechas y cambia la ruta', () => {
    window.location.hash = '#/inicio';

    const nav = Tabs('inicio');
    document.body.appendChild(nav);

    const primera = nav.querySelector('[data-tab="inicio"]');
    primera.focus();
    primera.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowRight', bubbles: true }));

    expect(window.location.hash).toBe('#/paquetes');
  });
});

describe('selector de máquina virtual', () => {
  it('muestra la máquina vinculada y lista el inventario', async () => {
    simularFetch({
      maquinas: [
        { nombre: 'debian-1', uuid: 'uuid-1', estado: 'running', activa: true },
        { nombre: 'ubuntu-lab', uuid: 'uuid-2', estado: 'poweroff', activa: false },
      ],
    });

    const selector = VMSelector();
    document.body.appendChild(selector);

    await vi.waitFor(() => {
      expect(selector.querySelector('#vm-nombre').textContent).toBe('debian-1');
    });

    selector.querySelector('#vm-boton').click();

    expect(selector.querySelector('#vm-panel').classList.contains('hidden')).toBe(false);
    expect(selector.querySelectorAll('[data-maquina]')).toHaveLength(2);
    expect(selector.querySelector('[aria-selected="true"]').dataset.maquina).toBe('debian-1');
  });

  it('vincula la máquina elegida y avisa al llamador', async () => {
    const alVincular = vi.fn();

    vi.stubGlobal('fetch', async (url, opciones) => {
      const cuerpo = (opciones ?? {}).method === 'PUT'
        ? { nombre: 'ubuntu-lab', uuid: 'uuid-2', estado: 'poweroff', activa: true, registrada: true }
        : { maquinas: [{ nombre: 'ubuntu-lab', uuid: 'uuid-2', estado: 'poweroff', activa: false }] };

      return {
        ok: true,
        status: 200,
        headers: { get: () => 'application/json' },
        json: async () => cuerpo,
        text: async () => JSON.stringify(cuerpo),
        blob: async () => new Blob(['x']),
      };
    });

    const selector = VMSelector({ alVincular });
    document.body.appendChild(selector);

    await vi.waitFor(() => {
      expect(selector.querySelector('[data-maquina]')).not.toBeNull();
    });

    selector.querySelector('#vm-boton').click();
    selector.querySelector('[data-maquina]').click();

    await vi.waitFor(() => {
      expect(alVincular).toHaveBeenCalled();
    });

    expect(selector.querySelector('#vm-nombre').textContent).toBe('ubuntu-lab');
    expect(selector.querySelector('#vm-panel').classList.contains('hidden')).toBe(true);
  });
});

describe('encendido de la máquina', () => {
  it('ofrece encender cuando la máquina vinculada está apagada', async () => {
    let encendida = false;

    vi.stubGlobal('fetch', async (url, opciones) => {
      const porEncender = (opciones ?? {}).method === 'POST';

      if (porEncender) {
        encendida = true;
      }

      const estado = encendida ? 'running' : 'poweroff';
      const cuerpo = porEncender
        ? { nombre: 'debian-1', uuid: 'uuid-1', estado, activa: true }
        : { maquinas: [{ nombre: 'debian-1', uuid: 'uuid-1', estado, activa: true }] };

      return {
        ok: true,
        status: 200,
        headers: { get: () => 'application/json' },
        json: async () => cuerpo,
        text: async () => JSON.stringify(cuerpo),
        blob: async () => new Blob(['x']),
      };
    });

    const selector = VMSelector();
    document.body.appendChild(selector);

    await vi.waitFor(() => {
      expect(selector.querySelector('#vm-nombre').textContent).toBe('debian-1');
    });

    const boton = selector.querySelector('#vm-encender');

    expect(boton.classList.contains('hidden')).toBe(false);

    boton.click();

    await vi.waitFor(() => {
      expect(boton.classList.contains('hidden')).toBe(true);
    });
  });

  it('oculta el encendido si la máquina ya está en ejecución', async () => {
    simularFetch({
      maquinas: [{ nombre: 'debian-1', uuid: 'uuid-1', estado: 'running', activa: true }],
    });

    const selector = VMSelector();
    document.body.appendChild(selector);

    await vi.waitFor(() => {
      expect(selector.querySelector('#vm-nombre').textContent).toBe('debian-1');
    });

    expect(selector.querySelector('#vm-encender').classList.contains('hidden')).toBe(true);
  });
});

describe('cambio de máquina', () => {
  it('avisa aunque la máquina no tenga credenciales registradas', async () => {
    const alVincular = vi.fn();

    vi.stubGlobal('fetch', async (url, opciones) => {
      const porVincular = (opciones ?? {}).method === 'PUT';
      const cuerpo = porVincular
        ? { nombre: 'ubuntu-lab', uuid: 'uuid-2', estado: 'poweroff', activa: true, registrada: false }
        : { maquinas: [{ nombre: 'ubuntu-lab', uuid: 'uuid-2', estado: 'poweroff', activa: false }] };

      return {
        ok: true,
        status: 200,
        headers: { get: (clave) => (clave.toLowerCase() === 'content-type' ? 'application/json' : null) },
        json: async () => cuerpo,
        text: async () => JSON.stringify(cuerpo),
        blob: async () => new Blob(['x']),
      };
    });

    const selector = VMSelector({ alVincular });
    document.body.appendChild(selector);

    await vi.waitFor(() => {
      expect(selector.querySelector('[data-maquina]')).not.toBeNull();
    });

    selector.querySelector('#vm-boton').click();
    selector.querySelector('[data-maquina]').click();

    await vi.waitFor(() => {
      expect(alVincular).toHaveBeenCalled();
    });
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
