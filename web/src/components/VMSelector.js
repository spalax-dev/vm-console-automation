import { renderIcons } from '../icons.js';
import { api } from '../api.js';
import { escapar, mensajeDeError, presentarEstado } from '../ui.js';

/** Milisegundos de espera antes de consultar mientras se escribe. */
const ESPERA_BUSQUEDA = 200;

/**
 * Construye el selector de máquina virtual de la barra superior.
 *
 * El botón muestra la máquina vinculada y abre un panel con un campo de
 * búsqueda y el inventario que devuelve `GET /api/vms`. Elegir una
 * máquina la vincula con `PUT /api/session` y avisa al llamador.
 *
 * @param {object} [opciones] Opciones del selector.
 * @param {(maquina: object) => void} [opciones.alVincular] Aviso tras vincular.
 * @returns {HTMLElement} Selector listo para insertarse.
 */
export function VMSelector({ alVincular } = {}) {
  const el = document.createElement('div');
  el.className = 'flex items-center gap-1.5';
  el.innerHTML = `
    <div class="relative">
      <button id="vm-boton" type="button" class="btn h-vm-btn pr-1.5 max-w-[260px]"
        aria-haspopup="listbox" aria-expanded="false" aria-controls="vm-panel" title="Elegir la máquina virtual (solo con Guest Additions)">
        <span id="vm-punto" class="w-1.5 h-1.5 rounded-full bg-vm-muted shrink-0"></span>
        <span id="vm-nombre" class="truncate">Buscando máquinas…</span>
        <i data-lucide="chevron-down" class="w-3.5 h-3.5 shrink-0"></i>
      </button>
      <div id="vm-panel" class="hidden absolute left-0 top-full mt-1 w-[330px] z-30 bg-vm-surface border border-vm-border shadow-none">
        <div class="flex items-center gap-1.5 px-2 h-vm-input border-b border-vm-border">
          <i data-lucide="search" class="w-4 h-4 text-vm-fgSecondary"></i>
          <input id="vm-busqueda" class="flex-1 bg-transparent outline-none text-[12px]" placeholder="Buscar por nombre o UUID…"
            autocomplete="off" spellcheck="false" aria-label="Buscar máquina virtual">
        </div>
        <ul id="vm-lista" role="listbox" aria-label="Máquinas virtuales" class="max-h-[280px] overflow-auto"></ul>
        <footer id="vm-pie" class="px-2 py-1 border-t border-vm-border text-[10px] text-vm-fgSecondary"></footer>
      </div>
    </div>
    <button id="vm-encender" type="button" class="btn h-vm-btn px-1.5 hidden" title="Encender la máquina virtual">
      <i data-lucide="power" class="w-4 h-4"></i>
    </button>
  `;

  renderIcons(el);

  const boton = el.querySelector('#vm-boton');
  const nombre = el.querySelector('#vm-nombre');
  const punto = el.querySelector('#vm-punto');
  const panel = el.querySelector('#vm-panel');
  const busqueda = el.querySelector('#vm-busqueda');
  const lista = el.querySelector('#vm-lista');
  const pie = el.querySelector('#vm-pie');
  const encender = el.querySelector('#vm-encender');

  /** Máquinas recibidas en la última consulta. */
  let maquinas = [];

  /** Identificador de la consulta en curso, para descartar respuestas viejas. */
  let turno = 0;

  /** Temporizador del antirrebote de la búsqueda. */
  let temporizador = 0;

  /** Máquina vinculada, según la última consulta. */
  let activa = null;

  /**
   * Refleja la máquina vinculada en el botón.
   *
   * @param {object|null} maquina Máquina vinculada.
   */
  const marcarActiva = (maquina) => {
    activa = maquina;

    if (!maquina) {
      nombre.textContent = 'Elegir máquina';
      punto.className = 'w-1.5 h-1.5 rounded-full bg-vm-muted shrink-0';
      encender.classList.add('hidden');

      return;
    }

    const presentacion = presentarEstado(maquina.estado);

    nombre.textContent = maquina.nombre;
    punto.className = `w-1.5 h-1.5 rounded-full ${presentacion.punto} shrink-0`;
    boton.title = `Máquina activa: ${maquina.nombre} (${presentacion.texto})`;

    // Solo se ofrece encender cuando la máquina vinculada no está en
    // ejecución: apagarla desde la consola queda fuera de su alcance.
    encender.classList.toggle('hidden', maquina.estado === 'running');
    encender.disabled = false;
  };

  /**
   * Pinta el inventario recibido.
   */
  const pintar = () => {
    if (maquinas.length === 0) {
      lista.innerHTML = `
        <li class="px-2 py-3 text-center text-[11px] text-vm-fgSecondary">
          No hay máquinas administrables que coincidan con la búsqueda
        </li>
      `;

      return;
    }

    lista.innerHTML = maquinas
      .map((maquina) => {
        const presentacion = presentarEstado(maquina.estado);

        return `
        <li>
          <button type="button" role="option" aria-selected="${maquina.activa}"
            class="w-full text-left px-2 py-1.5 flex items-center gap-2 hover:bg-[#fdf5f4] ${maquina.activa ? 'border-l-2 border-[#C0392B] bg-[#faf9f9]' : 'border-l-2 border-transparent'}"
            data-maquina="${escapar(maquina.nombre)}">
            <span class="w-1.5 h-1.5 rounded-full shrink-0 ${presentacion.punto}"></span>
            <span class="flex-1 min-w-0">
              <span class="block text-[12px] truncate">${escapar(maquina.nombre)}</span>
              <span class="block text-[10px] font-mono text-vm-fgSecondary truncate">${escapar(maquina.uuid)}</span>
            </span>
            <span class="text-[10px] text-vm-fgSecondary shrink-0">${presentacion.texto}</span>
            ${maquina.activa ? '<i data-lucide="check" class="w-3.5 h-3.5 text-[#C0392B] shrink-0"></i>' : ''}
          </button>
        </li>
      `;
      })
      .join('');

    renderIcons(lista);
  };

  /**
   * Consulta el inventario y pinta el resultado.
   *
   * @param {string} [filtro] Texto de búsqueda.
   */
  const cargar = async (filtro = '') => {
    const actual = ++turno;

    pie.textContent = 'Consultando…';
    pie.className = 'px-2 py-1 border-t border-vm-border text-[10px] text-vm-fgSecondary';

    try {
      const respuesta = await api.maquinas.listar(filtro);

      if (actual !== turno) {
        return;
      }

      maquinas = respuesta.maquinas ?? [];

      const vinculada = maquinas.find((maquina) => maquina.activa);
      marcarActiva(vinculada ?? activa);

      pintar();
      pie.textContent = `${maquinas.length} máquina(s) administrable(s) · solo con Guest Additions`;
      pie.className = 'px-2 py-1 border-t border-vm-border text-[10px] text-vm-fgSecondary';
    } catch (error) {
      if (actual !== turno) {
        return;
      }

      maquinas = [];
      lista.innerHTML = `
        <li class="px-2 py-3 text-center text-[11px] text-vm-accent">${escapar(mensajeDeError(error))}</li>
      `;
      pie.textContent = 'No se pudo leer el inventario';
      pie.className = 'px-2 py-1 border-t border-vm-border text-[10px] text-vm-accent';
    }
  };

  /** Abre el panel y carga el inventario completo. */
  const abrir = () => {
    panel.classList.remove('hidden');
    boton.setAttribute('aria-expanded', 'true');
    busqueda.value = '';
    busqueda.focus();
    cargar('');
  };

  /** Cierra el panel. */
  const cerrar = () => {
    panel.classList.add('hidden');
    boton.setAttribute('aria-expanded', 'false');
  };

  /**
   * Vincula la consola con la máquina elegida.
   *
   * @param {string} nombreMaquina Nombre de la máquina.
   * @returns {Promise<void>}
   */
  const vincular = async (nombreMaquina) => {
    pie.textContent = `Vinculando con ${nombreMaquina}…`;

    try {
      const maquina = await api.sesion.vincular(nombreMaquina);

      marcarActiva({ ...maquina, activa: true });
      cerrar();

      if (alVincular) {
        alVincular(maquina);
      }
    } catch (error) {
      pie.textContent = mensajeDeError(error);
      pie.className = 'px-2 py-1 border-t border-vm-border text-[10px] text-vm-accent';
    }
  };

  boton.addEventListener('click', () => {
    if (panel.classList.contains('hidden')) {
      abrir();
    } else {
      cerrar();
    }
  });

  encender.addEventListener('click', async () => {
    if (!activa) {
      return;
    }

    encender.disabled = true;
    pie.textContent = `Encendiendo ${activa.nombre}…`;

    try {
      const maquina = await api.maquinas.encender(activa.nombre);

      marcarActiva({ ...maquina, activa: true });
      pie.textContent = `${maquina.nombre} está en ejecución`;

      if (alVincular) {
        alVincular(maquina);
      }

      cargar('');
    } catch (error) {
      pie.textContent = mensajeDeError(error);
      pie.className = 'px-2 py-1 border-t border-vm-border text-[10px] text-vm-accent';
      panel.classList.remove('hidden');
      boton.setAttribute('aria-expanded', 'true');
      encender.disabled = false;
    }
  });

  busqueda.addEventListener('input', () => {
    window.clearTimeout(temporizador);
    temporizador = window.setTimeout(() => cargar(busqueda.value), ESPERA_BUSQUEDA);
  });

  busqueda.addEventListener('keydown', (evento) => {
    if (evento.key === 'Escape') {
      cerrar();
      boton.focus();
    }
  });

  lista.addEventListener('click', (evento) => {
    const opcion = evento.target.closest('[data-maquina]');

    if (opcion) {
      vincular(opcion.dataset.maquina);
    }
  });

  document.addEventListener('click', (evento) => {
    if (!el.contains(evento.target)) {
      cerrar();
    }
  });

  document.addEventListener('keydown', (evento) => {
    if (evento.key === 'Escape') {
      cerrar();
    }
  });

  cargar('');

  return el;
}
