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
    <header class="px-3 py-1.5 border-b border-vm-border flex items-center justify-between bg-[#f7f7f7]">
      <div class="flex items-center gap-1.5">
        <i data-lucide="info" class="w-4 h-4"></i>
        <span class="text-[12px] font-medium">Resultado</span>
        <span id="result-cuenta" class="text-[10px] text-vm-fgSecondary"></span>
      </div>
      <button id="result-limpiar" type="button" class="btn h-vm-btn px-1.5" title="Borrar el historial de resultados">
        <i data-lucide="trash-2" class="w-4 h-4"></i>
        <span class="text-[11px]">Limpiar</span>
      </button>
    </header>
    <div id="result-body" class="flex-1 min-h-0 p-2 overflow-auto flex flex-col gap-2"></div>
    <footer class="px-3 py-1 border-t border-vm-border text-[10px] text-vm-fgSecondary font-mono">
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
    mensaje: mensaje || (ok ? 'Operación exitosa' : 'Error en la operación'),
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
    cuenta.textContent = historial.length > 0 ? `(${historial.length})` : '';
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

  cuerpo.innerHTML = historial.map(entrada).join('');
  renderIcons(cuerpo);
  cuerpo.scrollTop = cuerpo.scrollHeight;

  const ultimo = historial[historial.length - 1];

  if (pie) {
    pie.textContent = `${hora(ultimo.hora)} · código de salida: ${ultimo.codigo ?? '-'}`;
  }
}

/**
 * Construye la ficha de un resultado del historial.
 *
 * @param {object} resultado Resultado guardado.
 * @returns {string} HTML de la ficha.
 */
function entrada(resultado) {
  const correcto = resultado.ok !== false;

  return `
    <article class="border ${correcto ? 'border-vm-border' : 'border-vm-accent'} fade-slide">
      <div class="flex items-start gap-2 px-2 py-1.5 ${correcto ? '' : 'bg-[#fdf1ef]'}">
        <i data-lucide="${correcto ? 'check-circle-2' : 'x-circle'}" class="w-4 h-4 shrink-0 ${correcto ? 'text-vm-success' : 'text-vm-accent'}"></i>
        <div class="flex-1 min-w-0">
          <div class="flex items-start justify-between gap-2">
            <span class="text-[12px] break-words">${escapar(resultado.mensaje)}</span>
            <span class="text-[10px] font-mono text-vm-fgSecondary shrink-0">${hora(resultado.hora)}</span>
          </div>
        </div>
      </div>
      <details class="border-t border-vm-border" ${correcto ? '' : 'open'}>
        <summary class="px-2 py-1 cursor-pointer text-[11px] text-vm-fgSecondary">Detalle técnico</summary>
        <div class="px-2 py-2 flex flex-col gap-1 text-[11px]">
          <div><span class="text-vm-fgSecondary">Orden ejecutada:</span> <span class="font-mono break-all">${escapar(resultado.operacion) || '-'}</span></div>
          <div><span class="text-vm-fgSecondary">Código de salida:</span> <span class="font-mono">${escapar(resultado.codigo)}</span></div>
          <div><span class="text-vm-fgSecondary">Salida:</span></div>
          <pre class="bg-vm-bg border border-vm-border p-2 overflow-auto text-[11px] font-mono max-h-[120px] whitespace-pre-wrap">${escapar(resultado.salida) || '(vacío)'}</pre>
          <div><span class="text-vm-fgSecondary">stderr:</span></div>
          <pre class="bg-vm-bg border border-vm-border p-2 overflow-auto text-[11px] font-mono max-h-[120px] whitespace-pre-wrap">${escapar(resultado.stderr) || '(vacío)'}</pre>
          ${resultado.tecnico ? `<div><span class="text-vm-fgSecondary">Diagnóstico:</span> <span class="font-mono break-all">${escapar(resultado.tecnico)}</span></div>` : ''}
        </div>
      </details>
    </article>
  `;
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
