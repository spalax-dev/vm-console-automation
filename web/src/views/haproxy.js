import { renderIcons } from '../icons.js';
import { api } from '../api.js';
import { showResult } from '../components/ResultPanel.js';
import {
  ejecutar,
  escapar,
  filaTabla,
  inhabilitar,
  mensajeDeError,
  presentarEstadoServidor,
} from '../ui.js';
import { esIpv4, esPuerto, identificadorServidor } from '../validacion.js';

/**
 * Construye la vista del back-end de HAProxy (numeral 3.4).
 *
 * @returns {HTMLElement} Vista lista para insertarse.
 */
export function renderHaproxy() {
  const el = document.createElement('div');
  el.className = 'grid grid-cols-[58%_42%] gap-vm-gap';
  el.innerHTML = `
    <div class="flex flex-col gap-vm-gap">
      <section class="card">
        <header class="px-3 py-1.5 border-b border-vm-border bg-[#f7f7f7] flex items-center justify-between">
          <div class="flex items-center gap-1.5"><i data-lucide="server" class="w-4 h-4"></i><span class="text-[12px] font-medium">Servidores en el back-end</span></div>
          <div class="flex items-center gap-1.5">
            <span id="backend-cuenta" class="text-[11px] text-vm-fgSecondary"></span>
            <button id="backend-refresh" class="btn h-vm-btn px-1.5" title="Consultar de nuevo"><i data-lucide="refresh-cw" class="w-4 h-4"></i></button>
          </div>
        </header>
        <div class="overflow-auto">
          <table class="w-full text-[12px]">
            <thead class="bg-vm-bg">
              <tr>
                <th class="text-left px-3 py-1 font-medium text-vm-fgSecondary">Dirección</th>
                <th class="text-left px-3 py-1 font-medium text-vm-fgSecondary">Puerto</th>
                <th class="text-left px-3 py-1 font-medium text-vm-fgSecondary">Estado</th>
                <th class="px-3 py-1"></th>
              </tr>
            </thead>
            <tbody id="backend-filas"></tbody>
          </table>
        </div>
      </section>
      <section class="card">
        <header class="px-3 py-1.5 border-b border-vm-border bg-[#f7f7f7]"><div class="flex items-center gap-1.5"><i data-lucide="plus-circle" class="w-4 h-4"></i><span class="text-[12px] font-medium">Agregar servidor al back-end</span></div></header>
        <form id="form-backend" class="p-3 flex items-end gap-2" novalidate>
          <div class="flex flex-col gap-1 flex-1">
            <label class="label" for="backend-direccion">Dirección</label>
            <input id="backend-direccion" name="direccion" class="input font-mono" placeholder="192.168.1.111" autocomplete="off" spellcheck="false">
          </div>
          <div class="flex flex-col gap-1 w-[100px]">
            <label class="label" for="backend-puerto">Puerto</label>
            <input id="backend-puerto" name="puerto" class="input font-mono" placeholder="8080" inputmode="numeric" maxlength="5" autocomplete="off">
          </div>
          <button type="submit" class="btn btn-primary"><i data-lucide="plus"></i> Agregar</button>
        </form>
      </section>
    </div>
    <div id="result-slot"></div>
  `;

  renderIcons(el);

  /** Última lista de servidores recibida del servidor. */
  let servidores = [];

  const filas = el.querySelector('#backend-filas');
  const cuenta = el.querySelector('#backend-cuenta');
  const formulario = el.querySelector('#form-backend');

  const pintar = () => {
    cuenta.textContent = String(servidores.length);

    if (servidores.length === 0) {
      filas.innerHTML = filaTabla(4, 'El back-end no tiene servidores');
      return;
    }

    filas.innerHTML = servidores
      .map((servidor) => {
        const presentacion = presentarEstadoServidor(servidor.estado);
        const identificador = identificadorServidor(servidor.direccion, servidor.puerto);

        return `
        <tr class="border-t border-vm-border hover:bg-vm-bg">
          <td class="px-3 py-1.5 font-mono">${escapar(servidor.direccion)}</td>
          <td class="px-3 py-1.5 font-mono">${escapar(servidor.puerto)}</td>
          <td class="px-3 py-1.5"><span class="px-1.5 py-0.5 border text-[10px] rounded-sm ${presentacion.clase}">${presentacion.texto}</span></td>
          <td class="px-3 py-1.5 text-right"><button class="btn h-vm-btn px-1" data-eliminar="${escapar(identificador)}" title="Quitar del back-end"><i data-lucide="trash-2" class="w-4 h-4"></i></button></td>
        </tr>
      `;
      })
      .join('');

    renderIcons(filas);
  };

  const cargar = async () => {
    filas.innerHTML = filaTabla(4, 'Consultando los servidores…');

    try {
      const respuesta = await api.haproxy.listar();
      servidores = respuesta.servidores ?? [];
      pintar();
    } catch (error) {
      servidores = [];
      cuenta.textContent = '';
      filas.innerHTML = filaTabla(4, mensajeDeError(error), 'text-vm-accent');
    }
  };

  /**
   * Quita un servidor del back-end.
   *
   * @param {string} identificador Servidor como `direccion:puerto`.
   * @returns {Promise<void>}
   */
  const eliminar = async (identificador) => {
    if (!window.confirm(`¿Quitar ${identificador} del back-end?`)) {
      return;
    }

    await ejecutar({
      accion: async () => {
        await api.haproxy.eliminar(identificador);

        return { mensaje: `Se quitó ${identificador} del back-end.` };
      },
      operacion: `DELETE /api/haproxy/backend/servers/${identificador}`,
      informar: showResult,
      recargar: cargar,
    });
  };

  filas.addEventListener('click', (evento) => {
    const boton = evento.target.closest('[data-eliminar]');

    if (boton) {
      eliminar(boton.dataset.eliminar);
    }
  });

  el.querySelector('#backend-refresh').addEventListener('click', cargar);

  formulario.addEventListener('submit', async (evento) => {
    evento.preventDefault();

    const direccion = formulario.elements.direccion.value.trim();
    const puerto = Number(formulario.elements.puerto.value.trim());

    if (!esIpv4(direccion)) {
      showResult({ ok: false, mensaje: 'La dirección debe ser una IPv4 válida, por ejemplo 192.168.1.111.', operacion: 'POST /api/haproxy/backend/servers' });
      return;
    }

    if (!esPuerto(puerto)) {
      showResult({ ok: false, mensaje: 'El puerto debe ser un número entre 1 y 65535.', operacion: 'POST /api/haproxy/backend/servers' });
      return;
    }

    inhabilitar(formulario, true);

    const exito = await ejecutar({
      accion: async () => {
        await api.haproxy.agregar({ direccion, puerto });

        return { mensaje: `Se agregó ${direccion}:${puerto} al back-end.` };
      },
      operacion: `POST /api/haproxy/backend/servers ${direccion}:${puerto}`,
      informar: showResult,
      recargar: cargar,
    });

    inhabilitar(formulario, false);

    if (exito) {
      formulario.reset();
    }
  });

  cargar();

  return el;
}
