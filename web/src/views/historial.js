import { renderIcons } from '../icons.js';
import { api } from '../api.js';
import { escapar, mensajeDeError } from '../ui.js';

/** Módulos que se pueden filtrar en el historial. */
const MODULOS = ['paquetes', 'llaves', 'issue', 'hostname', 'usuarios', 'systemd', 'haproxy'];

/**
 * Construye la vista del historial de operaciones.
 *
 * @returns {HTMLElement} Vista lista para insertarse.
 */
export function renderHistorial() {
  const el = document.createElement('div');
  el.className = 'flex flex-col gap-vm-gap';
  el.innerHTML = `
    <section class="card">
      <form id="filtros-historial" class="p-3 flex flex-wrap items-center gap-2">
        <div class="flex items-center gap-1 border border-vm-border px-2 h-vm-input flex-1 min-w-[200px]">
          <i data-lucide="search"></i>
          <input name="q" class="flex-1 outline-none text-[12px]" placeholder="Buscar: paquete, unidad, usuario, dirección…">
        </div>
        <input name="desde" class="input" type="date" title="Desde">
        <span class="text-vm-fgSecondary">→</span>
        <input name="hasta" class="input" type="date" title="Hasta">
        <input name="hora_desde" class="input w-[90px]" type="time" title="Hora desde">
        <span class="text-vm-fgSecondary">→</span>
        <input name="hora_hasta" class="input w-[90px]" type="time" title="Hora hasta">
        <input name="vm" class="input w-[140px] font-mono" placeholder="VM" title="Nombre de la VM">
        <select name="modulo" class="input">
          <option value="">Módulo: todos</option>
          ${MODULOS.map((modulo) => `<option value="${modulo}">${modulo}</option>`).join('')}
        </select>
        <button type="submit" class="btn btn-primary"><i data-lucide="search"></i> Aplicar</button>
        <button id="limpiar-historial" type="button" class="btn">Limpiar</button>
      </form>
    </section>
    <section class="card">
      <header class="px-3 py-1.5 border-b border-vm-border flex items-center justify-between bg-[#f7f7f7]">
        <div class="flex items-center gap-1.5"><i data-lucide="history" class="w-4 h-4"></i><span class="text-[12px] font-medium">Historial de operaciones</span></div>
        <button id="exportar-historial" type="button" class="btn" disabled><i data-lucide="file-down"></i> Exportar CSV</button>
      </header>
      <div class="overflow-auto">
        <table class="w-full text-[12px]">
          <thead class="bg-vm-bg">
            <tr>
              <th class="text-left px-3 py-1 font-medium text-vm-fgSecondary">FECHA Y HORA</th>
              <th class="text-left px-3 py-1 font-medium text-vm-fgSecondary">VM</th>
              <th class="text-left px-3 py-1 font-medium text-vm-fgSecondary">MÓDULO</th>
              <th class="text-left px-3 py-1 font-medium text-vm-fgSecondary">OPERACIÓN</th>
              <th class="text-left px-3 py-1 font-medium text-vm-fgSecondary">RESULTADO</th>
            </tr>
          </thead>
          <tbody id="historial-filas">
            <tr><td colspan="5" class="px-3 py-6 text-center text-vm-fgSecondary">Consultando…</td></tr>
          </tbody>
        </table>
      </div>
      <footer class="px-3 py-1.5 border-t border-vm-border text-[11px] text-vm-fgSecondary flex items-center justify-between">
        <span id="historial-estado">Consultando…</span>
        <span id="historial-total" class="font-mono">0</span>
      </footer>
    </section>
  `;

  renderIcons(el);

  const form = el.querySelector('#filtros-historial');
  const filas = el.querySelector('#historial-filas');
  const estado = el.querySelector('#historial-estado');
  const total = el.querySelector('#historial-total');
  const exportar = el.querySelector('#exportar-historial');
  let operaciones = [];

  /** Lee los filtros del formulario. */
  const filtros = () => {
    const datos = new FormData(form);
    const valores = {};

    for (const campo of ['q', 'desde', 'hasta', 'hora_desde', 'hora_hasta', 'vm', 'modulo']) {
      const valor = String(datos.get(campo) || '').trim();
      if (valor) {
        valores[campo] = valor;
      }
    }

    return valores;
  };

  /** Formatea la marca temporal en la zona local. */
  const fecha = (valor) => {
    const momento = new Date(valor);
    return Number.isNaN(momento.getTime()) ? valor : momento.toLocaleString('es-CO');
  };

  /** Pinta las filas recibidas. */
  const pintar = () => {
    if (operaciones.length === 0) {
      filas.innerHTML = `
        <tr><td colspan="5" class="px-3 py-6 text-center text-vm-fgSecondary">
          No hay operaciones que coincidan con los filtros.
        </td></tr>
      `;
      exportar.disabled = true;

      return;
    }

    filas.innerHTML = operaciones
      .map((operacion) => `
        <tr class="border-t border-vm-border hover:bg-[#faf9f9]">
          <td class="px-3 py-1 font-mono text-[11px] whitespace-nowrap">${escapar(fecha(operacion.fecha_hora))}</td>
          <td class="px-3 py-1 font-mono text-[11px]">${escapar(operacion.vm || '—')}</td>
          <td class="px-3 py-1">${escapar(operacion.modulo)}</td>
          <td class="px-3 py-1 font-mono text-[11px]">${escapar(operacion.operacion)}</td>
          <td class="px-3 py-1">
            <span class="px-1.5 py-0.5 border text-[10px] rounded-sm ${operacion.resultado === 'ok' ? 'border-vm-success text-vm-success' : 'border-vm-accent text-vm-accent'}">
              ${operacion.resultado === 'ok' ? 'OK' : 'Error'}
            </span>
          </td>
        </tr>
      `)
      .join('');

    exportar.disabled = false;
  };

  /** Consulta el historial con los filtros actuales. */
  const cargar = async () => {
    estado.textContent = 'Consultando…';
    estado.className = 'text-vm-fgSecondary';

    try {
      const respuesta = await api.historial.listar(filtros());
      operaciones = respuesta.operaciones ?? [];
      pintar();
      estado.textContent = operaciones.length === 0 ? 'Sin operaciones' : 'Historial actualizado';
      total.textContent = String(operaciones.length);
    } catch (error) {
      operaciones = [];
      filas.innerHTML = `
        <tr><td colspan="5" class="px-3 py-6 text-center text-vm-accent">${escapar(mensajeDeError(error))}</td></tr>
      `;
      estado.textContent = 'No fue posible consultar el historial';
      estado.className = 'text-vm-accent';
      total.textContent = '0';
      exportar.disabled = true;
    }
  };

  form.addEventListener('submit', (evento) => {
    evento.preventDefault();
    cargar();
  });

  el.querySelector('#limpiar-historial').addEventListener('click', () => {
    form.reset();
    cargar();
  });

  exportar.addEventListener('click', () => {
    const encabezado = 'fecha_hora,vm,modulo,operacion,resultado,detalle\n';
    const contenido = operaciones
      .map((operacion) => [
        operacion.fecha_hora,
        operacion.vm,
        operacion.modulo,
        operacion.operacion,
        operacion.resultado,
        operacion.detalle,
      ].map((valor) => `"${String(valor ?? '').replaceAll('"', '""')}"`).join(','))
      .join('\n');

    const enlace = document.createElement('a');
    enlace.href = URL.createObjectURL(new Blob([encabezado + contenido], { type: 'text/csv;charset=utf-8' }));
    enlace.download = 'historial-vmcons.csv';
    enlace.click();
    URL.revokeObjectURL(enlace.href);
  });

  cargar();

  return el;
}
