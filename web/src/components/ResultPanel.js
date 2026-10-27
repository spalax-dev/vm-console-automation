import { renderIcons } from '../icons.js';
import { escapar } from '../ui.js';
import { agregar, getActiveVM, leer, limpiar } from '../resultados.js';

/** Fondo del panel y color del texto, en tonos oscuros. */
const PANEL = 'bg-[#2B2B2B] border-[#3C3C3C]';
const TARJETA = 'bg-[#333333] border-[#404040]';
const TEXTO = 'text-[#E5E5E5]';
const TEXTO_TENUE = 'text-[#999999]';
const BLOQUE = 'bg-[#242424] border-[#404040] text-[#D4D4D4] scroll-oscuro';

/**
 * Construye el panel lateral con el historial de resultados. El
 * contenido se lee del almacenamiento, así que sobrevive al cambio de
 * módulo y a la recarga de la página.
 *
 * @returns {HTMLElement} Panel listo para insertarse.
 */
export function ResultPanel() {
  const el = document.createElement('section');
  el.className = `flex flex-col min-h-[280px] max-h-[calc(100vh-8rem)] overflow-hidden border rounded-[3px] sticky top-0 ${PANEL}`;
  el.innerHTML = `
    <header class="px-3 py-1.5 border-b flex items-center justify-between ${TARJETA}">
      <div class="flex items-center gap-1.5">
        <i data-lucide="info" class="w-4 h-4 ${TEXTO}"></i>
        <span class="text-[12px] font-medium ${TEXTO}">Resultado</span>
        <span id="result-cuenta" class="text-[10px] text-[#888888]"></span>
      </div>
      <button id="result-limpiar" type="button"
        class="h-vm-btn px-1.5 inline-flex items-center gap-1 rounded-[3px] border border-[#4A4A4A] bg-[#3A3A3A] text-[#E5E5E5] text-[11px] transition-colors duration-100 hover:border-vm-accent hover:bg-[#454545]"
        title="Borrar el historial de resultados">
        <i data-lucide="trash-2" class="w-4 h-4"></i>
        <span>Limpiar</span>
      </button>
    </header>
    <div id="result-body" class="flex-1 min-h-0 p-2 overflow-auto flex flex-col gap-2 scroll-oscuro ${PANEL}"></div>
    <footer class="px-3 py-1 border-t text-[10px] font-mono text-[#888888] ${TARJETA}">
      <span id="result-pie">Sin operaciones ejecutadas</span>
    </footer>
  `;

  renderIcons(el);

  el.querySelector('#result-limpiar').addEventListener('click', () => {
    limpiar();
    clearProgress();
    pintar(el);
  });

  if (typeof document !== 'undefined') {
    document.addEventListener('vmcons:progreso', (evento) => {
      showProgress(evento.detail);
    });
  }

  pintar(el);

  return el;
}

/**
 * Muestra una tarjeta temporal de avance mientras una operación se ejecuta.
 *
 * @param {object} [opciones] Parámetros del progreso.
 * @param {string} [opciones.operacion] Orden que se está ejecutando.
 * @param {string} [opciones.mensaje] Texto descriptivo opcional.
 */
export function showProgress({ operacion = '', mensaje = 'Operación en ejecución…' } = {}) {
  const cuerpo = document.querySelector('#result-body');
  const pie = document.querySelector('#result-pie');

  if (!cuerpo) {
    return;
  }

  const previo = cuerpo.querySelector('#result-progreso');
  if (previo) {
    previo.remove();
  }

  const vacio = cuerpo.querySelector('#result-vacio');
  if (vacio) {
    vacio.classList.add('hidden');
  }

  const el = document.createElement('article');
  el.id = 'result-progreso';
  el.className = 'border rounded-[3px] overflow-hidden shrink-0 bg-[#333333] border-[#3B82F6] fade-slide';
  el.innerHTML = `
    <div class="flex items-start gap-2 px-2.5 py-2 bg-[#253245]">
      <i data-lucide="loader-2" class="w-4 h-4 shrink-0 text-[#60A5FA] animate-spin mt-0.5"></i>
      <div class="flex-1 min-w-0">
        <div class="flex items-start justify-between gap-2">
          <span class="text-[12px] font-medium text-[#E5E5E5]">${escapar(mensaje)}</span>
          <span class="text-[10px] font-mono text-[#888888] shrink-0">${hora(new Date().toISOString())}</span>
        </div>
        ${operacion ? `<div class="text-[11px] font-mono text-[#93C5FD] break-all mt-1">${escapar(operacion)}</div>` : ''}
      </div>
    </div>
  `;

  cuerpo.appendChild(el);
  renderIcons(el);
  alFinal(cuerpo);

  if (pie) {
    pie.textContent = 'Ejecutando en la máquina virtual…';
  }
}

/**
 * Retira la tarjeta de progreso en caso de cancelarse o completarse.
 */
export function clearProgress() {
  if (typeof document === 'undefined') {
    return;
  }

  const progreso = document.querySelector('#result-progreso');
  if (progreso) {
    progreso.remove();
  }
}

/**
 * Añade un resultado al historial y repinta el panel.
 *
 * @param {object} resultado Resultado de la operación.
 * @param {boolean} [resultado.ok] Indica si la operación terminó bien.
 * @param {string} [resultado.mensaje] Texto comprensible.
 * @param {string} [resultado.operacion] Orden ejecutada.
 * @param {number} [resultado.codigo] Código de salida.
 * @param {string} [resultado.salida] Salida estándar.
 * @param {string} [resultado.stderr] Salida de error.
 * @param {string} [resultado.tecnico] Detalle técnico.
 */
export function showResult({
  ok = true,
  mensaje = '',
  operacion = '',
  codigo = 0,
  salida = '',
  stderr = '',
  tecnico = '',
} = {}) {
  clearProgress();

  agregar({
    ok,
    mensaje: mensaje || describir(ok, operacion),
    operacion,
    codigo,
    salida,
    stderr,
    tecnico,
    hora: new Date().toISOString(),
  });

  pintar();
}

/**
 * Repinta el historial en el panel y lo desplaza al último resultado.
 *
 * @param {HTMLElement|Document} [raiz] Contenedor donde está el panel.
 */
function pintar(raiz = document) {
  const cuerpo = raiz.querySelector('#result-body');
  const cuenta = raiz.querySelector('#result-cuenta');
  const pie = raiz.querySelector('#result-pie');

  if (!cuerpo) {
    return;
  }

  const vm = getActiveVM();
  const historial = leer().filter((resultado) => (resultado.vm ?? '') === vm);

  if (cuenta) {
    cuenta.textContent = historial.length > 0 ? `(${historial.length})` : '';
  }

  if (historial.length === 0) {
    cuerpo.innerHTML = `
      <div id="result-vacio" class="flex-1 shrink-0 flex items-center justify-center text-center">
        <div>
          <div class="text-[12px] text-[#AAAAAA]">Sin operaciones ejecutadas aún</div>
          <div class="text-[11px] text-[#777777] mt-1">Los resultados son de la máquina activa.</div>
        </div>
      </div>
    `;

    if (pie) {
      pie.textContent = 'Sin operaciones ejecutadas';
    }

    return;
  }

  cuerpo.innerHTML = historial
    .map((resultado, indice) => entrada(resultado, indice === historial.length - 1))
    .join('');
  renderIcons(cuerpo);
  alFinal(cuerpo);

  const ultimo = historial[historial.length - 1];

  if (pie) {
    pie.textContent = `${hora(ultimo.hora)} · código de salida: ${ultimo.codigo ?? '-'}`;
  }
}

/** Repinta el panel activo, si existe. */
export function refreshResultPanel() {
  pintar();
}

/**
 * Desplaza el historial hasta la última operación. Se hace dos veces: en
 * el momento y tras el siguiente cuadro, porque al montar el panel el
 * elemento todavía puede no estar en el documento y no tener medidas.
 *
 * @param {HTMLElement} cuerpo Contenedor del historial.
 */
function alFinal(cuerpo) {
  const bajar = () => {
    cuerpo.scrollTop = cuerpo.scrollHeight;
  };

  bajar();

  if (typeof requestAnimationFrame === 'function') {
    requestAnimationFrame(bajar);
  }
}

/**
 * Decide si una salida corresponde a `systemctl status` por sus cabeceras.
 *
 * @param {string} salida Texto capturado.
 * @returns {boolean} `true` si se puede pintar con el formato de systemd.
 */
function esSalidaSystemctl(salida) {
  return /(^|\n)\s*(Loaded|Active):\s/.test(salida);
}

/**
 * Pinta una salida de `systemctl status` con etiquetas y estados
 * resaltados. Todo el contenido se escapa antes de insertarse.
 *
 * @param {string} salida Texto capturado.
 * @returns {string} HTML seguro.
 */
function formatearSalidaSystemctl(salida) {
  return salida
    .split('\n')
    .map((linea) => {
      if (linea.trim() === '') {
        return '<div class="h-2"></div>';
      }

      const cabecera = linea.match(/^(\s*)●\s+(.*)$/);

      if (cabecera) {
        return `<div class="whitespace-pre-wrap">${cabecera[1]}<span class="text-vm-accent">●</span> <span class="font-medium text-[#E8E8E8]">${escapar(cabecera[2])}</span></div>`;
      }

      const journal = linea.match(/^(\s*)([A-Z][a-z]{2}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2})\s+(\S+)\s+([^:]+):\s*(.*)$/);

      if (journal) {
        return `<div class="whitespace-pre-wrap">${journal[1]}<span class="text-vm-fgSecondary">${escapar(journal[2])}</span> <span class="text-vm-fgSecondary">${escapar(journal[3])}</span> <span class="text-vm-fgSecondary">${escapar(journal[4])}:</span> ${resaltarEstado(journal[5])}</div>`;
      }

      const propiedad = linea.match(/^(\s*)([A-Z][A-Za-z0-9 ]*?):(.*)$/);

      if (propiedad) {
        return `<div class="whitespace-pre-wrap">${propiedad[1]}<span class="text-vm-fgSecondary">${escapar(propiedad[2])}:</span>${resaltarEstado(propiedad[3])}</div>`;
      }

      return `<div class="whitespace-pre-wrap">${resaltarEstado(linea)}</div>`;
    })
    .join('');
}

/**
 * Colorea los estados habituales de systemd sin alterar el texto.
 *
 * @param {string} texto Fragmento de una línea.
 * @returns {string} HTML seguro.
 */
function resaltarEstado(texto) {
  const clases = {
    active: 'text-vm-success',
    running: 'text-vm-success',
    loaded: 'text-vm-success',
    enabled: 'text-vm-success',
    listening: 'text-vm-success',
    plugged: 'text-vm-success',
    mounted: 'text-vm-success',
    exited: 'text-vm-success',
    inactive: 'text-vm-warning',
    dead: 'text-vm-warning',
    disabled: 'text-vm-warning',
    static: 'text-vm-warning',
    waiting: 'text-vm-warning',
    skipped: 'text-vm-warning',
    activating: 'text-vm-warning',
    deactivating: 'text-vm-warning',
    reloading: 'text-vm-warning',
    failed: 'text-vm-accent',
    error: 'text-vm-accent',
  };

  return texto
    .split(/(\b(?:active|running|loaded|enabled|listening|plugged|mounted|exited|inactive|dead|disabled|static|waiting|skipped|activating|deactivating|reloading|failed|error)\b)/gi)
    .map((parte) => {
      const clase = clases[parte.toLowerCase()];

      return clase ? `<span class="${clase}">${escapar(parte)}</span>` : escapar(parte);
    })
    .join('');
}

/**
 * Elige entre la salida formateada de systemd y un bloque de texto plano.
 *
 * @param {string} salida Texto capturado.
 * @returns {string} HTML seguro.
 */
function salidaHtml(salida) {
  if (esSalidaSystemctl(salida)) {
    return `<div class="salida-systemctl border p-2 overflow-auto text-[11px] font-mono max-h-[180px] whitespace-pre-wrap ${BLOQUE}">${formatearSalidaSystemctl(salida)}</div>`;
  }

  return `<pre class="border p-2 overflow-auto text-[11px] font-mono max-h-[120px] whitespace-pre-wrap ${BLOQUE}">${escapar(salida)}</pre>`;
}

/**
 * Construye la ficha de un resultado del historial.
 *
 * @param {object} resultado Resultado guardado.
 * @param {boolean} reciente Indica si es la última operación.
 * @returns {string} HTML de la ficha.
 */
function entrada(resultado, reciente) {
  const correcto = resultado.ok !== false;
  const marco = correcto ? TARJETA : 'bg-[#3A2A28] border-vm-accent';
  const franja = correcto ? '' : 'bg-[#3A2A28]';

  return `
    <article class="border rounded-[3px] overflow-hidden shrink-0 ${marco} fade-slide">
      <div class="flex items-start gap-2 px-2 py-1.5 ${franja}">
        <i data-lucide="${correcto ? 'check-circle-2' : 'x-circle'}" class="w-4 h-4 shrink-0 ${correcto ? 'text-vm-success' : 'text-vm-accent'}"></i>
        <div class="flex-1 min-w-0">
          <div class="flex items-start justify-between gap-2">
            <span class="text-[12px] break-words ${TEXTO}">${escapar(resultado.mensaje)}</span>
            <span class="text-[10px] font-mono text-[#888888] shrink-0">${hora(resultado.hora)}</span>
          </div>
        </div>
      </div>
      <details class="border-t border-[#404040]" ${reciente ? 'open' : ''}>
        <summary class="px-2 py-1 cursor-pointer text-[11px] ${TEXTO_TENUE}">Detalle técnico</summary>
        <div class="px-2 py-2 flex flex-col gap-1 text-[11px] ${TEXTO}">
          <div><span class="${TEXTO_TENUE}">Orden ejecutada:</span> <span class="font-mono break-all">${escapar(resultado.operacion) || '-'}</span></div>
          <div><span class="${TEXTO_TENUE}">Código de salida:</span> <span class="font-mono">${escapar(resultado.codigo)}</span></div>
          ${
            resultado.salida
              ? `<div><span class="${TEXTO_TENUE}">Salida:</span></div>${salidaHtml(resultado.salida)}`
              : ''
          }
          ${
            resultado.stderr
              ? `<div><span class="${TEXTO_TENUE}">stderr:</span></div>
                 <pre class="border p-2 overflow-auto text-[11px] font-mono max-h-[120px] whitespace-pre-wrap ${BLOQUE}">${escapar(resultado.stderr)}</pre>`
              : ''
          }
          ${
            !resultado.salida && !resultado.stderr
              ? `<div class="${TEXTO_TENUE}">La orden no produjo salida.</div>`
              : ''
          }
          ${
            resultado.tecnico
              ? `<div><span class="${TEXTO_TENUE}">Diagnóstico:</span> <span class="font-mono break-all">${escapar(resultado.tecnico)}</span></div>`
              : ''
          }
        </div>
      </details>
    </article>
  `;
}

/**
 * Compone un mensaje concreto cuando la operación no trajo uno.
 *
 * @param {boolean} correcto Indica si la operación terminó bien.
 * @param {string} operacion Orden ejecutada.
 * @returns {string} Mensaje comprensible.
 */
function describir(correcto, operacion) {
  const orden = operacion ? ` «${operacion}»` : '';

  return correcto ? `Se completó la orden${orden}.` : `Falló la orden${orden}.`;
}

/**
 * Da formato a la hora de un resultado.
 *
 * @param {string} [iso] Marca de tiempo en ISO 8601.
 * @returns {string} Hora local `HH:MM:SS`.
 */
function hora(iso) {
  if (!iso) {
    return '';
  }

  const fecha = new Date(iso);

  if (Number.isNaN(fecha.getTime())) {
    return '';
  }

  return fecha.toLocaleTimeString('es-CO', { hour12: false });
}
