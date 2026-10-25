import { renderIcons } from '../icons.js';
import { api } from '../api.js';
import { showResult } from '../components/ResultPanel.js';
import { ejecutar, escapar, filaTabla, inhabilitar, mensajeDeError } from '../ui.js';
import { validarNombrePaquete } from '../validacion.js';

/**
 * Construye la vista del módulo de paquetes APT (numeral 3.1.1).
 *
 * @returns {HTMLElement} Vista lista para insertarse.
 */
export function renderPaquetes() {
  const el = document.createElement('div');
  el.className = 'grid grid-cols-[58%_42%] gap-vm-gap';
  el.innerHTML = `
    <div class="flex flex-col gap-vm-gap">
      <section class="card">
        <header class="px-3 py-1.5 border-b border-vm-border bg-[#f7f7f7]"><div class="flex items-center gap-1.5"><i data-lucide="package" class="w-4 h-4"></i><span class="text-[12px] font-medium">Instalación y eliminación de paquetes</span></div></header>
        <form id="form-paquetes" class="p-3 flex flex-col gap-2" novalidate>
          <div class="flex gap-4 text-[12px]">
            <label class="flex items-center gap-1.5"><input type="radio" name="accion" value="instalar" checked> Instalar</label>
            <label class="flex items-center gap-1.5"><input type="radio" name="accion" value="eliminar"> Eliminar</label>
          </div>
          <div class="flex flex-col gap-1">
            <label class="label" for="paquete-nombre">Nombre del paquete</label>
            <input id="paquete-nombre" name="nombre" class="input font-mono" placeholder="nano" autocomplete="off" spellcheck="false" required>
            <span id="paquete-error" class="text-[11px] text-vm-accent hidden"></span>
          </div>
          <div class="flex gap-2">
            <button type="submit" class="btn btn-primary"><i data-lucide="play"></i> Ejecutar</button>
            <button type="reset" class="btn">Limpiar</button>
          </div>
        </form>
      </section>
      <section class="card">
        <header class="px-3 py-1.5 border-b border-vm-border bg-[#f7f7f7] flex items-center justify-between">
          <div class="flex items-center gap-1.5"><i data-lucide="list" class="w-4 h-4"></i><span class="text-[12px] font-medium">Paquetes instalados</span></div>
          <div class="flex items-center gap-1.5">
            <span id="paquetes-cuenta" class="text-[11px] text-vm-fgSecondary"></span>
            <button id="paquetes-refresh" class="btn h-vm-btn px-1.5" title="Consultar de nuevo"><i data-lucide="refresh-cw" class="w-4 h-4"></i></button>
          </div>
        </header>
        <div class="p-2 border-b border-vm-border">
          <div class="flex items-center gap-1 border border-vm-border px-2 h-vm-input">
            <i data-lucide="search"></i>
            <input id="paquetes-filtro" class="flex-1 outline-none text-[12px] bg-transparent" placeholder="Filtrar por nombre…">
          </div>
        </div>
        <div class="overflow-auto max-h-[320px]">
          <table class="w-full text-[12px]">
            <thead class="bg-vm-bg sticky top-0">
              <tr>
                <th class="text-left px-3 py-1 font-medium text-vm-fgSecondary">Nombre</th>
                <th class="text-left px-3 py-1 font-medium text-vm-fgSecondary">Versión</th>
                <th class="text-left px-3 py-1 font-medium text-vm-fgSecondary">Estado</th>
              </tr>
            </thead>
            <tbody id="paquetes-filas"></tbody>
          </table>
        </div>
      </section>
    </div>
    <div id="result-slot"></div>
  `;

  renderIcons(el);

  /** Última lista recibida del servidor. */
  let paquetes = [];

  const filas = el.querySelector('#paquetes-filas');
  const cuenta = el.querySelector('#paquetes-cuenta');
  const filtro = el.querySelector('#paquetes-filtro');

  const pintar = () => {
    const texto = filtro.value.trim().toLowerCase();
    const visibles = texto ? paquetes.filter((p) => p.nombre.toLowerCase().includes(texto)) : paquetes;

    cuenta.textContent = `${visibles.length} de ${paquetes.length}`;

    if (visibles.length === 0) {
      filas.innerHTML = filaTabla(3, paquetes.length === 0 ? 'No hay paquetes para mostrar' : 'Ningún paquete coincide con el filtro');
      return;
    }

    filas.innerHTML = visibles
      .map(
        (paquete) => `
        <tr class="border-t border-vm-border hover:bg-vm-bg">
          <td class="px-3 py-1.5 font-mono">${escapar(paquete.nombre)}</td>
          <td class="px-3 py-1.5 font-mono text-vm-fgSecondary">${escapar(paquete.version) || '—'}</td>
          <td class="px-3 py-1.5">
            <span class="px-1.5 py-0.5 border text-[10px] rounded-sm ${paquete.estado === 'installed' ? 'text-vm-success border-vm-success bg-[#e8f5e9]' : 'text-vm-fgSecondary border-vm-border bg-vm-bg'}">${escapar(paquete.estado)}</span>
          </td>
        </tr>
      `,
      )
      .join('');
  };

  const cargar = async () => {
    filas.innerHTML = filaTabla(3, 'Consultando los paquetes…');

    try {
      const respuesta = await api.paquetes.listar();
      paquetes = respuesta.paquetes ?? [];
      pintar();
    } catch (error) {
      paquetes = [];
      cuenta.textContent = '';
      filas.innerHTML = filaTabla(3, mensajeDeError(error), 'text-vm-accent');
    }
  };

  filtro.addEventListener('input', pintar);
  el.querySelector('#paquetes-refresh').addEventListener('click', cargar);

  const formulario = el.querySelector('#form-paquetes');
  const campo = el.querySelector('#paquete-nombre');
  const aviso = el.querySelector('#paquete-error');

  formulario.addEventListener('submit', async (evento) => {
    evento.preventDefault();

    const accion = formulario.elements.accion.value;
    const nombre = campo.value.trim();
    const invalido = validarNombrePaquete(nombre);

    if (invalido) {
      aviso.textContent = invalido;
      aviso.classList.remove('hidden');
      campo.focus();
      return;
    }

    aviso.classList.add('hidden');
    inhabilitar(formulario, true);

    await ejecutar({
      accion: () => (accion === 'instalar' ? api.paquetes.instalar(nombre) : api.paquetes.eliminar(nombre)),
      operacion: accion === 'instalar' ? `apt-get install -y ${nombre}` : `apt-get remove -y ${nombre}`,
      informar: (resultado) => {
        if (accion === 'eliminar' && resultado.ok) {
          resultado.mensaje = resultado.mensaje || `Se eliminó el paquete «${nombre}».`;
        }
        showResult(resultado);
      },
      recargar: cargar,
    });

    inhabilitar(formulario, false);
  });

  formulario.addEventListener('reset', () => {
    aviso.classList.add('hidden');
    setTimeout(() => campo.focus(), 0);
  });

  cargar();

  return el;
}
