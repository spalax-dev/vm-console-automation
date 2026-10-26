import { renderIcons } from '../icons.js';
import { api, guardarArchivo } from '../api.js';
import { showResult } from '../components/ResultPanel.js';
import { ejecutar, escapar, inhabilitar } from '../ui.js';

/** Nombre con el que se guarda la llave privada de root. */
const NOMBRE_ARCHIVO_ROOT = 'id_ed25519_vmcons';

/**
 * Construye la vista de llaves SSH (numeral 3.1.2 y 3.2.1).
 *
 * Permite generar y descargar el par de llaves de root —la llave de
 * conexión de la consola— o de cualquier usuario regular de la máquina.
 *
 * @returns {HTMLElement} Vista lista para insertarse.
 */
export function renderLlaves() {
  const el = document.createElement('div');
  el.className = 'grid grid-cols-[58%_42%] gap-vm-gap';
  el.innerHTML = `
    <div class="flex flex-col gap-vm-gap">
      <section class="card">
        <header class="px-3 py-1.5 border-b border-vm-border bg-[#f7f7f7]"><div class="flex items-center gap-1.5"><i data-lucide="key-round" class="w-4 h-4"></i><span id="llave-titulo" class="text-[12px] font-medium">Par de llaves SSH</span></div></header>
        <div class="p-3 flex flex-col gap-3">
          <div class="flex items-center gap-2">
            <label class="label shrink-0" for="llave-usuario">Usuario</label>
            <select id="llave-usuario" class="input flex-1"></select>
          </div>
          <span id="llave-usuarios-error" class="text-[11px] text-vm-accent hidden"></span>
          <div class="flex justify-between items-center text-[12px]">
            <span class="text-vm-fgSecondary">Estado</span>
            <span id="llave-estado" class="px-1.5 py-0.5 border text-[11px] rounded-sm text-vm-fgSecondary border-vm-border bg-vm-bg">sin verificar</span>
          </div>
          <p id="llave-descripcion" class="text-[11px] text-vm-fgSecondary"></p>
          <div class="flex gap-2">
            <button id="llave-generar" class="btn btn-primary"><i data-lucide="settings"></i> Generar par de llaves</button>
            <button id="llave-descargar" class="btn"><i data-lucide="download"></i> Descargar llave privada</button>
          </div>
        </div>
      </section>
      <section class="card border border-vm-accent">
        <div class="p-3 flex gap-2">
          <i data-lucide="alert-triangle" class="text-vm-accent shrink-0"></i>
          <div id="llave-aviso" class="text-[12px]"></div>
        </div>
      </section>
    </div>
    <div id="result-slot"></div>
  `;

  renderIcons(el);

  const titulo = el.querySelector('#llave-titulo');
  const select = el.querySelector('#llave-usuario');
  const errorUsuarios = el.querySelector('#llave-usuarios-error');
  const estado = el.querySelector('#llave-estado');
  const descripcion = el.querySelector('#llave-descripcion');
  const aviso = el.querySelector('#llave-aviso');
  const generar = el.querySelector('#llave-generar');
  const descargar = el.querySelector('#llave-descargar');

  /** Usuarios regulares de la máquina. */
  let usuarios = [];

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

  /** Rellena el selector con root y los usuarios regulares. */
  const llenarSelect = () => {
    const elegido = select.value;

    select.innerHTML =
      '<option value="root">root (conexión)</option>' +
      usuarios
        .map((usuario) => `<option value="${escapar(usuario.nombre)}">${escapar(usuario.nombre)}</option>`)
        .join('');

    if (elegido === 'root' || usuarios.some((usuario) => usuario.nombre === elegido)) {
      select.value = elegido;
    }
  };

  /** Actualiza título, ruta, estado y aviso según el usuario elegido. */
  const actualizar = () => {
    const seleccion = select.value || 'root';
    const usuario = usuarios.find((item) => item.nombre === seleccion);
    const esRoot = seleccion === 'root';
    const destino = esRoot
      ? '/root/.ssh/authorized_keys'
      : `/home/${seleccion}/.ssh/authorized_keys`;

    titulo.textContent = esRoot
      ? 'Par de llaves SSH para root (conexión)'
      : `Par de llaves SSH para ${seleccion}`;

    descripcion.innerHTML =
      'La generación crea el par en el anfitrión, instala la llave pública en ' +
      `<span class="font-mono">${escapar(destino)}</span> y deshabilita el acceso por contraseña.`;

    aviso.textContent = esRoot
      ? 'Guarda la llave privada en un lugar seguro y no la compartas; quien la posea tendrá acceso root a la máquina.'
      : `Guarda la llave privada en un lugar seguro y no la compartas; quien la posea tendrá acceso a la cuenta «${seleccion}».`;

    if (esRoot) {
      marcar('sin verificar', 'neutro');
    } else {
      marcar(usuario?.llaves ? 'generada' : 'no generada', usuario?.llaves ? 'ok' : 'neutro');
    }
  };

  /** Consulta los usuarios regulares de la máquina. */
  const cargarUsuarios = async () => {
    errorUsuarios.classList.add('hidden');

    try {
      const respuesta = await api.usuarios.listar();
      usuarios = respuesta.usuarios ?? [];
    } catch (error) {
      usuarios = [];
      errorUsuarios.textContent = 'No se pudieron listar los usuarios; solo está disponible root.';
      errorUsuarios.classList.remove('hidden');
    }

    llenarSelect();
    actualizar();
  };

  select.addEventListener('change', actualizar);

  generar.addEventListener('click', async () => {
    const seleccion = select.value || 'root';
    inhabilitar(el, true);

    const exito = await ejecutar({
      accion: () =>
        seleccion === 'root' ? api.llaves.generarRoot() : api.usuarios.generarLlave(seleccion),
      operacion:
        seleccion === 'root'
          ? 'ssh-keygen -t ed25519 + instalación en /root/.ssh/authorized_keys'
          : `ssh-keygen -t ed25519 + instalación en /home/${seleccion}/.ssh/authorized_keys`,
      informar: showResult,
    });

    inhabilitar(el, false);

    if (exito) {
      const usuario = usuarios.find((item) => item.nombre === seleccion);

      if (usuario) {
        usuario.llaves = true;
      }

      marcar('generada', 'ok');
    } else {
      actualizar();
    }
  });

  descargar.addEventListener('click', async () => {
    const seleccion = select.value || 'root';
    const archivo = seleccion === 'root' ? NOMBRE_ARCHIVO_ROOT : `id_ed25519_${seleccion}`;
    inhabilitar(el, true);

    const exito = await ejecutar({
      accion: async () => {
        const descarga =
          seleccion === 'root'
            ? await api.llaves.descargarRoot()
            : await api.usuarios.descargarLlave(seleccion);

        guardarArchivo(descarga, archivo);

        return {
          mensaje: `Se descargó la llave privada de «${seleccion}» como «${archivo}».`,
          operacion:
            seleccion === 'root'
              ? 'GET /api/keys/root'
              : `GET /api/users/${seleccion}/key`,
          salida: `Archivo descargado: ${archivo}`,
        };
      },
      operacion:
        seleccion === 'root' ? 'GET /api/keys/root' : `GET /api/users/${seleccion}/key`,
      informar: showResult,
    });

    inhabilitar(el, false);
    marcar(exito ? 'generada' : 'no disponible', exito ? 'ok' : 'error');
  });

  cargarUsuarios();

  return el;
}
