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
      <section id="haproxy-estado-card" class="card hidden border border-vm-accent">
        <header class="px-3 py-1.5 border-b border-vm-border bg-[#f7f7f7]"><div class="flex items-center gap-1.5"><i data-lucide="alert-triangle" class="w-4 h-4 text-vm-accent"></i><span class="text-[12px] font-medium">HAProxy requiere configuración</span></div></header>
        <div class="p-3 flex flex-col gap-2">
          <p id="haproxy-estado-texto" class="text-[12px] text-vm-fg"></p>
          <ul id="haproxy-problemas" class="text-[11px] text-vm-accent list-disc pl-4"></ul>
          <ul id="haproxy-sugerencias" class="text-[11px] text-vm-fgSecondary list-disc pl-4"></ul>
          <button id="haproxy-configurar" type="button" class="btn btn-primary self-start"><i data-lucide="settings"></i> Configuración rápida</button>
        </div>
      </section>
      <section id="haproxy-entrada" class="card hidden">
        <div class="p-3 flex items-center justify-between gap-3 text-[12px]">
          <div class="flex flex-col gap-0.5 min-w-0">
            <span class="text-vm-fgSecondary">Entrada del balanceador</span>
            <a id="haproxy-url" class="font-mono text-vm-accent break-all" target="_blank" rel="noopener"></a>
          </div>
          <button id="haproxy-copiar" type="button" class="btn shrink-0" title="Copiar dirección y puerto">
            <i data-lucide="copy"></i> <span id="haproxy-copiar-texto">Copiar</span>
          </button>
        </div>
      </section>
      <section class="card">
        <header class="px-3 py-1.5 border-b border-vm-border bg-[#f7f7f7] flex items-center justify-between">
          <div class="flex items-center gap-1.5"><i data-lucide="server" class="w-4 h-4"></i><span class="text-[12px] font-medium">Servidores en el back-end</span></div>
          <div class="flex items-center gap-1.5">
            <span id="backend-cuenta" class="text-[11px] text-vm-fgSecondary"></span>
            <span class="vm-help">
              <button type="button" class="vm-help-boton" aria-label="Ayuda sobre las métricas" aria-describedby="backend-ayuda">
                <i data-lucide="info" class="w-4 h-4"></i>
              </button>
              <span id="backend-ayuda" class="vm-help-texto" role="tooltip">
                <span class="block"><span class="font-medium">Sesiones:</span> activas / máximo desde el arranque. Un <span class="font-mono">0/1</span> sin tráfico es normal; el total acumulado está en el detalle.</span>
                <span class="block mt-1"><span class="font-medium">Petición / respuesta:</span> bytes enviados al servidor y recibidos del servidor.</span>
                <span class="block mt-1"><span class="font-mono">L4OK</span> es un chequeo TCP: confirma que el puerto acepta conexiones, no que la aplicación responda HTTP 200.</span>
              </span>
            </span>
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
                <th class="text-left px-3 py-1 font-medium text-vm-fgSecondary">Sesiones (act./máx.)</th>
                <th class="text-left px-3 py-1 font-medium text-vm-fgSecondary" title="Bytes enviados al servidor / bytes recibidos del servidor">Petición / respuesta</th>
                <th class="text-left px-3 py-1 font-medium text-vm-fgSecondary">Peso</th>
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
  const estadoCard = el.querySelector('#haproxy-estado-card');
  const estadoTexto = el.querySelector('#haproxy-estado-texto');
  const problemas = el.querySelector('#haproxy-problemas');
  const sugerencias = el.querySelector('#haproxy-sugerencias');
  const configurar = el.querySelector('#haproxy-configurar');
  const entradaCard = el.querySelector('#haproxy-entrada');
  const entradaURL = el.querySelector('#haproxy-url');
  const copiar = el.querySelector('#haproxy-copiar');
  const copiarTexto = el.querySelector('#haproxy-copiar-texto');

  /**
   * Muestra el enlace de entrada del balanceador con la IP y el puerto.
   *
   * @param {object} estado Estado devuelto por la API.
   * @returns {Promise<void>}
   */
  const cargarEntrada = async (estado) => {
    const puerto = Number(estado.puerto) || 0;

    if (puerto <= 0) {
      entradaCard.classList.add('hidden');

      return;
    }

    try {
      const sesion = await api.sesion.obtener();
      const ip = sesion.ip;

      if (!ip) {
        entradaCard.classList.add('hidden');

        return;
      }

      const url = `http://${ip}:${puerto}`;
      entradaURL.href = url;
      entradaURL.textContent = url;
      entradaCard.classList.remove('hidden');
    } catch {
      entradaCard.classList.add('hidden');
    }
  };

  /** Consulta el estado de HAProxy y muestra el aviso si requiere configuración. */
  const cargarEstado = async () => {
    try {
      const estado = await api.haproxy.estado();
      await cargarEntrada(estado);
      const listaProblemas = estado.problemas ?? [];
      const listaSugerencias = estado.sugerencias ?? [];

      if (listaProblemas.length === 0) {
        estadoCard.classList.add('hidden');
        configurar.classList.add('hidden');

        return;
      }

      const puerto = Number(estado.puerto) > 0 ? ` Puerto configurado: ${estado.puerto}.` : '';
      estadoTexto.textContent = `No se puede administrar el balanceador todavía.${puerto}`;
      problemas.innerHTML = listaProblemas.map((problema) => `<li>${escapar(problema)}</li>`).join('');
      sugerencias.innerHTML = listaSugerencias.map((sugerencia) => `<li>${escapar(sugerencia)}</li>`).join('');
      estadoCard.classList.remove('hidden');
      configurar.classList.remove('hidden');
    } catch (error) {
      estadoTexto.textContent = mensajeDeError(error);
      problemas.innerHTML = '';
      sugerencias.innerHTML = '';
      estadoCard.classList.remove('hidden');
      configurar.classList.add('hidden');
      entradaCard.classList.add('hidden');
    }
  };

  /**
   * Formatea una cantidad de bytes.
   *
   * @param {number} valor Bytes.
   * @returns {string} Texto legible.
   */
  const formatearBytes = (valor) => {
    const bytes = Number(valor) || 0;
    const unidades = ['B', 'KB', 'MB', 'GB', 'TB'];
    let cantidad = bytes;
    let indice = 0;

    while (cantidad >= 1024 && indice < unidades.length - 1) {
      cantidad /= 1024;
      indice += 1;
    }

    return indice === 0 ? `${cantidad} B` : `${cantidad.toFixed(1)} ${unidades[indice]}`;
  };

  /**
   * Formatea una duración en segundos.
   *
   * @param {number} valor Segundos.
   * @returns {string} Texto legible.
   */
  const formatearDuracion = (valor) => {
    let segundos = Number(valor) || 0;

    if (segundos <= 0) {
      return '—';
    }

    const dias = Math.floor(segundos / 86400);
    segundos %= 86400;

    const horas = Math.floor(segundos / 3600);
    segundos %= 3600;

    const minutos = Math.floor(segundos / 60);
    const partes = [];

    if (dias > 0) partes.push(`${dias} d`);
    if (horas > 0) partes.push(`${horas} h`);
    if (minutos > 0) partes.push(`${minutos} min`);
    if (partes.length === 0) partes.push(`${segundos} s`);

    return partes.join(' ');
  };

  const pintar = () => {
    cuenta.textContent = String(servidores.length);

    if (servidores.length === 0) {
      filas.innerHTML = filaTabla(7, 'El back-end no tiene servidores');
      return;
    }

    filas.innerHTML = servidores
      .map((servidor, indice) => {
        const presentacion = presentarEstadoServidor(servidor.estado);
        const identificador = identificadorServidor(servidor.direccion, servidor.puerto);
        const sesiones = servidor.sesiones ?? 0;
        const sesionesMax = servidor.sesiones_max ?? 0;
        const entrada = formatearBytes(servidor.bytes_entrada);
        const salida = formatearBytes(servidor.bytes_salida);
        const check = servidor.check ? `${servidor.check}${servidor.check_codigo ? ` (${servidor.check_codigo})` : ''}` : '—';
        const checkDuracion = Number(servidor.check_duracion) || 0;
        const ultimaSesion = Number(servidor.ultima_sesion) || 0;
        const sinMetricas = servidor.estado === 'unknown';

        return `
        <tr class="border-t border-vm-border hover:bg-vm-bg">
          <td class="px-3 py-1.5 font-mono">${escapar(servidor.direccion)}</td>
          <td class="px-3 py-1.5 font-mono">${escapar(servidor.puerto)}</td>
          <td class="px-3 py-1.5"><span class="px-1.5 py-0.5 border text-[10px] rounded-sm ${presentacion.clase}">${presentacion.texto}</span></td>
          <td class="px-3 py-1.5 font-mono text-[11px]">${sesiones} / ${sesionesMax}</td>
          <td class="px-3 py-1.5 font-mono text-[11px] whitespace-nowrap" title="Enviado al servidor / recibido del servidor">${entrada} / ${salida}</td>
          <td class="px-3 py-1.5 font-mono text-[11px]">${servidor.peso ?? '—'}</td>
          <td class="px-3 py-1.5 text-right whitespace-nowrap">
            <button class="btn h-vm-btn px-1" data-detalle="${indice}" title="Ver métricas"><i data-lucide="chevron-down" class="w-4 h-4"></i></button>
            <button class="btn h-vm-btn px-1" data-eliminar="${escapar(identificador)}" title="Quitar del back-end"><i data-lucide="trash-2" class="w-4 h-4"></i></button>
          </td>
        </tr>
        <tr class="hidden border-t border-vm-border bg-vm-bg" data-detalle-fila="${indice}">
          <td colspan="7" class="px-3 py-2 text-[11px]">
            <div class="grid grid-cols-2 gap-x-6 gap-y-1">
              <span><span class="text-vm-fgSecondary">Chequeo:</span> ${escapar(check)} <span class="text-vm-fgSecondary">· ${checkDuracion > 0 ? `${checkDuracion} ms` : '—'}</span></span>
              <span><span class="text-vm-fgSecondary">Activo / respaldo:</span> ${servidor.activo ? 'activo' : '—'} / ${servidor.respaldo ? 'respaldo' : '—'}</span>
              <span><span class="text-vm-fgSecondary">Sesiones:</span> ${sesiones} actuales · ${sesionesMax} máx · ${servidor.sesiones_total ?? 0} total</span>
              <span><span class="text-vm-fgSecondary">Cola:</span> ${servidor.cola ?? 0} actual · ${servidor.cola_max ?? 0} máx</span>
              <span><span class="text-vm-fgSecondary">Enviado al servidor:</span> ${entrada}</span>
              <span><span class="text-vm-fgSecondary">Recibido del servidor:</span> ${salida}</span>
              <span><span class="text-vm-fgSecondary">Indisponibilidad:</span> ${formatearDuracion(servidor.downtime)}</span>
              <span><span class="text-vm-fgSecondary">Último cambio:</span> ${formatearDuracion(servidor.ultimo_cambio)}</span>
              <span><span class="text-vm-fgSecondary">Última sesión:</span> ${sinMetricas ? '—' : ultimaSesion > 0 ? `hace ${formatearDuracion(ultimaSesion)}` : 'ahora'}</span>
              <span><span class="text-vm-fgSecondary">Mensaje del chequeo:</span> ${escapar(servidor.ultimo_check || '—')}</span>
            </div>
          </td>
        </tr>
      `;
      })
      .join('');

    renderIcons(filas);
  };

  const cargar = async () => {
    filas.innerHTML = filaTabla(7, 'Consultando los servidores…');

    try {
      const respuesta = await api.haproxy.listar();
      servidores = respuesta.servidores ?? [];
      pintar();
    } catch (error) {
      servidores = [];
      cuenta.textContent = '';
      filas.innerHTML = filaTabla(7, mensajeDeError(error), 'text-vm-accent');
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
    const detalle = evento.target.closest('[data-detalle]');

    if (detalle) {
      const fila = filas.querySelector(`[data-detalle-fila="${detalle.dataset.detalle}"]`);
      fila?.classList.toggle('hidden');

      return;
    }

    const boton = evento.target.closest('[data-eliminar]');

    if (boton) {
      eliminar(boton.dataset.eliminar);
    }
  });

  el.querySelector('#backend-refresh').addEventListener('click', () => {
    cargarEstado();
    cargar();
  });

  configurar.addEventListener('click', async () => {
    const confirmado = window.confirm(
      'Se instalará HAProxy si falta y se reemplazará /etc/haproxy/haproxy.cfg por una configuración mínima con un back-end vacío. Se guarda una copia de la configuración anterior. ¿Continuar?',
    );

    if (!confirmado) {
      return;
    }

    inhabilitar(el, true);

    const exito = await ejecutar({
      accion: () => api.haproxy.configurar(),
      operacion: 'POST /api/haproxy/setup',
      informar: showResult,
    });

    inhabilitar(el, false);

    if (exito) {
      await cargarEstado();
      await cargar();
    }
  });

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

  copiar.addEventListener('click', async () => {
    const texto = entradaURL.href;

    try {
      await navigator.clipboard.writeText(texto);
    } catch {
      const area = document.createElement('textarea');
      area.value = texto;
      document.body.appendChild(area);
      area.select();
      document.execCommand('copy');
      area.remove();
    }

    copiarTexto.textContent = 'Copiado';
    window.setTimeout(() => {
      copiarTexto.textContent = 'Copiar';
    }, 1500);
  });

  cargarEstado();
  cargar();

  return el;
}
