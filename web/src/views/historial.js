import { renderIcons } from '../icons.js';

export function renderHistorial() {
  const el = document.createElement('div');
  el.className = 'flex flex-col gap-vm-gap';
  el.innerHTML = `
    <section class="card">
      <div class="p-3 flex flex-wrap items-center gap-2">
        <div class="flex items-center gap-1 border border-vm-border px-2 h-vm-input flex-1 min-w-[200px]">
          <i data-lucide="search"></i>
          <input class="flex-1 outline-none text-[12px]" placeholder="Buscar: paquete, unidad, usuario, dirección...">
        </div>
        <input class="input" type="date">
        <span class="text-vm-fgSecondary">→</span>
        <input class="input" type="date">
        <input class="input w-[90px]" type="time">
        <span class="text-vm-fgSecondary">→</span>
        <input class="input w-[90px]" type="time">
        <select class="input"><option>VM: todas</option></select>
        <select class="input"><option>Módulo: todos</option></select>
        <button class="btn btn-primary"><i data-lucide="search"></i> Aplicar filtros</button>
        <button class="btn">Limpiar</button>
      </div>
    </section>
    <section class="card">
      <header class="px-3 py-1.5 border-b border-vm-border flex items-center justify-between bg-[#f7f7f7]">
        <div class="flex items-center gap-1.5"><i data-lucide="history" class="w-4 h-4"></i><span class="text-[12px] font-medium">Historial de operaciones</span></div>
        <button class="btn"><i data-lucide="file-down"></i> Exportar CSV</button>
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
          <tbody>
            <tr>
              <td colspan="5" class="px-3 py-6 text-center text-[12px] text-vm-fgSecondary">
                El historial se habilitará cuando esté conectada su persistencia
                (<span class="font-mono">docs/TODO.md</span> §9).
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <footer class="px-3 py-1.5 border-t border-vm-border text-[11px] text-vm-fgSecondary flex items-center justify-between">
        <span>Sin operaciones registradas</span>
        <span class="font-mono">0</span>
      </footer>
    </section>
  `;
  renderIcons(el);
  return el;
}
