import { renderIcons } from '../icons.js';
import { api, guardarArchivo } from '../api.js';
import { showResult } from '../components/ResultPanel.js';
import { ejecutar, escapar, filaTabla, inhabilitar, mensajeDeError } from '../ui.js';
import { validarNombreUsuario } from '../validacion.js';

/**
 * Construye la vista de cuentas locales (numeral 3.2).
 *
 * @returns {HTMLElement} Vista lista para insertarse.
 */
export function renderUsuarios() {
  const el = document.createElement('div');
  el.className = 'grid grid-cols-[58%_42%] gap-vm-gap';
  el.innerHTML = `
    <div class="flex flex-col gap-vm-gap">
      <section class="card">
        <header class="px-3 py-1.5 border-b border-vm-border bg-[#f7f7f7]"><div class="flex items-center gap-1.5"><i data-lucide="user-plus" class="w-4 h-4"></i><span class="text-[12px] font-medium">Crear usuario</span></div></header>
        <form id="form-crear" class="p-3 flex flex-col gap-2" novalidate>
          <div class="flex flex-col gap-1">
            <label class="label" for="usuario-nombre">Nombre de usuario</label>
            <input id="usuario-nombre" name="nombre" class="input font-mono" placeholder="jorge" autocomplete="off" spellcheck="false">
            <span id="usuario-error" class="text-[11px] text-vm-accent hidden"></span>
          </div>
          <div class="flex flex-col gap-1">
            <label class="label" for="usuario-password">Contraseña</label>
            <input id="usuario-password" name="password" type="password" class="input" autocomplete="new-password" placeholder="Opcional">
            <span class="text-[11px] text-vm-fgSecondary">Permite el login local y <span class="font-mono">sudo</span>; el acceso SSH sigue usando la llave.</span>
          </div>
          <label class="flex items-center gap-1.5 text-[12px]"><input type="checkbox" name="sudo"> Otorgar privilegios sudo</label>
          <label class="flex items-center gap-1.5 text-[12px]"><input type="checkbox" name="llaves"> Generar par de llaves e instalar la pública</label>
          <div class="flex gap-2">
            <button type="submit" class="btn btn-primary"><i data-lucide="play"></i> Crear usuario</button>
            <button type="button" id="usuario-descargar-nueva" class="btn hidden"><i data-lucide="download"></i> Descargar llave privada</button>
          </div>
        </form>
      </section>
      <section class="card">
        <header class="px-3 py-1.5 border-b border-vm-border bg-[#f7f7f7]"><div class="flex items-center gap-1.5"><i data-lucide="user-minus" class="w-4 h-4"></i><span class="text-[12px] font-medium">Eliminar usuario</span></div></header>
        <form id="form-eliminar" class="p-3 flex flex-col gap-2">
          <div class="flex flex-col gap-1">
            <label class="label" for="eliminar-nombre">Nombre de usuario</label>
            <select id="eliminar-nombre" class="input"></select>
          </div>
          <label class="flex items-center gap-1.5 text-[12px]"><input type="checkbox" name="archivos"> Eliminar también todos sus archivos</label>
          <button type="submit" class="btn btn-danger-outline w-fit"><i data-lucide="trash-2"></i> Eliminar usuario</button>
        </form>
      </section>
      <section class="card">
        <header class="px-3 py-1.5 border-b border-vm-border bg-[#f7f7f7] flex items-center justify-between">
          <div class="flex items-center gap-1.5"><i data-lucide="users" class="w-4 h-4"></i><span class="text-[12px] font-medium">Usuarios de la máquina</span></div>
          <div class="flex items-center gap-1.5">
            <span id="usuarios-cuenta" class="text-[11px] text-vm-fgSecondary"></span>
            <button id="usuarios-refresh" class="btn h-vm-btn px-1.5" title="Consultar de nuevo"><i data-lucide="refresh-cw" class="w-4 h-4"></i></button>
          </div>
        </header>
        <div class="overflow-auto max-h-[260px]">
          <table class="w-full text-[12px]">
            <thead class="bg-vm-bg">
              <tr>
                <th class="text-left px-3 py-1 font-medium text-vm-fgSecondary">Usuario</th>
                <th class="text-left px-3 py-1 font-medium text-vm-fgSecondary">sudo</th>
                <th class="text-left px-3 py-1 font-medium text-vm-fgSecondary">llaves</th>
                <th class="text-left px-3 py-1 font-medium text-vm-fgSecondary">UID</th>
                <th class="text-left px-3 py-1 font-medium text-vm-fgSecondary">Home</th>
                <th class="px-3 py-1"></th>
              </tr>
            </thead>
            <tbody id="usuarios-filas"></tbody>
          </table>
        </div>
      </section>
    </div>
    <div id="result-slot"></div>
  `;

  renderIcons(el);

  /** Última lista recibida del servidor. */
  let usuarios = [];

  const filas = el.querySelector('#usuarios-filas');
  const cuenta = el.querySelector('#usuarios-cuenta');
  const selectEliminar = el.querySelector('#eliminar-nombre');
  const formularioCrear = el.querySelector('#form-crear');
  const formularioEliminar = el.querySelector('#form-eliminar');

  /**
   * Rellena un `select` conservando la opción elegida.
   *
   * @param {HTMLSelectElement} select Lista desplegable.
   * @param {string} etiquetaVacia Texto de la opción vacía.
   */
  const llenarSelect = (select, etiquetaVacia) => {
    const elegido = select.value;

    select.innerHTML =
      `<option value="">${etiquetaVacia}</option>` +
      usuarios.map((usuario) => `<option value="${escapar(usuario.nombre)}">${escapar(usuario.nombre)}</option>`).join('');

    if (usuarios.some((usuario) => usuario.nombre === elegido)) {
      select.value = elegido;
    }
  };

  const pintar = () => {
    cuenta.textContent = String(usuarios.length);
    llenarSelect(selectEliminar, usuarios.length ? 'Seleccionar usuario' : 'Sin usuarios');

    if (usuarios.length === 0) {
      filas.innerHTML = filaTabla(6, 'No hay usuarios para mostrar');
      return;
    }

    filas.innerHTML = usuarios
      .map(
        (usuario) => `
        <tr class="border-t border-vm-border hover:bg-vm-bg">
          <td class="px-3 py-1.5 font-mono">${escapar(usuario.nombre)}</td>
          <td class="px-3 py-1.5">${usuario.sudo ? '<i data-lucide="check" class="w-4 h-4 text-vm-success"></i>' : '<span class="text-vm-fgSecondary">—</span>'}</td>
          <td class="px-3 py-1.5">${usuario.llaves ? '<i data-lucide="key-round" class="w-4 h-4 text-vm-fg"></i>' : '<span class="text-vm-fgSecondary">—</span>'}</td>
          <td class="px-3 py-1.5 font-mono text-vm-fgSecondary">${escapar(usuario.uid) || '—'}</td>
          <td class="px-3 py-1.5 font-mono text-vm-fgSecondary">${escapar(usuario.home) || '—'}</td>
          <td class="px-3 py-1.5 text-right whitespace-nowrap">
            ${usuario.llaves ? `<button class="btn h-vm-btn px-1" data-descargar="${escapar(usuario.nombre)}" title="Descargar llave privada"><i data-lucide="download" class="w-4 h-4"></i></button>` : ''}
            <button class="btn h-vm-btn px-1" data-eliminar="${escapar(usuario.nombre)}" title="Eliminar usuario"><i data-lucide="trash-2" class="w-4 h-4"></i></button>
          </td>
        </tr>
      `,
      )
      .join('');

    renderIcons(filas);
  };

  const cargar = async () => {
    filas.innerHTML = filaTabla(6, 'Consultando los usuarios…');

    try {
      const respuesta = await api.usuarios.listar();
      usuarios = respuesta.usuarios ?? [];
      pintar();
    } catch (error) {
      usuarios = [];
      cuenta.textContent = '';
      filas.innerHTML = filaTabla(6, mensajeDeError(error), 'text-vm-accent');
    }
  };

  /**
   * Descarga la llave privada de un usuario.
   *
   * @param {string} nombre Nombre del usuario.
   * @returns {Promise<void>}
   */
  const descargarLlave = async (nombre) => {
    await ejecutar({
      accion: async () => {
        const archivo = await api.usuarios.descargarLlave(nombre);
        guardarArchivo(archivo, `id_ed25519_${nombre}`);

        return {
          mensaje: `Se descargó la llave privada de «${nombre}».`,
          operacion: `GET /api/users/${nombre}/key`,
          salida: `Archivo descargado: id_ed25519_${nombre}`,
        };
      },
      operacion: `GET /api/users/${nombre}/key`,
      informar: showResult,
    });
  };

  /**
   * Elimina un usuario y recarga la lista.
   *
   * @param {string} nombre Nombre del usuario.
   * @param {boolean} archivos Si se elimina también el directorio personal.
   * @returns {Promise<void>}
   */
  const eliminarUsuario = async (nombre, archivos) => {
    if (!nombre) {
      return;
    }

    await ejecutar({
      accion: async () => {
        await api.usuarios.eliminar(nombre, { archivos });

        return { mensaje: `Se eliminó el usuario «${nombre}».` };
      },
      operacion: `DELETE /api/users/${nombre}${archivos ? '?files=true' : ''}`,
      informar: showResult,
      recargar: cargar,
    });
  };

  filas.addEventListener('click', (evento) => {
    const descargar = evento.target.closest('[data-descargar]');
    const eliminar = evento.target.closest('[data-eliminar]');

    if (descargar) {
      descargarLlave(descargar.dataset.descargar);
    }

    if (eliminar && window.confirm(`¿Eliminar el usuario «${eliminar.dataset.eliminar}»?`)) {
      eliminarUsuario(eliminar.dataset.eliminar, false);
    }
  });

  el.querySelector('#usuarios-refresh').addEventListener('click', cargar);

  formularioEliminar.addEventListener('submit', (evento) => {
    evento.preventDefault();

    const nombre = selectEliminar.value;

    if (!nombre) {
      showResult({ ok: false, mensaje: 'Selecciona el usuario que se va a eliminar.', operacion: 'DELETE /api/users/{name}' });
      return;
    }

    if (!window.confirm(`¿Eliminar el usuario «${nombre}»?`)) {
      return;
    }

    eliminarUsuario(nombre, formularioEliminar.elements.archivos.checked);
  });

  const campoNombre = el.querySelector('#usuario-nombre');
  const aviso = el.querySelector('#usuario-error');
  const botonDescarga = el.querySelector('#usuario-descargar-nueva');

  /** Usuario recién creado, si se generó su par de llaves. */
  let recienCreado = '';

  formularioCrear.addEventListener('input', () => {
    aviso.classList.add('hidden');
    botonDescarga.classList.add('hidden');
  });

  formularioCrear.addEventListener('submit', async (evento) => {
    evento.preventDefault();

    const nombre = campoNombre.value.trim();
    const password = el.querySelector('#usuario-password').value;
    const invalido = validarNombreUsuario(nombre);

    if (invalido) {
      aviso.textContent = invalido;
      aviso.classList.remove('hidden');
      campoNombre.focus();
      return;
    }

    const solicitud = {
      nombre,
      sudo: formularioCrear.elements.sudo.checked,
      llaves: formularioCrear.elements.llaves.checked,
      password,
    };

    inhabilitar(formularioCrear, true);

    const exito = await ejecutar({
      accion: async () => {
        await api.usuarios.crear(solicitud);

        return { mensaje: `Se creó el usuario «${nombre}».` };
      },
      operacion: `useradd ${nombre}${solicitud.sudo ? ' + grupo sudo' : ''}${solicitud.llaves ? ' + par de llaves' : ''}${solicitud.password ? ' + contraseña' : ''}`,
      informar: showResult,
      recargar: cargar,
    });

    inhabilitar(formularioCrear, false);

    if (exito) {
      recienCreado = solicitud.llaves ? nombre : '';
      botonDescarga.classList.toggle('hidden', !solicitud.llaves);
      formularioCrear.reset();
    }
  });

  botonDescarga.addEventListener('click', () => descargarLlave(recienCreado));

  cargar();

  return el;
}
