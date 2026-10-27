// @vitest-environment happy-dom

import { afterEach, describe, expect, it, vi } from 'vitest';

import { renderIcons } from './icons.js';
import { Tabs } from './components/Tabs.js';
import { VMSelector } from './components/VMSelector.js';
import { renderPaquetes } from './views/paquetes.js';
import { renderUsuarios } from './views/usuarios.js';
import { renderHaproxy } from './views/haproxy.js';
import { renderHistorial } from './views/historial.js';
import { renderIssue } from './views/issue.js';
import { renderLlaves } from './views/llaves.js';

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
  window.localStorage.clear();
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
    expect(selector.querySelector('#vm-olvidar').classList.contains('hidden')).toBe(true);
  });


  it('muestra las máquinas sin Guest Additions con un aviso', async () => {
    simularFetch({
      maquinas: [{
        nombre: 'debian-1', uuid: 'uuid-1', estado: 'running', activa: false,
        administrable: false, advertencia: 'Guest Additions no está disponible',
      }],
    });

    const selector = VMSelector();
    document.body.appendChild(selector);

    await vi.waitFor(() => {
      expect(selector.querySelector('#vm-lista').textContent).toContain('Sin Guest Additions');
    });

    expect(selector.querySelector('[data-maquina]')).toBeNull();

    selector.querySelector('[data-ocultar]').click();

    await vi.waitFor(() => {
      expect(selector.querySelector('#vm-pie').textContent).toContain('1 oculta');
    });
  });

  it('oculta una máquina no administrable y permite mostrarla de nuevo', async () => {
    window.localStorage.setItem('vmcons:maquinas-ocultas', JSON.stringify(['debian-1']));

    simularFetch({
      maquinas: [{
        nombre: 'debian-1', uuid: 'uuid-1', estado: 'running', activa: false,
        administrable: false, advertencia: 'Guest Additions no está disponible',
      }],
    });

    const selector = VMSelector();
    document.body.appendChild(selector);

    await vi.waitFor(() => {
      expect(selector.querySelector('#vm-pie').textContent).toContain('1 oculta');
    });

    expect(selector.querySelector('[data-maquina]')).toBeNull();

    selector.querySelector('#vm-mostrar-ocultas').click();

    await vi.waitFor(() => {
      expect(selector.querySelector('#vm-lista').textContent).toContain('Sin Guest Additions');
    });
  });

  it('conserva el inventario si una recarga falla', async () => {
    let fallar = false;

    vi.stubGlobal('fetch', async () => {
      if (fallar) {
        return {
          ok: false,
          status: 500,
          headers: { get: (clave) => (clave.toLowerCase() === 'content-type' ? 'application/json' : null) },
          json: async () => ({ mensaje: 'No fue posible consultar las máquinas virtuales del hipervisor.' }),
          text: async () => '{"mensaje":"No fue posible consultar las máquinas virtuales del hipervisor."}',
          blob: async () => new Blob(['x']),
        };
      }

      const cuerpo = {
        maquinas: [{ nombre: 'debian-1', uuid: 'uuid-1', estado: 'running', activa: true }],
      };

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
      expect(selector.querySelector('[data-maquina]')).not.toBeNull();
    });

    fallar = true;
    selector.querySelector('#vm-boton').click();

    await vi.waitFor(() => {
      expect(selector.querySelector('#vm-pie').textContent).toContain('No se pudo actualizar');
    });

    expect(selector.querySelector('[data-maquina]')).not.toBeNull();
    expect(selector.querySelector('#vm-lista').textContent).toContain('debian-1');
  });

  it('olvida el registro de la máquina vinculada tras confirmar', async () => {
    const llamadas = [];

    vi.stubGlobal('fetch', async (url, opciones = {}) => {
      const metodo = opciones.method ?? 'GET';
      llamadas.push(`${metodo} ${url}`);

      if (metodo === 'DELETE') {
        return {
          ok: true, status: 204, headers: { get: () => '' }, json: async () => null,
          text: async () => '', blob: async () => new Blob([]),
        };
      }

      const cuerpo = {
        maquinas: [{ nombre: 'debian-1', uuid: 'uuid-1', estado: 'running', activa: true, registrada: true }],
      };

      return {
        ok: true, status: 200, headers: { get: () => 'application/json' },
        json: async () => cuerpo, text: async () => JSON.stringify(cuerpo), blob: async () => new Blob(['x']),
      };
    });

    vi.stubGlobal('confirm', () => true);

    const selector = VMSelector();
    document.body.appendChild(selector);

    await vi.waitFor(() => {
      expect(selector.querySelector('#vm-olvidar').classList.contains('hidden')).toBe(false);
    });

    selector.querySelector('#vm-olvidar').click();

    await vi.waitFor(() => {
      expect(llamadas.some((llamada) => llamada.startsWith('DELETE /api/session/register'))).toBe(true);
    });

    await vi.waitFor(() => {
      expect(selector.querySelector('#vm-pie').textContent).toContain('olvidado');
    });
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
      const metodo = (opciones ?? {}).method ?? 'GET';
      const porEncender = metodo === 'POST';

      if (porEncender) {
        encendida = true;
      }

      const estado = encendida ? 'running' : 'poweroff';
      let cuerpo;

      if (porEncender) {
        cuerpo = { nombre: 'debian-1', uuid: 'uuid-1', estado, activa: true };
      } else if (url.includes('/session')) {
        cuerpo = { vm: 'debian-1', conector: 'ssh', ip: '192.168.1.16', estado };
      } else {
        cuerpo = { maquinas: [{ nombre: 'debian-1', uuid: 'uuid-1', estado, activa: true }] };
      }

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
      expect(document.querySelector('#vm-power-overlay')).not.toBeNull();
    });

    await vi.waitFor(() => {
      expect(boton.classList.contains('hidden')).toBe(true);
    }, { timeout: 3000 });

    await vi.waitFor(() => {
      expect(document.querySelector('#vm-power-overlay')).toBeNull();
    }, { timeout: 3000 });
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
  it('notifies the caller without opening registration for an unregistered machine', async () => {
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

    expect(document.querySelector('#registro-form')).toBeNull();
    expect(document.querySelector('[role="dialog"]')).toBeNull();
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

  it('no muestra el bloque de descarga de llaves y conserva la descarga por fila', async () => {
    prepararRanura();
    simularFetch({
      usuarios: [{
        nombre: 'alumno',
        sudo: false,
        llaves: true,
        uid: '1001',
        home: '/home/alumno',
      }],
    });

    const vista = renderUsuarios();
    document.body.appendChild(vista);

    await vi.waitFor(() => {
      expect(vista.textContent).toContain('alumno');
    });

    expect(vista.textContent).not.toContain('Descarga de llaves');
    expect(vista.querySelector('#descarga-nombre')).toBeNull();
    expect(vista.querySelector('[data-descargar="alumno"]')).not.toBeNull();
  });

  it('muestra el resumen legible de cada operación del historial', async () => {
    prepararRanura();
    simularFetch({
      operaciones: [{
        id: 1,
        fecha_hora: '2026-10-25T10:00:00Z',
        vm: 'debian-1',
        modulo: 'paquetes',
        operacion: 'POST /api/packages',
        descripcion: 'Instalar el paquete «nano»',
        resultado: 'ok',
        detalle: 'HTTP 201',
      }],
    });

    const vista = renderHistorial();
    document.body.appendChild(vista);

    await vi.waitFor(() => {
      expect(vista.querySelector('#historial-filas').textContent).toContain('Instalar el paquete «nano»');
    });

    const celdas = vista.querySelectorAll('#historial-filas tr td');
    expect(celdas).toHaveLength(6);
    expect(celdas[4].textContent).toBe('POST /api/packages');
  });
});

describe('configuración de /etc/issue', () => {
  it('renderiza la vista previa con los valores y conserva el mensaje', async () => {
    vi.stubGlobal('fetch', async (url) => {
      const cuerpo = url.includes('/hostname')
        ? { nombre: 'vm1', ip: '192.168.1.16', actualizar_hosts: true }
        : {
            mensaje: 'Bienvenido',
            nombre_host: true,
            fecha: true,
            hora: true,
            ip: true,
            contenido: 'Bienvenido\nHostname: \\n\nFecha: \\d\nHora: \\t\nIP: \\4\n',
          };

      return {
        ok: true,
        status: 200,
        headers: { get: () => 'application/json' },
        json: async () => cuerpo,
        text: async () => JSON.stringify(cuerpo),
        blob: async () => new Blob(['x']),
      };
    });

    const vista = renderIssue();
    document.body.appendChild(vista);

    await vi.waitFor(() => {
      expect(vista.querySelector('#issue-mensaje').value).toContain('Hostname: \\n');
    });

    const campo = vista.querySelector('#issue-mensaje').value;

    expect(campo).toContain('Bienvenido');
    expect(campo).toContain('Fecha: \\d');
    expect(campo).toContain('Hora: \\t');
    expect(campo).toContain('IP: \\4');

    const preview = vista.querySelector('#issue-preview').textContent;

    expect(preview).toContain('Hostname: vm1');
    expect(preview).toContain('IP: 192.168.1.16');
    expect(preview).not.toContain('\\4');
    expect(preview).not.toContain('\\n');
  });

  it('inserta y quita el bloque de una casilla en el mensaje', async () => {
    vi.stubGlobal('fetch', async (url) => {
      const cuerpo = url.includes('/hostname')
        ? { nombre: 'vm1', ip: '192.168.1.16', actualizar_hosts: true }
        : {
            mensaje: 'Bienvenido',
            nombre_host: false,
            fecha: false,
            hora: false,
            ip: false,
            contenido: 'Bienvenido\n',
          };

      return {
        ok: true,
        status: 200,
        headers: { get: () => 'application/json' },
        json: async () => cuerpo,
        text: async () => JSON.stringify(cuerpo),
        blob: async () => new Blob(['x']),
      };
    });

    const vista = renderIssue();
    document.body.appendChild(vista);

    await vi.waitFor(() => {
      expect(vista.querySelector('#issue-mensaje').value).toBe('Bienvenido');
    });

    const casilla = vista.querySelector('input[name="hora"]');
    casilla.click();

    await vi.waitFor(() => {
      expect(vista.querySelector('#issue-mensaje').value).toContain('Hora: \\t');
    });

    expect(casilla.checked).toBe(true);

    casilla.click();

    await vi.waitFor(() => {
      expect(vista.querySelector('#issue-mensaje').value).not.toContain('Hora: \\t');
    });

    expect(casilla.checked).toBe(false);
  });
});

describe('llaves SSH', () => {
  it('permite elegir root o un usuario y muestra su ruta', async () => {
    const llamadas = [];

    vi.stubGlobal('fetch', async (url, opciones = {}) => {
      llamadas.push(`${opciones.method ?? 'GET'} ${url}`);

      const cuerpo = url.includes('/keys/root/status')
        ? { existe: true }
        : url.includes('/users')
        ? {
            usuarios: [
              { nombre: 'alumno', sudo: false, llaves: false, uid: '1001', home: '/home/alumno' },
              { nombre: 'andrey', sudo: false, llaves: true, uid: '1002', home: '/home/andrey' },
            ],
          }
        : {};

      return {
        ok: true,
        status: 200,
        headers: { get: () => 'application/json' },
        json: async () => cuerpo,
        text: async () => JSON.stringify(cuerpo),
        blob: async () => new Blob(['x']),
      };
    });

    const vista = renderLlaves();
    document.body.appendChild(vista);

    await vi.waitFor(() => {
      expect(vista.querySelector('#llave-usuario').options).toHaveLength(3);
    });

    const select = vista.querySelector('#llave-usuario');

    expect(select.options[0].textContent).toContain('root (conexión)');
    expect(vista.querySelector('#llave-estado').textContent).toBe('activa');
    expect(vista.querySelector('#llave-generar').textContent).toContain('La llave de conexión ya existe');
    expect(vista.querySelector('#llave-generar').disabled).toBe(true);

    select.value = 'alumno';
    select.dispatchEvent(new Event('change'));

    expect(vista.querySelector('#llave-titulo').textContent).toContain('alumno');
    expect(vista.querySelector('#llave-descripcion').textContent).toContain('/home/alumno/.ssh/authorized_keys');
    expect(vista.querySelector('#llave-estado').textContent).toBe('sin llave');
    expect(vista.querySelector('#llave-generar').textContent).toContain('Generar par de llaves');

    select.value = 'andrey';
    select.dispatchEvent(new Event('change'));

    expect(vista.querySelector('#llave-estado').textContent).toBe('activa');
    expect(vista.querySelector('#llave-generar').textContent).toContain('Regenerar par de llaves');

    vista.querySelector('#llave-generar').click();

    expect(vista.querySelector('#llave-confirmacion').classList.contains('hidden')).toBe(false);

    vista.querySelector('#llave-confirmar-regenerar').click();

    await vi.waitFor(() => {
      expect(llamadas.some((llamada) => llamada.includes('regenerar=true'))).toBe(true);
    });
  });
});
