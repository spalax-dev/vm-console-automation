import { renderIcons } from '../icons.js';
import { api } from '../api.js';
import { showResult } from '../components/ResultPanel.js';
import { ejecutar, escapar, inhabilitar, mensajeDeError } from '../ui.js';

/** Opciones del banner y su escape de `agetty`. */
const OPCIONES = [
  ['nombre_host', 'Mostrar nombre del host', '\\n'],
  ['fecha', 'Mostrar fecha', '\\d'],
  ['hora', 'Mostrar hora', '\\t'],
  ['ip', 'Mostrar dirección IP', '\\4'],
];

/**
 * Construye la vista de configuración de `/etc/issue` (numeral 3.1.3).
 *
 * @returns {HTMLElement} Vista lista para insertarse.
 */
export function renderIssue() {
  const el = document.createElement('div');
  el.className = 'grid grid-cols-[58%_42%] gap-vm-gap';
  el.innerHTML = `
    <section class="card">
      <header class="px-3 py-1.5 border-b border-vm-border bg-[#f7f7f7] flex items-center justify-between">
        <div class="flex items-center gap-1.5"><i data-lucide="file-text" class="w-4 h-4"></i><span class="text-[12px] font-medium">Configuración de /etc/issue</span></div>
        <button id="issue-refresh" class="btn h-vm-btn px-1.5" title="Consultar de nuevo"><i data-lucide="refresh-cw" class="w-4 h-4"></i></button>
      </header>
      <form id="form-issue" class="p-3 flex flex-col gap-2" novalidate>
        <div class="flex flex-col gap-1">
          <label class="label" for="issue-mensaje">Mensaje personalizado</label>
          <textarea id="issue-mensaje" name="mensaje" class="input h-auto py-1 font-mono" rows="2" maxlength="512" placeholder="Debian GNU/Linux 13"></textarea>
          <span class="text-[11px] text-vm-fgSecondary">Hasta 512 caracteres. No se admite el carácter nulo.</span>
        </div>
        <div class="flex flex-col gap-1 text-[12px]">
          ${OPCIONES.map(([clave, etiqueta]) => `<label class="flex items-center gap-1.5"><input type="checkbox" name="${clave}"> ${etiqueta}</label>`).join('')}
        </div>
        <section class="card">
          <header class="px-3 py-1 border-b border-vm-border bg-[#f7f7f7]"><div class="flex items-center gap-1.5"><i data-lucide="eye" class="w-3 h-3"></i><span class="text-[11px]">Vista previa</span></div></header>
          <pre id="issue-preview" class="p-2 font-mono text-[11px] whitespace-pre-wrap"></pre>
        </section>
        <div class="flex gap-2">
          <button type="submit" class="btn btn-primary"><i data-lucide="save"></i> Guardar</button>
          <button type="button" id="issue-restablecer" class="btn"><i data-lucide="rotate-ccw"></i> Restablecer</button>
        </div>
      </form>
    </section>
    <div id="result-slot"></div>
  `;

  renderIcons(el);

  const formulario = el.querySelector('#form-issue');
  const mensaje = el.querySelector('#issue-mensaje');
  const vistaPrevia = el.querySelector('#issue-preview');

  /** Última configuración confirmada por el servidor. */
  let confirmada = { mensaje: '', nombre_host: false, fecha: false, hora: false, ip: false, contenido: '' };

  /**
   * Lee el formulario y devuelve la solicitud del contrato.
   *
   * @returns {{mensaje: string, nombre_host: boolean, fecha: boolean, hora: boolean, ip: boolean}}
   */
  const leer = () => ({
    mensaje: mensaje.value,
    nombre_host: formulario.elements.nombre_host.checked,
    fecha: formulario.elements.fecha.checked,
    hora: formulario.elements.hora.checked,
    ip: formulario.elements.ip.checked,
  });

  /** Repinta la vista previa local a partir del formulario. */
  const previsualizar = () => {
    const configuracion = leer();
    const marcas = OPCIONES.filter(([clave]) => configuracion[clave]).map(([, , escape]) => escape);
    const lineas = [];

    if (configuracion.mensaje.trim()) {
      lineas.push(configuracion.mensaje);
    }

    if (marcas.length > 0) {
      lineas.push(marcas.join(' '));
    }

    vistaPrevia.textContent = lineas.length > 0 ? lineas.join('\n') : '(banner vacío)';
  };

  /**
   * Vuelca una configuración en el formulario.
   *
   * @param {object} configuracion Configuración del contrato.
   */
  const volcar = (configuracion) => {
    confirmada = { ...confirmada, ...configuracion };
    mensaje.value = configuracion.mensaje ?? '';
    formulario.elements.nombre_host.checked = Boolean(configuracion.nombre_host);
    formulario.elements.fecha.checked = Boolean(configuracion.fecha);
    formulario.elements.hora.checked = Boolean(configuracion.hora);
    formulario.elements.ip.checked = Boolean(configuracion.ip);

    if (configuracion.contenido) {
      vistaPrevia.textContent = configuracion.contenido;
    } else {
      previsualizar();
    }
  };

  const cargar = async () => {
    inhabilitar(formulario, true);
    vistaPrevia.textContent = 'Consultando…';

    try {
      volcar(await api.issue.obtener());
    } catch (error) {
      vistaPrevia.textContent = mensajeDeError(error);
    } finally {
      inhabilitar(formulario, false);
    }
  };

  formulario.addEventListener('input', previsualizar);
  formulario.addEventListener('change', previsualizar);
  el.querySelector('#issue-refresh').addEventListener('click', cargar);
  el.querySelector('#issue-restablecer').addEventListener('click', () => volcar(confirmada));

  formulario.addEventListener('submit', async (evento) => {
    evento.preventDefault();
    inhabilitar(formulario, true);

    const configuracion = leer();

    const exito = await ejecutar({
      accion: async () => {
        await api.issue.reemplazar(configuracion);

        return { mensaje: 'Se actualizó el banner de /etc/issue.' };
      },
      operacion: 'PUT /api/issue',
      informar: showResult,
    });

    inhabilitar(formulario, false);

    if (exito) {
      confirmada = { ...configuracion, contenido: '' };
      cargar();
    }
  });

  cargar();

  return el;
}
