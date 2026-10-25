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
    <div id="result-body" class="flex-1 p-3 flex items-center justify-center text-center tty">
      <div>
        <div class="text-[12px] tty-muted">Sin operaciones ejecutadas aún</div>
        <div class="text-[11px] tty-muted mt-1">Selecciona un módulo para ejecutar tareas sobre la máquina.</div>
      </div>
    </div>
    <footer class="px-3 py-1 border-t border-[#333] text-[10px] tty-muted font-mono tty">
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
 * @param {string} [resultado.solicitud] Identificador de la solicitud.
 */
export function showResult({ ok = true, mensaje = '', operacion = '', codigo = 0, salida = '', stderr = '', tecnico = '', solicitud = '' } = {}) {
  const cuerpo = document.getElementById('result-body');
  const etiqueta = document.getElementById('result-estado');
  const pie = document.getElementById('result-pie');

  if (!cuerpo) {
    return;
  }

  cuerpo.className = 'flex-1 p-3 overflow-auto tty';
  cuerpo.innerHTML = `
    <div class="flex flex-col gap-2 fade-slide">
      <div class="flex items-start gap-2 px-2 py-1.5 border ${ok ? 'border-[#3a3a3a] bg-[#242424]' : 'border-vm-accent bg-[#2a1a1a]'}">
        <i data-lucide="${ok ? 'check-circle-2' : 'x-circle'}" class="w-4 h-4 shrink-0 ${ok ? 'text-vm-success' : 'text-vm-accent'}"></i>
        <span class="text-[12px] text-[#e6e6e6]">${escapar(mensaje || (ok ? 'Operación exitosa' : 'Error en la operación'))}</span>
      </div>
      <details class="border border-[#3a3a3a]" ${ok ? '' : 'open'}>
        <summary class="px-2 py-1 cursor-pointer text-[11px] tty-muted">Detalle técnico</summary>
        <div class="px-2 py-2 flex flex-col gap-1 text-[11px]">
          <div><span class="tty-muted">Orden ejecutada:</span> <span class="font-mono break-all">${escapar(operacion) || '-'}</span></div>
          <div><span class="tty-muted">Código de salida:</span> <span class="font-mono">${escapar(codigo)}</span></div>
          <div><span class="tty-muted">Salida:</span></div>
          <pre class="bg-[#161616] border border-[#333] p-2 overflow-auto text-[11px] font-mono max-h-[140px] whitespace-pre-wrap text-[#d6d6d6]">${escapar(salida) || '(vacío)'}</pre>
          <div><span class="tty-muted">stderr:</span></div>
          <pre class="bg-[#161616] border border-[#333] p-2 overflow-auto text-[11px] font-mono max-h-[140px] whitespace-pre-wrap text-[#d6d6d6]">${escapar(stderr) || '(vacío)'}</pre>
          ${tecnico ? `<div><span class="tty-muted">Diagnóstico:</span> <span class="font-mono break-all">${escapar(tecnico)}</span></div>` : ''}
          ${solicitud ? `<div><span class="tty-muted">Solicitud:</span> <span class="font-mono">${escapar(solicitud)}</span></div>` : ''}
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
