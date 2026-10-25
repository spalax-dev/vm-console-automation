import { renderIcons } from '../icons.js';
import { api, guardarArchivo } from '../api.js';
import { showResult } from '../components/ResultPanel.js';
import { ejecutar, inhabilitar } from '../ui.js';

/** Nombre con el que se guarda la llave privada de root. */
const NOMBRE_ARCHIVO = 'id_ed25519_vmcons';

/**
 * Construye la vista de llaves SSH de root (numeral 3.1.2).
 *
 * @returns {HTMLElement} Vista lista para insertarse.
 */
export function renderLlaves() {
  const el = document.createElement('div');
  el.className = 'grid grid-cols-[58%_42%] gap-vm-gap items-start';
  el.innerHTML = `
    <div class="flex flex-col gap-vm-gap">
      <section class="card">
        <header class="px-3 py-1.5 border-b border-vm-border bg-[#f7f7f7]"><div class="flex items-center gap-1.5"><i data-lucide="key-round" class="w-4 h-4"></i><span class="text-[12px] font-medium">Par de llaves SSH para root</span></div></header>
        <div class="p-3 flex flex-col gap-3">
          <div class="flex justify-between items-center text-[12px]">
            <span class="text-vm-fgSecondary">Estado</span>
            <span id="llave-estado" class="px-1.5 py-0.5 border text-[11px] rounded-sm text-vm-fgSecondary border-vm-border bg-vm-bg">sin verificar</span>
          </div>
          <p class="text-[11px] text-vm-fgSecondary">
            La generación crea el par en el anfitrión, instala la llave pública en
            <span class="font-mono">/root/.ssh/authorized_keys</span> y deshabilita
            el acceso por contraseña.
          </p>
          <div class="flex gap-2">
            <button id="llave-generar" class="btn btn-primary"><i data-lucide="settings"></i> Generar par de llaves</button>
            <button id="llave-descargar" class="btn"><i data-lucide="download"></i> Descargar llave privada</button>
          </div>
        </div>
      </section>
      <section class="card border border-vm-accent">
        <div class="p-3 flex gap-2">
          <i data-lucide="alert-triangle" class="text-vm-accent shrink-0"></i>
          <div class="text-[12px]">Guarda la llave privada en un lugar seguro y no la compartas; quien la posea tendrá acceso root a la máquina.</div>
        </div>
      </section>
    </div>
    <div id="result-slot"></div>
  `;

  renderIcons(el);

  const estado = el.querySelector('#llave-estado');
  const generar = el.querySelector('#llave-generar');
  const descargar = el.querySelector('#llave-descargar');

  /**
   * Cambia la etiqueta del estado del par.
   *
   * @param {string} texto Texto visible.
   * @param {'ok'|'error'|'neutro'} tono Tono semántico.
   */
  const marcar = (texto, tono) => {
    estado.textContent = texto;
    estado.className =
      'px-1.5 py-0.5 border text-[11px] rounded-sm ' +
      (tono === 'ok'
        ? 'text-vm-success border-vm-success bg-[#e8f5e9]'
        : tono === 'error'
          ? 'text-vm-accent border-vm-accent bg-[#fdf1ef]'
          : 'text-vm-fgSecondary border-vm-border bg-vm-bg');
  };

  generar.addEventListener('click', async () => {
    inhabilitar(el, true);

    const exito = await ejecutar({
      accion: () => api.llaves.generarRoot(),
      operacion: 'ssh-keygen -t ed25519 + instalación en authorized_keys',
      informar: showResult,
    });

    inhabilitar(el, false);

    if (exito) {
      marcar('generada', 'ok');
    } else {
      marcar('sin verificar', 'neutro');
    }
  });

  descargar.addEventListener('click', async () => {
    inhabilitar(el, true);

    const exito = await ejecutar({
      accion: async () => {
        const archivo = await api.llaves.descargarRoot();
        guardarArchivo(archivo, NOMBRE_ARCHIVO);

        return { mensaje: `Se descargó la llave privada como «${NOMBRE_ARCHIVO}».` };
      },
      operacion: 'GET /api/keys/root',
      informar: showResult,
    });

    inhabilitar(el, false);

    marcar(exito ? 'generada' : 'no disponible', exito ? 'ok' : 'error');
  });

  return el;
}
