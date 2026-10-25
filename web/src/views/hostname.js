import { renderIcons } from '../icons.js';
import { api } from '../api.js';
import { showResult } from '../components/ResultPanel.js';
import { ejecutar, escapar, inhabilitar, mensajeDeError } from '../ui.js';
import { esIpv4, validarNombreHost } from '../validacion.js';

/**
 * Construye la vista del nombre de host (numeral 3.1.4).
 *
 * @returns {HTMLElement} Vista lista para insertarse.
 */
export function renderHostname() {
  const el = document.createElement('div');
  el.className = 'grid grid-cols-[58%_42%] gap-vm-gap items-start';
  el.innerHTML = `
    <section class="card">
      <header class="px-3 py-1.5 border-b border-vm-border bg-[#f7f7f7] flex items-center justify-between">
        <div class="flex items-center gap-1.5"><i data-lucide="server" class="w-4 h-4"></i><span class="text-[12px] font-medium">Nombre de host</span></div>
        <button id="hostname-refresh" class="btn h-vm-btn px-1.5" title="Consultar de nuevo"><i data-lucide="refresh-cw" class="w-4 h-4"></i></button>
      </header>
      <form id="form-hostname" class="p-3 flex flex-col gap-2" novalidate>
        <div class="flex flex-col gap-1">
          <label class="label" for="hostname-nombre">Nombre de host</label>
          <input id="hostname-nombre" name="nombre" class="input font-mono" placeholder="debian13" autocomplete="off" spellcheck="false">
          <span id="hostname-error-nombre" class="text-[11px] text-vm-accent hidden"></span>
          <span class="text-[11px] text-vm-fgSecondary">Minúsculas, dígitos y guiones internos. Máximo 63 caracteres.</span>
        </div>
        <div class="flex flex-col gap-1">
          <label class="label" for="hostname-ip">Dirección IP</label>
          <input id="hostname-ip" name="ip" class="input font-mono" placeholder="192.168.1.105" autocomplete="off" spellcheck="false">
          <span id="hostname-error-ip" class="text-[11px] text-vm-accent hidden"></span>
        </div>
        <label class="flex items-center gap-1.5 text-[12px]"><input type="checkbox" name="actualizar_hosts" checked> Actualizar también <span class="font-mono">/etc/hosts</span></label>
        <div class="flex gap-2">
          <button type="submit" class="btn btn-primary"><i data-lucide="play"></i> Aplicar</button>
          <button type="button" id="hostname-restablecer" class="btn"><i data-lucide="rotate-ccw"></i> Restablecer</button>
        </div>
      </form>
    </section>
    <div id="result-slot"></div>
  `;

  renderIcons(el);

  const formulario = el.querySelector('#form-hostname');
  const nombre = el.querySelector('#hostname-nombre');
  const ip = el.querySelector('#hostname-ip');
  const errorNombre = el.querySelector('#hostname-error-nombre');
  const errorIp = el.querySelector('#hostname-error-ip');

  /** Última configuración confirmada por el servidor. */
  let confirmada = { nombre: '', ip: '', actualizar_hosts: true };

  /**
   * Muestra u oculta el mensaje de error de un campo.
   *
   * @param {HTMLElement} aviso Elemento del aviso.
   * @param {string|null} texto Mensaje o `null` para ocultarlo.
   */
  const avisar = (aviso, texto) => {
    aviso.textContent = texto ?? '';
    aviso.classList.toggle('hidden', !texto);
  };

  const cargar = async () => {
    inhabilitar(formulario, true);

    try {
      const configuracion = await api.hostname.obtener();

      confirmada = { ...confirmada, ...configuracion };
      nombre.value = configuracion.nombre ?? '';
      ip.value = configuracion.ip ?? '';
      formulario.elements.actualizar_hosts.checked = configuracion.actualizar_hosts !== false;
    } catch (error) {
      avisar(errorNombre, mensajeDeError(error));
    } finally {
      inhabilitar(formulario, false);
    }
  };

  formulario.addEventListener('input', () => {
    avisar(errorNombre, null);
    avisar(errorIp, null);
  });

  el.querySelector('#hostname-refresh').addEventListener('click', cargar);
  el.querySelector('#hostname-restablecer').addEventListener('click', () => {
    nombre.value = confirmada.nombre ?? '';
    ip.value = confirmada.ip ?? '';
    formulario.elements.actualizar_hosts.checked = confirmada.actualizar_hosts !== false;
    avisar(errorNombre, null);
    avisar(errorIp, null);
  });

  formulario.addEventListener('submit', async (evento) => {
    evento.preventDefault();

    const invalidoNombre = validarNombreHost(nombre.value.trim());
    const invalidoIp = esIpv4(ip.value.trim()) ? null : 'La dirección debe ser una IPv4 válida, por ejemplo 192.168.1.105';

    avisar(errorNombre, invalidoNombre);
    avisar(errorIp, invalidoIp);

    if (invalidoNombre) {
      nombre.focus();
      return;
    }

    if (invalidoIp) {
      ip.focus();
      return;
    }

    const solicitud = {
      nombre: nombre.value.trim(),
      ip: ip.value.trim(),
      actualizar_hosts: formulario.elements.actualizar_hosts.checked,
    };

    inhabilitar(formulario, true);

    await ejecutar({
      accion: async () => {
        await api.hostname.reemplazar(solicitud);

        return { mensaje: `Se aplicó el nombre de host «${solicitud.nombre}».` };
      },
      operacion: 'PUT /api/hostname',
      informar: showResult,
      recargar: cargar,
    });

    inhabilitar(formulario, false);
  });

  cargar();

  return el;
}
