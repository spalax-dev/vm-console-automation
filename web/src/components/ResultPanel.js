import { renderIcons } from '../icons.js';
import { escapar } from '../ui.js';

/**
 * Construye el panel lateral donde se muestra el resultado de la última
 * operación ejecutada.
 *
 * @returns {HTMLElement} Panel vacío listo para insertarse.
 */
export function ResultPanel() {
  const el = document.createElement('section');
  el.className = 'card flex flex-col h-full min-h-[280px]';
  el.innerHTML = `
    <header class="px-3 py-1.5 border-b border-vm-border flex items-center justify-between bg-[#f7f7f7]">
      <div class="flex items-center gap-1.5"><i data-lucide="info" class="w-4 h-4"></i><span class="text-[12px] font-medium">Resultado</span></div>
      <span id="result-estado" class="text-[11px] text-vm-fgSecondary">En espera</span>
    </header>
    <div id="result-body" class="flex-1 p-3 flex items-center justify-center text-center">
      <div>
        <div class="text-[12px] text-vm-fgSecondary">Sin operaciones ejecutadas aún</div>
        <div class="text-[11px] text-vm-fgSecondary mt-1">Selecciona un módulo para ejecutar tareas sobre la máquina.</div>
      </div>
    </div>
    <footer class="px-3 py-1 border-t border-vm-border text-[10px] text-vm-fgSecondary font-mono">
      <span id="result-pie">Código de salida: -</span>
    </footer>
  `;

  return el;
}

/**
 * Muestra el resultado de una operación en el panel lateral.
 *
 * @param {object} resultado Resultado de la operación.
 * @param {boolean} [resultado.ok] Indica si la operación terminó bien.
 * @param {string} [resultado.mensaje] Texto comprensible.
 * @param {string} [resultado.operacion] Orden ejecutada.
 * @param {number} [resultado.codigo] Código de salida del proceso remoto.
 * @param {string} [resultado.salida] Salida estándar.
 * @param {string} [resultado.stderr] Salida de error.
 * @param {string} [resultado.tecnico] Detalle técnico.
 */
export function showResult({ ok = true, mensaje = '', operacion = '', codigo = 0, salida = '', stderr = '', tecnico = '' } = {}) {
  const cuerpo = document.getElementById('result-body');
  const etiqueta = document.getElementById('result-estado');
  const pie = document.getElementById('result-pie');

  if (!cuerpo) {
    return;
  }

  cuerpo.className = 'flex-1 p-3 overflow-auto';
  cuerpo.innerHTML = `
    <div class="flex flex-col gap-2 fade-slide">
      <div class="flex items-start gap-2 px-2 py-1.5 border ${ok ? 'border-vm-border' : 'border-vm-accent bg-[#fdf1ef]'}">
        <i data-lucide="${ok ? 'check-circle-2' : 'x-circle'}" class="w-4 h-4 shrink-0 ${ok ? 'text-vm-success' : 'text-vm-accent'}"></i>
        <span class="text-[12px]">${escapar(mensaje || (ok ? 'Operación exitosa' : 'Error en la operación'))}</span>
      </div>
      <details class="border border-vm-border" ${ok ? '' : 'open'}>
        <summary class="px-2 py-1 cursor-pointer text-[11px] text-vm-fgSecondary">Detalle técnico</summary>
        <div class="px-2 py-2 flex flex-col gap-1 text-[11px]">
          <div><span class="text-vm-fgSecondary">Orden ejecutada:</span> <span class="font-mono break-all">${escapar(operacion) || '-'}</span></div>
          <div><span class="text-vm-fgSecondary">Código de salida:</span> <span class="font-mono">${escapar(codigo)}</span></div>
          <div><span class="text-vm-fgSecondary">Salida:</span></div>
          <pre class="bg-vm-bg border border-vm-border p-2 overflow-auto text-[11px] font-mono max-h-[140px] whitespace-pre-wrap">${escapar(salida) || '(vacío)'}</pre>
          <div><span class="text-vm-fgSecondary">stderr:</span></div>
          <pre class="bg-vm-bg border border-vm-border p-2 overflow-auto text-[11px] font-mono max-h-[140px] whitespace-pre-wrap">${escapar(stderr) || '(vacío)'}</pre>
          ${tecnico ? `<div><span class="text-vm-fgSecondary">Diagnóstico:</span> <span class="font-mono break-all">${escapar(tecnico)}</span></div>` : ''}
        </div>
      </details>
    </div>
  `;

  if (etiqueta) {
    etiqueta.textContent = ok ? 'Correcto' : 'Con error';
    etiqueta.className = ok ? 'text-[11px] text-vm-success' : 'text-[11px] text-vm-accent';
  }

  if (pie) {
    pie.textContent = `Código de salida: ${codigo ?? '-'}`;
  }

  renderIcons(cuerpo);
}
