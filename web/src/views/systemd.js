import { renderIcons } from '../icons.js';
import { api } from '../api.js';
import { showResult } from '../components/ResultPanel.js';
import { ejecutar, escapar, filaTabla, inhabilitar, mensajeDeError, presentarEstadoUnidad } from '../ui.js';
import { esModo, esRutaAbsoluta, validarNombreUnidad } from '../validacion.js';

/**
 * Construye la vista de unidades systemd (numeral 3.3).
 *
 * @returns {HTMLElement} Vista lista para insertarse.
 */
export function renderSystemd() {
  const el = document.createElement('div');
  el.className = 'grid grid-cols-[58%_42%] gap-vm-gap items-start';
  el.innerHTML = `
    <div class="flex flex-col gap-vm-gap">
      <section class="card">
        <header class="px-3 py-1.5 border-b border-vm-border bg-[#f7f7f7]"><div class="flex items-center gap-1.5"><i data-lucide="upload" class="w-4 h-4"></i><span class="text-[12px] font-medium">Despliegue de unidad</span></div></header>
        <form id="form-despliegue" class="p-3 flex flex-col gap-2" novalidate>
          <div class="flex flex-col gap-1">
            <label class="label" for="unidad-nombre">Nombre de unidad (.service)</label>
            <input id="unidad-nombre" name="unidad" class="input font-mono" placeholder="catalogo.service" autocomplete="off" spellcheck="false">
            <span id="unidad-error" class="text-[11px] text-vm-accent hidden"></span>
          </div>
          <div class="flex flex-col gap-2">
            <div class="flex items-center justify-between">
              <span class="label">Archivos que se despliegan</span>
              <button type="button" id="agregar-archivo" class="btn h-vm-btn px-1.5"><i data-lucide="plus" class="w-4 h-4"></i> Añadir archivo</button>
            </div>
            <div id="archivos" class="flex flex-col gap-2"></div>
          </div>
          <button type="submit" class="btn btn-primary w-fit"><i data-lucide="upload"></i> Desplegar</button>
        </form>
      </section>
      <section class="card">
        <header class="px-3 py-1.5 border-b border-vm-border bg-[#f7f7f7] flex items-center justify-between">
          <div class="flex items-center gap-1.5"><i data-lucide="sliders-horizontal" class="w-4 h-4"></i><span class="text-[12px] font-medium">Control de servicios</span></div>
          <button id="unidades-refresh" class="btn h-vm-btn px-1.5" title="Consultar de nuevo"><i data-lucide="refresh-cw" class="w-4 h-4"></i></button>
        </header>
        <div class="p-3 flex flex-col gap-2">
          <div class="flex items-center gap-2">
            <select id="unidad-seleccion" class="input flex-1"></select>
            <span id="unidad-estado" class="px-1.5 py-0.5 border text-[11px] rounded-sm text-vm-fgSecondary border-vm-border bg-vm-bg">sin datos</span>
          </div>
          <div class="flex flex-wrap gap-1">
            <button class="btn" data-accion="iniciar"><i data-lucide="play"></i> Iniciar</button>
            <button class="btn" data-accion="detener"><i data-lucide="square"></i> Detener</button>
            <button class="btn" data-accion="reiniciar"><i data-lucide="rotate-ccw"></i> Reiniciar</button>
            <button class="btn" data-accion="estado"><i data-lucide="info"></i> Estado</button>
            <button class="btn" data-accion="habilitar"><i data-lucide="power"></i> Habilitar</button>
            <button class="btn" data-accion="deshabilitar"><i data-lucide="power-off"></i> Deshabilitar</button>
          </div>
        </div>
      </section>
      <section class="card">
        <header class="px-3 py-1.5 border-b border-vm-border bg-[#f7f7f7] flex items-center justify-between">
          <div class="flex items-center gap-1.5"><i data-lucide="activity" class="w-4 h-4"></i><span class="text-[12px] font-medium">Unidades administradas</span></div>
          <span id="unidades-cuenta" class="text-[11px] text-vm-fgSecondary"></span>
        </header>
        <div class="overflow-auto max-h-[220px]">
          <table class="w-full text-[12px]">
            <thead class="bg-vm-bg">
              <tr>
                <th class="text-left px-3 py-1 font-medium text-vm-fgSecondary">Unidad</th>
                <th class="text-left px-3 py-1 font-medium text-vm-fgSecondary">Estado</th>
                <th class="text-left px-3 py-1 font-medium text-vm-fgSecondary">Habilitada</th>
              </tr>
            </thead>
            <tbody id="unidades-filas"></tbody>
          </table>
        </div>
      </section>
    </div>
    <div id="result-slot"></div>
  `;

  renderIcons(el);

  /** Última lista de unidades recibida del servidor. */
  let unidades = [];

  const contenedorArchivos = el.querySelector('#archivos');
  const formulario = el.querySelector('#form-despliegue');
  const select = el.querySelector('#unidad-seleccion');
  const insignia = el.querySelector('#unidad-estado');
  const filas = el.querySelector('#unidades-filas');
  const cuenta = el.querySelector('#unidades-cuenta');

  /**
   * Añade una fila de archivo al formulario de despliegue.
   *
   * @param {{origen?: string, destino?: string, modo?: string}} [archivo] Valores iniciales.
   */
  const agregarArchivo = (archivo = {}) => {
    const fila = document.createElement('div');
    fila.className = 'grid grid-cols-[1fr_1fr_72px_28px] gap-1 items-center';
    fila.innerHTML = `
      <input class="input font-mono" data-campo="origen" placeholder="./app/catalogo" value="${escapar(archivo.origen)}" spellcheck="false">
      <input class="input font-mono" data-campo="destino" placeholder="/opt/catalogo/app" value="${escapar(archivo.destino)}" spellcheck="false">
      <input class="input font-mono" data-campo="modo" placeholder="755" value="${escapar(archivo.modo) || '755'}" maxlength="4" spellcheck="false">
      <button type="button" class="btn h-vm-btn px-1" data-quitar title="Quitar el archivo"><i data-lucide="x" class="w-4 h-4"></i></button>
    `;

    fila.querySelector('[data-quitar]').addEventListener('click', () => {
      if (contenedorArchivos.children.length > 1) {
        fila.remove();
      }
    });

    contenedorArchivos.appendChild(fila);
    renderIcons(fila);
  };

  /**
   * Cambia la insignia del estado de la unidad seleccionada.
   *
   * @param {object|null} estado Estado del contrato.
   */
  const marcarEstado = (estado) => {
    if (!estado) {
      insignia.textContent = 'sin datos';
      insignia.className = 'px-1.5 py-0.5 border text-[11px] rounded-sm text-vm-fgSecondary border-vm-border bg-vm-bg';
      return;
    }

    const presentacion = presentarEstadoUnidad(estado.estado);
    insignia.textContent = presentacion.texto;
    insignia.className = `px-1.5 py-0.5 border text-[11px] rounded-sm ${presentacion.clase}`;
    insignia.title = `activa: ${estado.activo} · habilitada: ${estado.habilitado}`;
  };

  const pintar = () => {
    cuenta.textContent = String(unidades.length);
    select.innerHTML = unidades.length
      ? unidades.map((unidad) => `<option value="${escapar(unidad.unidad)}">${escapar(unidad.unidad)}</option>`).join('')
      : '<option value="">Sin unidades</option>';

    if (unidades.length === 0) {
      filas.innerHTML = filaTabla(3, 'No hay unidades para mostrar');
      marcarEstado(null);
      return;
    }

    filas.innerHTML = unidades
      .map((unidad) => {
        const presentacion = presentarEstadoUnidad(unidad.estado);

        return `
        <tr class="border-t border-vm-border hover:bg-vm-bg">
          <td class="px-3 py-1.5 font-mono">${escapar(unidad.unidad)}</td>
          <td class="px-3 py-1.5"><span class="px-1.5 py-0.5 border text-[10px] rounded-sm ${presentacion.clase}">${presentacion.texto}</span></td>
          <td class="px-3 py-1.5">${unidad.habilitado ? '<i data-lucide="check" class="w-4 h-4 text-vm-success"></i>' : '<span class="text-vm-fgSecondary">—</span>'}</td>
        </tr>
      `;
      })
      .join('');

    renderIcons(filas);
    marcarEstado(unidades[0]);
  };

  const cargar = async () => {
    filas.innerHTML = filaTabla(3, 'Consultando las unidades…');

    try {
      const respuesta = await api.unidades.listar();
      unidades = respuesta.unidades ?? [];
      pintar();
      consultarEstado();
    } catch (error) {
      unidades = [];
      cuenta.textContent = '';
      filas.innerHTML = filaTabla(3, mensajeDeError(error), 'text-vm-accent');
      marcarEstado(null);
    }
  };

  /**
   * Consulta el estado de la unidad seleccionada.
   *
   * @returns {Promise<void>}
   */
  const consultarEstado = async () => {
    const unidad = select.value;

    if (!unidad) {
      return;
    }

    try {
      marcarEstado(await api.unidades.obtener(unidad));
    } catch {
      marcarEstado(null);
    }
  };

  el.querySelector('#agregar-archivo').addEventListener('click', () => agregarArchivo());
  el.querySelector('#unidades-refresh').addEventListener('click', cargar);
  select.addEventListener('change', consultarEstado);

  el.querySelectorAll('[data-accion]').forEach((boton) => {
    boton.addEventListener('click', async () => {
      const unidad = select.value;

      if (!unidad) {
        showResult({ ok: false, mensaje: 'Selecciona una unidad para operar.', operacion: 'systemctl' });
        return;
      }

      const accion = boton.dataset.accion;
      inhabilitar(el, true);

      const exito = await ejecutar({
        accion: async () => {
          switch (accion) {
            case 'iniciar':
              await api.unidades.cambiarEstado(unidad, { activo: true });
              return { mensaje: `Se activó la unidad «${unidad}».` };
            case 'detener':
              await api.unidades.cambiarEstado(unidad, { activo: false });
              return { mensaje: `Se desactivó la unidad «${unidad}».` };
            case 'habilitar':
              await api.unidades.cambiarEstado(unidad, { habilitado: true });
              return { mensaje: `Se habilitó el arranque de «${unidad}».` };
            case 'deshabilitar':
              await api.unidades.cambiarEstado(unidad, { habilitado: false });
              return { mensaje: `Se deshabilitó el arranque de «${unidad}».` };
            case 'reiniciar':
              await api.unidades.reiniciar(unidad);
              return { mensaje: `Se reinició la unidad «${unidad}».` };
            default:
              return api.unidades.obtener(unidad);
          }
        },
        operacion: `systemctl ${accion} ${unidad}`,
        informar: showResult,
        recargar: accion === 'estado' ? undefined : cargar,
      });

      inhabilitar(el, false);

      if (accion === 'estado' && exito) {
        marcarEstado(await api.unidades.obtener(unidad));
      }
    });
  });

  formulario.addEventListener('submit', async (evento) => {
    evento.preventDefault();

    const unidad = el.querySelector('#unidad-nombre').value.trim();
    const aviso = el.querySelector('#unidad-error');
    const invalido = validarNombreUnidad(unidad);

    if (invalido) {
      aviso.textContent = invalido;
      aviso.classList.remove('hidden');
      return;
    }

    aviso.classList.add('hidden');

    const archivos = [...contenedorArchivos.querySelectorAll('[data-campo="origen"]')].map((entrada) => {
      const fila = entrada.parentElement;

      return {
        origen: entrada.value.trim(),
        destino: fila.querySelector('[data-campo="destino"]').value.trim(),
        modo: fila.querySelector('[data-campo="modo"]').value.trim(),
      };
    });

    const problema = archivos.find(
      (archivo) => !archivo.origen || !esRutaAbsoluta(archivo.destino) || !esModo(archivo.modo),
    );

    if (problema) {
      showResult({
        ok: false,
        mensaje: 'Revisa los archivos: el origen es obligatorio, el destino debe ser una ruta absoluta y el modo octal de tres o cuatro dígitos.',
        operacion: `PUT /api/units/${unidad}`,
      });
      return;
    }

    inhabilitar(formulario, true);

    await ejecutar({
      accion: async () => {
        await api.unidades.desplegar(unidad, { archivos });

        return { mensaje: `Se desplegó la unidad «${unidad}» con ${archivos.length} archivo(s).` };
      },
      operacion: `PUT /api/units/${unidad}`,
      informar: showResult,
      recargar: cargar,
    });

    inhabilitar(formulario, false);
  });

  agregarArchivo();
  cargar();

  return el;
}
