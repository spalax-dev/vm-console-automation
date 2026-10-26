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
        <div id="vm-progreso" class="hidden flex-col gap-1 px-2 py-1.5 border-t border-vm-border" aria-live="polite">
          <div class="flex items-center justify-between text-[10px] text-vm-fgSecondary">
            <span id="vm-progreso-texto">Encendiendo la máquina…</span>
            <i data-lucide="loader-2" class="w-3 h-3 text-vm-accent animate-spin"></i>
          </div>
          <div class="h-1 w-full bg-vm-bg border border-vm-border overflow-hidden">
            <div class="barra-indeterminada h-full bg-vm-accent"></div>
          </div>
        </div>
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
  const progreso = el.querySelector('#vm-progreso');
  const progresoTexto = el.querySelector('#vm-progreso-texto');

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

  /**
   * Espera a que la máquina arranque y la sesión esté disponible.
   *
   * @param {string} nombre Nombre de la máquina.
   * @returns {Promise<object|null>} Sesión disponible o null si expira.
   */
  const esperarArranque = async (nombre) => {
    const inicio = Date.now();
    const limite = 90_000;
    let avisoMostrado = false;

    /**
     * Consulta el diagnóstico y devuelve el motivo por el que la máquina
     * no es administrable, si Guest Additions no está listo.
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

    while (Date.now() - inicio < limite) {
      await new Promise((resolver) => window.setTimeout(resolver, 2000));

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

      if (!avisoMostrado && actual?.administrable === false && !ocultas.has(nombre) && transcurrido >= 12) {
        const motivo = actual.advertencia || (await motivoSinGuestAdditions()) || 'Guest Additions no está disponible';
        const ocultar = window.confirm(
          `No es posible conectarse con ${nombre} porque no tiene Guest Additions.\n\n` +
          'Aceptar: ocultar la máquina de la lista.\nCancelar: mantenerla en la lista.',
        );

        avisoMostrado = true;

        if (ocultar) {
          ocultas.add(nombre);
        } else {
          ocultas.delete(nombre);
        }

        guardarOcultas();
        await cargar('');

        progresoTexto.textContent = ocultar
          ? `${nombre} no es administrable: ${motivo}. Se ocultó de la lista.`
          : `${nombre} no es administrable: ${motivo}. Se mantiene en la lista.`;
        pie.textContent = ocultar
          ? `${nombre} está encendida, pero no tiene Guest Additions; se ocultó de la lista.`
          : `${nombre} está encendida, pero no tiene Guest Additions; se mantiene con el aviso.`;
        pie.className = 'px-2 py-1 border-t border-vm-border text-[10px] text-vm-accent';
      }

      if (!avisoMostrado) {
        if (transcurrido >= 30) {
          progresoTexto.textContent = 'Esperando Guest Additions o SSH…';
        } else if (transcurrido >= 8) {
          progresoTexto.textContent = 'Esperando a que arranque el sistema…';
        }
      }
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

    // El progreso vive dentro del panel: hay que abrirlo para que se vea.
    panel.classList.remove('hidden');
    boton.setAttribute('aria-expanded', 'true');
    progreso.classList.remove('hidden');
    progreso.classList.add('flex');
    progresoTexto.textContent = `Encendiendo ${activa.nombre}…`;
    pie.textContent = 'La máquina está arrancando…';
    pie.className = 'px-2 py-1 border-t border-vm-border text-[10px] text-vm-fgSecondary';

    try {
      const maquina = await api.maquinas.encender(activa.nombre);

      marcarActiva({ ...maquina, activa: true });
      progresoTexto.textContent = `${maquina.nombre} encendida. Esperando la sesión…`;
      pie.textContent = `${maquina.nombre} está en ejecución; esperando Guest Additions o SSH…`;

      const sesion = await esperarArranque(maquina.nombre);

      if (sesion) {
        progresoTexto.textContent = `${maquina.nombre} en ejecución y sesión disponible.`;
        pie.textContent = `${maquina.nombre} está en ejecución`;

        if (alVincular) {
          await Promise.resolve(alVincular(maquina));
        }

        await cargar('');
        await new Promise((resolver) => window.setTimeout(resolver, 800));
      } else {
        const motivo = await motivoSinGuestAdditions();
        progresoTexto.textContent = motivo
          ? `${maquina.nombre} no es administrable: ${motivo}`
          : `${maquina.nombre} está encendida, pero Guest Additions/SSH aún no responde.`;
        pie.textContent = motivo
          ? `${maquina.nombre} está encendida, pero no tiene Guest Additions disponibles.`
          : 'La máquina está encendida; actualiza en unos segundos.';
        pie.className = motivo
          ? 'px-2 py-1 border-t border-vm-border text-[10px] text-vm-accent'
          : 'px-2 py-1 border-t border-vm-border text-[10px] text-vm-fgSecondary';
        panel.classList.remove('hidden');
        boton.setAttribute('aria-expanded', 'true');
        encender.disabled = false;
      }
    } catch (error) {
      pie.textContent = mensajeDeError(error);
      pie.className = 'px-2 py-1 border-t border-vm-border text-[10px] text-vm-accent';
      panel.classList.remove('hidden');
      boton.setAttribute('aria-expanded', 'true');
      encender.disabled = false;
    } finally {
      progreso.classList.add('hidden');
      progreso.classList.remove('flex');
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
