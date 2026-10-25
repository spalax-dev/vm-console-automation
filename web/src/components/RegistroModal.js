import { renderIcons } from '../icons.js';
import { api } from '../api.js';
import { escapar, mensajeDeError } from '../ui.js';

/**
 * Abre el modal de registro de una máquina con Guest Additions.
 *
 * El usuario ingresa sus credenciales y la consola prepara SSH: instala
 * `openssh-server`, la llave de root y deja la máquina registrada.
 *
 * @param {object} opciones Opciones del modal.
 * @param {{nombre: string}} opciones.maquina Máquina por registrar.
 * @param {() => void} [opciones.alRegistrar] Aviso tras registrar.
 * @returns {void}
 */
export function abrirRegistro({ maquina, alRegistrar }) {
  const overlay = document.createElement('div');
  overlay.className = 'fixed inset-0 z-50 bg-black/30 flex items-center justify-center p-4';
  overlay.innerHTML = `
    <section class="card w-[400px] max-w-full" role="dialog" aria-modal="true" aria-labelledby="registro-titulo">
      <header class="px-3 py-1.5 border-b border-vm-border bg-[#f7f7f7] flex items-center justify-between">
        <span id="registro-titulo" class="text-[12px] font-medium">Registrar ${escapar(maquina.nombre)}</span>
        <button id="registro-cerrar" type="button" class="btn h-vm-btn px-1" title="Cerrar" aria-label="Cerrar">
          <i data-lucide="x" class="w-4 h-4"></i>
        </button>
      </header>
      <form id="registro-form" class="p-3 flex flex-col gap-2">
        <p class="text-[11px] text-vm-fgSecondary">
          La máquina tiene Guest Additions pero aún no está registrada. Ingresa un usuario con permisos
          para preparar SSH; las credenciales se guardan solo en este equipo.
        </p>
        <label class="flex flex-col gap-1">
          <span class="label">Usuario</span>
          <input name="usuario" value="root" class="input font-mono" autocomplete="username" required>
        </label>
        <label class="flex flex-col gap-1">
          <span class="label">Contraseña</span>
          <input name="password" type="password" class="input" autocomplete="current-password" required>
        </label>
        <div id="registro-error" class="text-[11px] text-vm-accent min-h-[14px]"></div>
        <footer class="flex justify-end gap-2 pt-1">
          <button id="registro-cancelar" type="button" class="btn">Cancelar</button>
          <button type="submit" class="btn btn-primary"><i data-lucide="key-round"></i> Registrar</button>
        </footer>
      </form>
    </section>
  `;

  renderIcons(overlay);

  const cerrar = () => overlay.remove();
  const form = overlay.querySelector('#registro-form');
  const errorEl = overlay.querySelector('#registro-error');
  const boton = form.querySelector('button[type=submit]');

  overlay.querySelector('#registro-cerrar').addEventListener('click', cerrar);
  overlay.querySelector('#registro-cancelar').addEventListener('click', cerrar);
  overlay.addEventListener('click', (evento) => {
    if (evento.target === overlay) {
      cerrar();
    }
  });
  overlay.addEventListener('keydown', (evento) => {
    if (evento.key === 'Escape') {
      cerrar();
    }
  });

  form.addEventListener('submit', async (evento) => {
    evento.preventDefault();
    errorEl.textContent = '';
    boton.disabled = true;

    const datos = new FormData(form);

    try {
      await api.sesion.registrar(String(datos.get('usuario') || ''), String(datos.get('password') || ''));
      cerrar();

      if (alRegistrar) {
        alRegistrar();
      }
    } catch (fallo) {
      errorEl.textContent = mensajeDeError(fallo);
      boton.disabled = false;
    }
  });

  document.body.appendChild(overlay);
  form.querySelector('input[name=usuario]').focus();
}
