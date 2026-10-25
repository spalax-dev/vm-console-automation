import { renderIcons } from '../icons.js';

const tabs = [
  { id: 'inicio', label: 'Inicio', icon: 'home' },
  { id: 'paquetes', label: 'Paquetes', icon: 'package' },
  { id: 'llaves', label: 'Llaves SSH', icon: 'key-round' },
  { id: 'issue', label: '/etc/issue', icon: 'file-text' },
  { id: 'hostname', label: 'Hostname', icon: 'server' },
  { id: 'usuarios', label: 'Usuarios', icon: 'users' },
  { id: 'systemd', label: 'systemd', icon: 'activity' },
  { id: 'haproxy', label: 'HAProxy', icon: 'git-merge' },
  { id: 'historial', label: 'Historial', icon: 'history' },
];

export function Tabs(active = 'inicio') {
  const nav = document.createElement('nav');
  nav.className = 'bg-vm-surface border-b border-vm-border flex items-center gap-0.5 px-2';
  nav.innerHTML = tabs.map(t => `
    <button class="tab ${t.id === active ? 'active' : ''}" data-tab="${t.id}">
      <i data-lucide="${t.icon}"></i>
      <span>${t.label}</span>
    </button>
  `).join('');
  renderIcons(nav);
  nav.querySelectorAll('.tab').forEach(btn => {
    btn.addEventListener('click', () => {
      const id = btn.dataset.tab;
      window.location.hash = `#/${id}`;
    });
  });
  return nav;
}
