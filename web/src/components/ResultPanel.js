import { renderIcons } from '../icons.js';
import { escapar } from '../ui.js';
import { agregar, leer, limpiar } from '../resultados.js';

/**
 * Construye el panel lateral con el historial de resultados. El
 * contenido se lee del almacenamiento, así que sobrevive al cambio de
 * módulo y a la recarga de la página.
 *
 * @returns {HTMLElement} Panel listo para insertarse.
 */
export function ResultPanel() {
  const el = document.createElement('section');
  el.className = 'card flex flex-col min-h-[280px] max-h-[calc(100vh-8.5rem)] overflow-hidden';
  el.innerHTML = `
    <header class="px-3 py-1.5 border-b border-vm-border flex items-center justify-between bg-[#F9F9F9]">
      <div class="flex items-center gap-1.5">
        <i data-lucide="square-terminal" class="w-4 h-4"></i>
        <span class="text-[12px] font-semibold">Resultado</span>
        <span id="result-cuenta" class="text-[10px] text-vm-fgSecondary font-mono"></span>
      </div>
      <button id="result-limpiar" type="button" class="btn h-vm-btn px-1.5" title="Borrar el historial de resultados">
        <i data-lucide="trash-2" class="w-4 h-4"></i>
        <span class="text-[11px]">Limpiar</span>
      </button>
    </header>
    <div id="result-body" class="flex-1 min-h-0 p-2 overflow-auto flex flex-col gap-2 bg-[#F5F5F5]"></div>
    <footer class="px-3 py-1 border-t border-vm-border text-[10px] text-vm-fgSecondary font-mono bg-[#F9F9F9]">
      <span id="result-pie">Sin operaciones ejecutadas</span>
    </footer>
  `;

  el.querySelector('#result-limpiar').addEventListener('click', () => {
    limpiar();
    pintar(el);
  });

  pintar(el);

  return el;
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

  const historial = leer();

  if (cuenta) {
    cuenta.textContent = historial.length > 0 ? `${historial.length} op.` : '';
  }

  if (historial.length === 0) {
    cuerpo.innerHTML = `
      <div class="flex-1 flex items-center justify-center text-center">
        <div>
          <div class="text-[12px] text-vm-fgSecondary">Sin operaciones ejecutadas aún</div>
          <div class="text-[11px] text-vm-fgSecondary mt-1">Los resultados se conservan al cambiar de módulo.</div>
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
    pie.textContent = `${hora(ultimo.hora)} · rc=${ultimo.codigo ?? '-'}`;
  }
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
 * Construye la ficha de un resultado: banda de estado, terminal con la
 * salida y detalle técnico plegable.
 *
 * @param {object} resultado Resultado guardado.
 * @param {boolean} reciente Indica si es la última operación.
 * @returns {string} HTML de la ficha.
 */
function entrada(resultado, reciente) {
  const correcto = resultado.ok !== false;
  const banda = correcto
    ? 'bg-[#E8F5E9] border-b border-[#A5D6A7] text-[#2E7D32]'
    : 'bg-[#FDF1EF] border-b border-[#E0B4AE] text-[#C0392B]';
  const insignia = correcto ? 'bg-[#C8E6C9] text-[#1B5E20]' : 'bg-[#F5C6C0] text-[#7A1B12]';

  return `
    <article class="border border-vm-border rounded-[3px] overflow-hidden bg-vm-surface">
      <div class="px-2.5 py-1.5 flex items-center justify-between gap-2 ${banda}">
        <div class="flex items-start gap-1.5 min-w-0">
          <span class="text-[12px] leading-4">${correcto ? '✔' : '✕'}</span>
          <span class="text-[12px] font-semibold break-words">${escapar(resultado.mensaje)}</span>
        </div>
        <span class="text-[10px] px-1 font-mono rounded shrink-0 ${insignia}">rc=${escapar(resultado.codigo)}</span>
      </div>
      <pre class="px-2.5 py-2 bg-[#1E1E1E] text-[#D4D4D4] font-mono text-[11px] leading-relaxed overflow-auto max-h-[180px] whitespace-pre-wrap break-words">${terminal(resultado)}</pre>
      <details class="border-t border-vm-border" ${reciente ? 'open' : ''}>
        <summary class="px-2.5 py-1 cursor-pointer text-[11px] text-vm-fgSecondary bg-[#F9F9F9] flex items-center justify-between">
          <span>Detalle técnico</span>
          <span class="font-mono">${hora(resultado.hora)}</span>
        </summary>
        <div class="px-2.5 py-2 flex flex-col gap-1.5 text-[11px]">
          <div class="flex flex-col gap-0.5">
            <span class="text-vm-fgSecondary">Orden ejecutada</span>
            <span class="font-mono break-all border border-vm-border bg-vm-bg px-1.5 py-1">${escapar(resultado.operacion) || '—'}</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-vm-fgSecondary">Código de salida</span>
            <span class="font-mono px-1.5 py-0.5 border border-vm-border bg-vm-bg">${escapar(resultado.codigo)}</span>
          </div>
          <div class="flex flex-col gap-0.5">
            <span class="text-vm-fgSecondary">Salida estándar</span>
            <pre class="font-mono max-h-[120px] overflow-auto whitespace-pre-wrap break-words border border-vm-border bg-vm-bg px-1.5 py-1">${escapar(resultado.salida) || '(vacío)'}</pre>
          </div>
          <div class="flex flex-col gap-0.5">
            <span class="text-vm-fgSecondary">Salida de error</span>
            <pre class="font-mono max-h-[120px] overflow-auto whitespace-pre-wrap break-words border border-vm-border bg-vm-bg px-1.5 py-1">${escapar(resultado.stderr) || '(vacío)'}</pre>
          </div>
          ${
            resultado.tecnico
              ? `<div class="flex flex-col gap-0.5">
                   <span class="text-vm-fgSecondary">Diagnóstico</span>
                   <span class="font-mono break-all border border-vm-border bg-vm-bg px-1.5 py-1">${escapar(resultado.tecnico)}</span>
                 </div>`
              : ''
          }
        </div>
      </details>
    </article>
  `;
}

/**
 * Compone el cuerpo del terminal con la orden, la salida y el error.
 *
 * @param {object} resultado Resultado guardado.
 * @returns {string} HTML del terminal.
 */
function terminal(resultado) {
  const lineas = [`<span class="text-[#888888]">$ ${escapar(resultado.operacion) || 'operación'}</span>`];

  if (resultado.salida) {
    lineas.push(`<span class="text-[#AAAAAA]">${escapar(recortar(resultado.salida))}</span>`);
  }

  if (resultado.stderr) {
    lineas.push(`<span class="text-[#E06C75]">${escapar(recortar(resultado.stderr))}</span>`);
  }

  if (!resultado.salida && !resultado.stderr) {
    lineas.push('<span class="text-[#666666]">(sin salida)</span>');
  }

  return lineas.join('\n');
}

/**
 * Recorta una salida muy larga para el terminal; el texto completo queda
 * en el detalle técnico.
 *
 * @param {string} texto Salida del proceso.
 * @returns {string} Salida recortada.
 */
function recortar(texto) {
  const lineas = String(texto).trimEnd().split('\n');

  if (lineas.length <= 12) {
    return lineas.join('\n');
  }

  return `${lineas.slice(0, 12).join('\n')}\n… (${lineas.length - 12} líneas más)`;
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
