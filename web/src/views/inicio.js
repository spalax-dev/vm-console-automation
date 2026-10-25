import { renderIcons } from '../icons.js';
import { api } from '../api.js';
import { escapar, mensajeDeError, presentarEstado } from '../ui.js';

/** Módulos de la consola que se ofrecen en la portada. */
const MODULOS = [
  ['package', 'Paquetes', 'Gestión de paquetes APT/dpkg', '#/paquetes'],
  ['key-round', 'Llaves SSH', 'Gestión de authorized_keys', '#/llaves'],
  ['file-text', '/etc/issue', 'Banner pre-login del sistema', '#/issue'],
  ['server', 'Hostname', 'Identificador y FQDN del host', '#/hostname'],
  ['users', 'Usuarios', 'Cuentas locales y privilegios', '#/usuarios'],
  ['activity', 'systemd', 'Servicios, timers y unidades', '#/systemd'],
  ['git-merge', 'HAProxy', 'Balanceo y configuración L4/L7', '#/haproxy'],
  ['history', 'Historial', 'Registro de auditoría y tareas', '#/historial'],
];

/**
 * Construye la vista de inicio con el resumen de la sesión y el acceso a los
 * módulos.
 *
 * @returns {HTMLElement} Vista lista para insertarse.
 */
export function renderInicio() {
  const el = document.createElement('div');
  el.className = 'grid grid-cols-[58%_42%] gap-vm-gap items-start';
  el.innerHTML = `
    <div class="flex flex-col gap-vm-gap">
      <section class="card">
        <header class="px-3 py-1.5 border-b border-vm-border bg-[#f7f7f7] flex items-center justify-between">
          <div class="flex items-center gap-1.5"><i data-lucide="monitor" class="w-4 h-4"></i><span class="text-[12px] font-medium">Sesión con la máquina virtual</span></div>
          <button id="sesion-refresh" class="btn h-vm-btn px-1.5" title="Consultar de nuevo"><i data-lucide="refresh-cw" class="w-4 h-4"></i></button>
        </header>
        <div id="sesion-cuerpo" class="p-3 flex flex-col gap-1.5 text-[12px]">
          <div class="text-vm-fgSecondary">Consultando la sesión…</div>
        </div>
      </section>
      <section class="card">
        <header class="px-3 py-1.5 border-b border-vm-border bg-[#f7f7f7]"><div class="flex items-center gap-1.5"><i data-lucide="layout-grid" class="w-4 h-4"></i><span class="text-[12px] font-medium">Módulos</span></div></header>
        <div class="p-3 grid grid-cols-2 gap-2">
          ${MODULOS.map(([icono, titulo, descripcion, destino]) => `
            <a href="${destino}" class="border border-vm-border p-2 rounded-sm hover:border-[#C0392B] hover:bg-[#fdf5f4] transition-colors duration-100">
              <div class="flex items-center gap-1.5"><i data-lucide="${icono}"></i><span class="text-[12px] font-medium">${titulo}</span></div>
              <div class="text-[11px] text-vm-fgSecondary mt-0.5">${descripcion}</div>
            </a>
          `).join('')}
        </div>
      </section>
    </div>
    <div id="result-slot"></div>
  `;

  renderIcons(el);
  el.querySelector('#sesion-refresh').addEventListener('click', () => cargarSesion(el));
  cargarSesion(el);

  return el;
}

/**
 * Consulta `GET /api/session` y pinta el resumen de la conexión.
 *
 * @param {HTMLElement} el Vista de inicio.
 * @returns {Promise<void>}
 */
async function cargarSesion(el) {
  const cuerpo = el.querySelector('#sesion-cuerpo');
  const boton = el.querySelector('#sesion-refresh');

  if (!cuerpo) {
    return;
  }

  boton.disabled = true;

  try {
    const sesion = await api.sesion.obtener();
    const presentacion = presentarEstado(sesion.estado);
    const hora = new Date().toLocaleTimeString('es-CO', { hour12: false });

    cuerpo.innerHTML = `
      <div class="flex justify-between items-center">
        <span class="text-vm-fgSecondary">Máquina</span>
        <span class="font-mono">${escapar(sesion.vm) || 'sin vincular'}</span>
      </div>
      <div class="flex justify-between items-center">
        <span class="text-vm-fgSecondary">Estado</span>
        <span class="px-1.5 py-0.5 border text-[11px] rounded-sm ${presentacion.clase}">${presentacion.texto}</span>
      </div>
      <div class="flex justify-between items-center">
        <span class="text-vm-fgSecondary">Conector</span>
        <span class="flex items-center gap-1"><i data-lucide="key-round" class="w-4 h-4"></i>${escapar(sesion.conector)}</span>
      </div>
      <div class="flex justify-between items-center">
        <span class="text-vm-fgSecondary">Dirección IP</span>
        <span class="flex items-center gap-1.5">
          <span class="font-mono">${escapar(sesion.ip)}</span>
          <button id="copiar-ip" class="btn h-vm-btn px-1" title="Copiar la dirección"><i data-lucide="copy" class="w-4 h-4"></i></button>
        </span>
      </div>
      <div class="flex justify-between items-center">
        <span class="text-vm-fgSecondary">Última consulta</span>
        <span class="text-vm-fgSecondary font-mono">${hora}</span>
      </div>
    `;

    renderIcons(cuerpo);
    cuerpo.querySelector('#copiar-ip').addEventListener('click', () => navigator.clipboard?.writeText(sesion.ip));
  } catch (error) {
    cuerpo.innerHTML = `
      <div class="flex items-start gap-2 text-[12px]">
        <i data-lucide="alert-triangle" class="w-4 h-4 text-vm-accent shrink-0"></i>
        <div>
          <div class="text-vm-fg">No fue posible consultar la sesión</div>
          <div class="text-[11px] text-vm-fgSecondary mt-0.5">${escapar(mensajeDeError(error))}</div>
        </div>
      </div>
    `;
    renderIcons(cuerpo);
  } finally {
    boton.disabled = false;
  }
}
