import { renderIcons } from '../icons.js';
import { api } from '../api.js';
import { showResult } from '../components/ResultPanel.js';
import { ejecutar, escapar, inhabilitar, mensajeDeError } from '../ui.js';

/** Opciones del banner y su escape de `agetty`. */
const OPCIONES = [
  ['nombre_host', 'Mostrar nombre del host', 'Hostname: \\n'],
  ['fecha', 'Mostrar fecha', 'Fecha: \\d'],
  ['hora', 'Mostrar hora', 'Hora: \\t'],
  ['ip', 'Mostrar dirección IP', 'IP: \\4'],
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
          <span class="text-[11px] text-vm-fgSecondary">Hasta 512 caracteres. Se permiten saltos de línea.</span>
        </div>
        <div class="flex flex-col gap-1 text-[12px]">
          ${OPCIONES.map(([clave, etiqueta]) => `<label class="flex items-center gap-1.5"><input type="checkbox" name="${clave}"> ${etiqueta}</label>`).join('')}
          <span class="text-[11px] text-vm-fgSecondary">Al marcar una casilla se agrega su bloque al mensaje; puedes editarlo o quitarlo.</span>
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

  /** Datos de la máquina para renderizar la vista previa. */
  let contexto = { nombre: '', ip: '' };

  /**
   * Sustituye las secuencias de agetty por valores visibles.
   *
   * @param {string} texto Texto del banner.
   * @param {Record<string, string>} valores Valores por secuencia.
   * @returns {string} Texto renderizado.
   */
  const renderizar = (texto, valores) =>
    texto.replace(/\\([a-zA-Z0-9])/g, (original, clave) => valores[clave] ?? original);

  /** Valores de las secuencias para la vista previa. */
  const valoresPreview = () => ({
    n: contexto.nombre || 'nombre-host',
    d: new Intl.DateTimeFormat('es-CO').format(new Date()),
    t: new Date().toLocaleTimeString('es-CO', { hour12: false }),
    4: contexto.ip || '192.168.1.10',
    l: 'tty1',
    s: 'Linux',
    S: 'Debian GNU/Linux',
  });

  /**
   * Agrega un bloque de opción al mensaje si todavía no está.
   *
   * @param {string} texto Mensaje actual.
   * @param {string} bloque Línea que se agrega.
   * @returns {string} Mensaje con el bloque.
   */
  const agregarBloque = (texto, bloque) => {
    const lineas = texto.split('\n').map((linea) => linea.trim());

    if (lineas.includes(bloque.trim())) {
      return texto;
    }

    return texto.trimEnd() ? `${texto.trimEnd()}\n${bloque}` : bloque;
  };

  /**
   * Quita un bloque de opción del mensaje.
   *
   * @param {string} texto Mensaje actual.
   * @param {string} bloque Línea que se quita.
   * @returns {string} Mensaje sin el bloque.
   */
  const quitarBloque = (texto, bloque) =>
    texto
      .split('\n')
      .filter((linea) => linea.trim() !== bloque.trim())
      .join('\n');

  /** Sincroniza las casillas con los bloques presentes en el mensaje. */
  const sincronizarCasillas = () => {
    const lineas = mensaje.value.split('\n').map((linea) => linea.trim());

    for (const [clave, , bloque] of OPCIONES) {
      formulario.elements[clave].checked = lineas.includes(bloque.trim());
    }
  };

  /**
   * Lee el formulario y devuelve la solicitud del contrato. El mensaje
   * contiene también los bloques de las casillas, así que las opciones
   * booleanas van desactivadas para no duplicarlas.
   *
   * @returns {{mensaje: string, nombre_host: boolean, fecha: boolean, hora: boolean, ip: boolean}}
   */
  const leer = () => ({
    mensaje: mensaje.value,
    nombre_host: false,
    fecha: false,
    hora: false,
    ip: false,
  });

  /** Repinta la vista previa local a partir del formulario. */
  const previsualizar = () => {
    const configuracion = leer();
    const valores = valoresPreview();
    const texto = configuracion.mensaje.replace(/\r\n/g, '\n');
    const lineas = texto.split('\n').map((linea) => renderizar(linea, valores));

    vistaPrevia.textContent = lineas.some((linea) => linea.trim()) ? lineas.join('\n') : '(banner vacío)';
  };

  /**
   * Vuelca una configuración en el formulario.
   *
   * @param {object} configuracion Configuración del contrato.
   */
  const volcar = (configuracion) => {
    confirmada = { ...confirmada, ...configuracion };

    let texto = configuracion.mensaje ?? '';

    for (const [clave, , bloque] of OPCIONES) {
      if (configuracion[clave]) {
        texto = agregarBloque(texto, bloque);
      }
    }

    mensaje.value = texto;
    sincronizarCasillas();
    previsualizar();
  };

  const cargar = async () => {
    inhabilitar(formulario, true);
    vistaPrevia.textContent = 'Consultando…';

    try {
      const [configuracion, hostname] = await Promise.all([
        api.issue.obtener(),
        api.hostname.obtener().catch(() => null),
      ]);

      contexto = {
        nombre: hostname?.nombre ?? '',
        ip: hostname?.ip ?? '',
      };

      volcar(configuracion);
    } catch (error) {
      vistaPrevia.textContent = mensajeDeError(error);
    } finally {
      inhabilitar(formulario, false);
    }
  };

  formulario.addEventListener('input', (evento) => {
    // Solo el textarea sincroniza las casillas. En un checkbox, el evento
    // input llega antes que change y desmarcaría la casilla antes de
    // insertar su bloque.
    if (evento.target === mensaje) {
      sincronizarCasillas();
    }

    previsualizar();
  });

  formulario.addEventListener('change', (evento) => {
    const opcion = OPCIONES.find(([clave]) => clave === evento.target.name);

    if (opcion && evento.target.type === 'checkbox') {
      const [, , bloque] = opcion;
      mensaje.value = evento.target.checked
        ? agregarBloque(mensaje.value, bloque)
        : quitarBloque(mensaje.value, bloque);
    }

    sincronizarCasillas();
    previsualizar();
  });
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
