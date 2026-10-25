import { renderIcons } from '../icons.js';
import { api } from '../api.js';
import { escapar, mensajeDeError, presentarEstado } from '../ui.js';

/**
 * Nombre de la VM configurada en el arranque de la interfaz.
 *
 * El contrato todavía no expone la lista de máquinas de VirtualBox; hasta
 * entonces la barra superior muestra el nombre indicado por la variable
 * `VITE_VM_NOMBRE` y el estado real que devuelve `GET /api/session`.
 */
const VM_CONFIGURADA = import.meta.env.VITE_VM_NOMBRE || 'máquina activa';

/**
 * Construye la barra superior de la consola: identidad, máquina activa y
 * estado de la sesión.
 *
 * @returns {HTMLElement} Barra superior lista para insertarse.
 */
export function Header() {
  const el = document.createElement('header');
  el.className = 'h-vm-header bg-vm-surface border-b border-vm-border flex items-center justify-between px-3';
  el.innerHTML = `
    <div class="flex items-center gap-4">
      <div class="flex items-center gap-2.5 pr-4 border-r border-vm-border">
        <span class="w-2.5 h-2.5 rounded-full bg-vm-accent animate-pulse"></span>
        <span class="font-semibold text-vm-fg tracking-tight text-[16px]">vmcons</span>
      </div>
      <div class="flex items-center gap-2.5 text-[12px]">
        <select id="vm-select" class="input h-vm-btn pr-6">
          <option value="actual">VM activa: ${escapar(VM_CONFIGURADA)}</option>
        </select>
        <span class="h-4 w-px bg-vm-border"></span>
        <span id="vm-ip" class="font-mono text-vm-fg">—</span>
      </div>
    </div>
    <div class="flex items-center gap-3 text-[12px]">
      <span class="flex items-center gap-1.5" title="Estado de la sesión">
        <span id="vm-punto" class="w-1.5 h-1.5 rounded-full bg-vm-muted"></span>
        <span id="vm-estado" class="text-vm-fgSecondary">Consultando…</span>
      </span>
      <button id="btn-refresh" class="btn h-vm-btn px-1.5" title="Consultar la sesión">
        <i data-lucide="refresh-cw" class="w-4 h-4"></i>
      </button>
    </div>
  `;

  renderIcons(el);

  const actualizar = () => consultarSesion(el);

  el.querySelector('#btn-refresh').addEventListener('click', actualizar);

  actualizar();

  return el;
}

/**
 * Consulta `GET /api/session` y refleja el resultado en la barra superior.
 *
 * @param {HTMLElement} el Barra superior.
 * @returns {Promise<void>}
 */
async function consultarSesion(el) {
  const ip = el.querySelector('#vm-ip');
  const punto = el.querySelector('#vm-punto');
  const estado = el.querySelector('#vm-estado');
  const boton = el.querySelector('#btn-refresh');

  if (!ip || !punto || !estado) {
    return;
  }

  boton.disabled = true;
  estado.textContent = 'Consultando…';
  estado.className = 'text-vm-fgSecondary';
  punto.className = 'w-1.5 h-1.5 rounded-full bg-vm-muted';

  try {
    const sesion = await api.sesion.obtener();
    const presentacion = presentarEstado(sesion.estado);

    ip.textContent = sesion.ip || '—';
    punto.className = `w-1.5 h-1.5 rounded-full ${presentacion.punto}`;
    estado.textContent = presentacion.texto;
    estado.className = sesion.estado === 'running' ? 'text-vm-fg' : 'text-vm-fgSecondary';
    estado.title = `Conector: ${sesion.conector}`;
  } catch (error) {
    ip.textContent = '—';
    punto.className = 'w-1.5 h-1.5 rounded-full bg-vm-accent';
    estado.textContent = 'Sin sesión';
    estado.className = 'text-vm-accent';
    estado.title = mensajeDeError(error);
  } finally {
    boton.disabled = false;
  }
}
