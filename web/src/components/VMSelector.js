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
        aria-haspopup="listbox" aria-expanded="false" aria-controls="vm-panel" title="Elegir la máquina virtual (registrada o con Guest Additions, red puente/host-only)">
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
    <button id="vm-olvidar" type="button" class="btn h-vm-btn px-1.5 hidden" title="Olvidar el registro de esta máquina">
      <i data-lucide="trash-2" class="w-4 h-4"></i>
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
  const olvidar = el.querySelector('#vm-olvidar');

  /** Máquinas recibidas en la última consulta. */
  let maquinas = [];

  /** Identificador de la consulta en curso, para descartar respuestas viejas. */
  let turno = 0;

  /** Temporizador del antirrebote de la búsqueda. */
  let temporizador = 0;

  /** Máquina vinculada, según la última consulta. */
  let activa = null;

  /** Clave de las máquinas no administrables que el usuario ocultó. */
  const CLAVE_OCULTAS = 'vmcons:maquinas-ocultas';

  /** Lee la preferencia de máquinas ocultas del navegador. */
  const leerOcultas = () => {
    try {
      return new Set(JSON.parse(window.localStorage.getItem(CLAVE_OCULTAS) ?? '[]'));
    } catch {
      return new Set();
    }
  };

  /** Guarda la preferencia de máquinas ocultas. */
  const guardarOcultas = () => {
    try {
      window.localStorage.setItem(CLAVE_OCULTAS, JSON.stringify([...ocultas]));
    } catch {
      // Sin almacenamiento disponible: la preferencia dura la sesión.
    }
  };

  const ocultas = leerOcultas();

  /** Indica si una máquina no administrable fue ocultada por el usuario. */
  const esOculta = (maquina) =>
    maquina.administrable === false && maquina.estado === 'running' && ocultas.has(maquina.nombre);

  /** Máquinas que se pintan en el selector. */
  const visibles = () => maquinas.filter((maquina) => !esOculta(maquina));

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
      olvidar.classList.add('hidden');

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

    // Olvidar el registro solo tiene sentido en una máquina registrada.
    olvidar.classList.toggle('hidden', !maquina.registrada);
  };

  /**
   * Pinta el inventario recibido.
   */
  const pintar = () => {
    const listaVisible = visibles();

    if (listaVisible.length === 0) {
      lista.innerHTML = `
        <li class="px-2 py-3 text-center text-[11px] text-vm-fgSecondary">
          ${maquinas.length > 0
            ? 'Todas las máquinas encendidas están ocultas por falta de Guest Additions'
            : 'No hay máquinas administrables que coincidan con la búsqueda'}
        </li>
      `;

      return;
    }

    lista.innerHTML = listaVisible
      .map((maquina) => {
        const presentacion = presentarEstado(maquina.estado);
        const administrable = maquina.administrable !== false;
        const aviso = maquina.advertencia || 'Guest Additions no está disponible';

        if (!administrable) {
          return `
          <li class="px-2 py-1.5 flex items-center gap-2 opacity-60 border-l-2 border-transparent">
            <span class="w-1.5 h-1.5 rounded-full shrink-0 ${presentacion.punto}"></span>
            <span class="flex-1 min-w-0">
              <span class="block text-[12px] truncate">${escapar(maquina.nombre)}</span>
              <span class="block text-[10px] font-mono text-vm-fgSecondary truncate">${escapar(maquina.uuid)}</span>
            </span>
            <span class="text-[10px] text-vm-accent shrink-0 max-w-[88px] truncate" title="${escapar(aviso)}">Sin Guest Additions</span>
            <button type="button" data-ocultar="${escapar(maquina.nombre)}"
              class="text-[10px] underline text-vm-accent shrink-0">Ocultar</button>
          </li>
        `;
        }

        return `
        <li>
          <button type="button" role="option" aria-selected="${maquina.activa}"
            ${administrable ? `data-maquina="${escapar(maquina.nombre)}"` : 'disabled aria-disabled="true"'}
            class="w-full text-left px-2 py-1.5 flex items-center gap-2 ${administrable ? 'hover:bg-[#fdf5f4]' : 'opacity-60 cursor-not-allowed'} ${maquina.activa ? 'border-l-2 border-[#C0392B] bg-[#faf9f9]' : 'border-l-2 border-transparent'}">
            <span class="w-1.5 h-1.5 rounded-full shrink-0 ${presentacion.punto}"></span>
            <span class="flex-1 min-w-0">
              <span class="block text-[12px] truncate">${escapar(maquina.nombre)}</span>
              <span class="block text-[10px] font-mono text-vm-fgSecondary truncate">${escapar(maquina.uuid)}</span>
            </span>
            ${maquina.registrada ? '' : '<span class="text-[10px] text-vm-accent shrink-0">Sin registrar</span>'}
            <span class="text-[10px] text-vm-fgSecondary shrink-0">${presentacion.texto}</span>
            ${maquina.activa ? '<i data-lucide="check" class="w-3.5 h-3.5 text-[#C0392B] shrink-0"></i>' : ''}
          </button>
        </li>
      `;
      })
      .join('');

    renderIcons(lista);
  };

  /** Actualiza el pie con el conteo y el acceso a las máquinas ocultas. */
  const actualizarPie = () => {
    const total = visibles().length;
    const cantidadOcultas = maquinas.filter(esOculta).length;

    if (cantidadOcultas > 0) {
      pie.innerHTML = `${total} máquina(s) · ${cantidadOcultas} oculta(s) por falta de Guest Additions
        <button id="vm-mostrar-ocultas" type="button" class="underline text-vm-accent">Mostrar</button>`;
      pie.className = 'px-2 py-1 border-t border-vm-border text-[10px] text-vm-fgSecondary';
      pie.querySelector('#vm-mostrar-ocultas')?.addEventListener('click', () => {
        ocultas.clear();
        guardarOcultas();
        pintar();
        actualizarPie();
      });

      return;
    }

    pie.textContent = `${total} máquina(s) · las encendidas sin Guest Additions muestran un aviso`;
    pie.className = 'px-2 py-1 border-t border-vm-border text-[10px] text-vm-fgSecondary';
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
      actualizarPie();
    } catch (error) {
      if (actual !== turno) {
        return;
      }

      const detalle = mensajeDeError(error);

      // Se conserva el último inventario válido: un fallo transitorio de
      // VBoxManage no debe vaciar el selector ni ocultar las máquinas.
      if (maquinas.length === 0) {
        lista.innerHTML = `
          <li class="px-2 py-3 text-center text-[11px] text-vm-accent">${escapar(detalle)}</li>
        `;
      }

      pie.textContent = maquinas.length === 0
        ? 'No se pudo leer el inventario'
        : `No se pudo actualizar el inventario: ${detalle}`;
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
      const vinculada = { ...maquina, activa: true };

      marcarActiva(vinculada);
      cerrar();

      // La máquina ya está vinculada en el servidor: se avisa de inmediato
      // para que la vista activa pida sus datos y, si falta el registro,
      // muestre el botón para abrir el modal cuando el usuario lo pulse.
      if (alVincular) {
        alVincular(vinculada);
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

  /** Modal activo del encendido. */
  let modal = null;

  /** Indica que el usuario cerró el modal y ya no se espera la sesión. */
  let cancelado = false;

  /** Cierra el modal de encendido. */
  const cerrarModal = () => {
    modal?.overlay.remove();
    modal = null;
  };

  /**
   * Abre el modal de progreso del encendido.
   *
   * @param {string} nombre Nombre de la máquina.
   * @returns {object} Referencias del modal.
   */
  const crearModal = (nombre) => {
    cerrarModal();
    cancelado = false;

    const overlay = document.createElement('div');
    overlay.id = 'vm-power-overlay';
    overlay.className = 'fixed inset-0 z-50 bg-black/40 flex items-center justify-center px-3';
    overlay.innerHTML = `
      <section class="card w-[440px] max-w-[94vw] p-3 flex flex-col gap-2" role="dialog" aria-modal="true" aria-labelledby="vm-power-titulo">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-1.5"><i data-lucide="power" class="w-4 h-4"></i><span id="vm-power-titulo" class="text-[13px] font-medium">Encendiendo ${escapar(nombre)}…</span></div>
          <button id="vm-power-cerrar" type="button" class="btn h-vm-btn px-1.5" title="Cerrar"><i data-lucide="x" class="w-4 h-4"></i></button>
        </div>
        <div class="h-1 w-full bg-vm-bg border border-vm-border overflow-hidden">
          <div class="barra-indeterminada h-full bg-vm-accent"></div>
        </div>
        <p id="vm-power-estado" class="text-[12px] text-vm-fgSecondary" aria-live="polite">La máquina está arrancando…</p>
        <p id="vm-power-motivo" class="text-[11px] text-vm-accent hidden"></p>
        <div id="vm-power-acciones" class="hidden gap-2 justify-end">
          <button id="vm-power-ocultar" type="button" class="btn">Ocultar de la lista</button>
          <button id="vm-power-mantener" type="button" class="btn btn-primary">Mantener en la lista</button>
        </div>
      </section>
    `;

    document.body.appendChild(overlay);
    renderIcons(overlay);

    const refs = {
      overlay,
      titulo: overlay.querySelector('#vm-power-titulo'),
      estado: overlay.querySelector('#vm-power-estado'),
      motivo: overlay.querySelector('#vm-power-motivo'),
      acciones: overlay.querySelector('#vm-power-acciones'),
      cerrar: overlay.querySelector('#vm-power-cerrar'),
      ocultar: overlay.querySelector('#vm-power-ocultar'),
      mantener: overlay.querySelector('#vm-power-mantener'),
    };

    refs.cerrar.addEventListener('click', () => {
      cancelado = true;
      cerrarModal();
    });

    refs.ocultar.addEventListener('click', () => {
      ocultas.add(nombre);
      guardarOcultas();
      cancelado = true;
      cerrarModal();
      pintar();
      actualizarPie();
    });

    refs.mantener.addEventListener('click', () => {
      refs.acciones.classList.add('hidden');
      refs.acciones.classList.remove('flex');
      refs.estado.textContent = 'La máquina seguirá en la lista. Puedes cerrar este aviso cuando termine de arrancar.';
    });

    modal = refs;

    return refs;
  };

  /**
   * Actualiza el texto y las acciones del modal.
   *
   * @param {{estado?: string, motivo?: string, acciones?: boolean}} cambio Cambio.
   */
  const actualizarModal = ({ estado, motivo, acciones = false }) => {
    if (!modal) {
      return;
    }

    if (estado) {
      modal.estado.textContent = estado;
    }

    if (motivo) {
      modal.motivo.textContent = motivo;
      modal.motivo.classList.remove('hidden');
    } else {
      modal.motivo.textContent = '';
      modal.motivo.classList.add('hidden');
    }

    modal.acciones.classList.toggle('hidden', !acciones);
    modal.acciones.classList.toggle('flex', acciones);
  };

  /**
   * Consulta el diagnóstico y devuelve el motivo por el que la máquina
   * no es administrable.
   *
   * @returns {Promise<string>} Detalle del diagnóstico o cadena vacía.
   */
  const motivoSinGuestAdditions = async () => {
    try {
      const diagnostico = await api.sesion.diagnostico();
      const problemas = diagnostico.problemas ?? [];
      const sinGuestAdditions = (diagnostico.guest_additions_run_level ?? 0) < 2;

      if (!sinGuestAdditions && problemas.length === 0) {
        return '';
      }

      return problemas.length > 0
        ? problemas.join(' ')
        : 'Guest Additions no está listo.';
    } catch {
      return '';
    }
  };

  /**
   * Espera a que la máquina arranque y la sesión esté disponible. El
   * aviso de Guest Additions no bloquea la espera: se muestra en el
   * modal y el sondeo continúa.
   *
   * @param {string} nombre Nombre de la máquina.
   * @returns {Promise<object|null>} Sesión disponible o null si expira.
   */
  const esperarArranque = async (nombre) => {
    const inicio = Date.now();
    const limite = 90_000;
    let avisoMostrado = false;

    while (Date.now() - inicio < limite) {
      if (cancelado) {
        return null;
      }

      try {
        const sesion = await api.sesion.obtener();

        if (sesion.estado === 'running') {
          return sesion;
        }
      } catch {
        // Guest Additions o SSH todavía no responden.
      }

      try {
        await cargar('');
      } catch {
        // El inventario se reintenta en el siguiente ciclo.
      }

      const transcurrido = Math.floor((Date.now() - inicio) / 1000);
      const actual = maquinas.find((maquina) => maquina.nombre === nombre);

      if (!avisoMostrado && transcurrido >= 30 && actual?.administrable === false) {
        const motivo = actual.advertencia || (await motivoSinGuestAdditions());

        avisoMostrado = true;
        actualizarModal({
          estado: `${nombre} sigue arrancando. Aún no se detecta Guest Additions; puedes esperar, ocultarla o mantenerla en la lista.`,
          motivo,
          acciones: true,
        });
      }

      if (!avisoMostrado) {
        if (transcurrido >= 30) {
          actualizarModal({ estado: 'Esperando Guest Additions o SSH…' });
        } else if (transcurrido >= 8) {
          actualizarModal({ estado: 'Esperando a que arranque el sistema…' });
        }
      }

      await new Promise((resolver) => window.setTimeout(resolver, 2000));
    }

    return null;
  };

  encender.addEventListener('click', async () => {
    if (!activa) {
      return;
    }

    encender.disabled = true;
    encender.innerHTML = '<i data-lucide="loader-2" class="w-4 h-4 animate-spin"></i>';
    renderIcons(encender);

    crearModal(activa.nombre);
    pie.textContent = 'La máquina está arrancando…';
    pie.className = 'px-2 py-1 border-t border-vm-border text-[10px] text-vm-fgSecondary';

    try {
      const maquina = await api.maquinas.encender(activa.nombre);

      marcarActiva({ ...maquina, activa: true });
      actualizarModal({ estado: `${maquina.nombre} encendida. Esperando la sesión…` });
      pie.textContent = `${maquina.nombre} está en ejecución; esperando Guest Additions o SSH…`;

      const sesion = await esperarArranque(maquina.nombre);

      if (sesion) {
        actualizarModal({ estado: `${maquina.nombre} en ejecución y sesión disponible.` });
        pie.textContent = `${maquina.nombre} está en ejecución`;

        if (alVincular) {
          await Promise.resolve(alVincular(maquina));
        }

        await cargar('');
        window.setTimeout(cerrarModal, 900);
      } else if (!cancelado) {
        const motivo = await motivoSinGuestAdditions();

        actualizarModal({
          estado: motivo
            ? `${maquina.nombre} está encendida, pero no se puede administrar todavía.`
            : `${maquina.nombre} está encendida, pero Guest Additions/SSH aún no responde.`,
          motivo,
          acciones: true,
        });
        pie.textContent = motivo
          ? `${maquina.nombre} está encendida; revisa el aviso de Guest Additions.`
          : 'La máquina está encendida; actualiza en unos segundos.';
        pie.className = motivo
          ? 'px-2 py-1 border-t border-vm-border text-[10px] text-vm-accent'
          : 'px-2 py-1 border-t border-vm-border text-[10px] text-vm-fgSecondary';
      }
    } catch (error) {
      actualizarModal({ estado: mensajeDeError(error), acciones: true });
      pie.textContent = mensajeDeError(error);
      pie.className = 'px-2 py-1 border-t border-vm-border text-[10px] text-vm-accent';
    } finally {
      encender.innerHTML = '<i data-lucide="power" class="w-4 h-4"></i>';
      renderIcons(encender);
      encender.disabled = false;
    }
  });

  olvidar.addEventListener('click', async () => {
    if (!activa) {
      return;
    }

    const nombreMaquina = activa.nombre;
    const confirmado = window.confirm(
      `¿Olvidar el registro de ${nombreMaquina}? Tendrás que registrar la máquina otra vez para administrarla.`,
    );

    if (!confirmado) {
      return;
    }

    olvidar.disabled = true;

    try {
      await api.sesion.olvidar();
      marcarActiva({ ...activa, registrada: false });
      await cargar('');
      pie.textContent = `Registro de ${nombreMaquina} olvidado; vuelve a registrarla para administrarla.`;
      pie.className = 'px-2 py-1 border-t border-vm-border text-[10px] text-vm-fgSecondary';
    } catch (error) {
      pie.textContent = mensajeDeError(error);
      pie.className = 'px-2 py-1 border-t border-vm-border text-[10px] text-vm-accent';
    } finally {
      olvidar.disabled = false;
    }
  });

  busqueda.addEventListener('input', () => {
    window.clearTimeout(temporizador);
    temporizador = window.setTimeout(() => cargar(busqueda.value), ESPERA_BUSQUEDA);
  });  busqueda.addEventListener('keydown', (evento) => {
    if (evento.key === 'Escape') {
      cerrar();
      boton.focus();
    }
  });

  lista.addEventListener('click', (evento) => {
    const ocultarBoton = evento.target.closest('[data-ocultar]');

    if (ocultarBoton) {
      ocultas.add(ocultarBoton.dataset.ocultar);
      guardarOcultas();
      pintar();
      actualizarPie();

      return;
    }

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
