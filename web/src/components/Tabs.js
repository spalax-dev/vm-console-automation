import { renderIcons } from '../icons.js';

/** Módulos de la consola, en el orden de la barra de pestañas. */
export const PESTANAS = [
  { id: 'inicio', etiqueta: 'Inicio', icono: 'home' },
  { id: 'paquetes', etiqueta: 'Paquetes', icono: 'package' },
  { id: 'llaves', etiqueta: 'Llaves SSH', icono: 'key-round' },
  { id: 'issue', etiqueta: '/etc/issue', icono: 'file-text' },
  { id: 'hostname', etiqueta: 'Hostname', icono: 'server' },
  { id: 'usuarios', etiqueta: 'Usuarios', icono: 'users' },
  { id: 'systemd', etiqueta: 'systemd', icono: 'activity' },
  { id: 'haproxy', etiqueta: 'HAProxy', icono: 'git-merge' },
  { id: 'historial', etiqueta: 'Historial', icono: 'history' },
];

/**
 * Construye la barra de pestañas de módulos.
 *
 * Se comporta como un conjunto de pestañas accesible: la pestaña activa es
 * la única alcanzable con Tab y las flechas, Inicio y Fin mueven la
 * selección, igual que en una aplicación de escritorio.
 *
 * @param {string} [activa] Identificador de la pestaña activa.
 * @returns {HTMLElement} Barra de pestañas lista para insertarse.
 */
export function Tabs(activa = 'inicio') {
  const nav = document.createElement('nav');
  nav.className = 'bg-vm-surface border-b border-vm-border flex items-center gap-0.5 px-2';
  nav.setAttribute('role', 'tablist');
  nav.setAttribute('aria-label', 'Módulos de la consola');
  nav.innerHTML = PESTANAS.map(
    (pestana) => `
    <button type="button" role="tab" class="tab ${pestana.id === activa ? 'active' : ''}" data-tab="${pestana.id}"
      aria-selected="${pestana.id === activa}" tabindex="${pestana.id === activa ? 0 : -1}">
      <i data-lucide="${pestana.icono}"></i>
      <span>${pestana.etiqueta}</span>
    </button>
  `,
  ).join('');

  renderIcons(nav);

  const botones = [...nav.querySelectorAll('.tab')];

  /**
   * Navega a una pestaña por su identificador.
   *
   * @param {string} id Identificador de la pestaña.
   */
  const ir = (id) => {
    window.location.hash = `#/${id}`;
  };

  /**
   * Activa una pestaña desde el teclado: mueve el foco y cambia la vista.
   *
   * @param {number} indice Posición de la pestaña en la barra.
   */
  const activar = (indice) => {
    const destino = botones[(indice + botones.length) % botones.length];

    botones.forEach((boton) => {
      boton.tabIndex = boton === destino ? 0 : -1;
    });

    destino.focus();
    ir(destino.dataset.tab);
  };

  botones.forEach((boton, indice) => {
    boton.addEventListener('click', () => ir(boton.dataset.tab));
    boton.addEventListener('keydown', (evento) => {
      switch (evento.key) {
        case 'ArrowRight':
          evento.preventDefault();
          activar(indice + 1);
          break;
        case 'ArrowLeft':
          evento.preventDefault();
          activar(indice - 1);
          break;
        case 'Home':
          evento.preventDefault();
          activar(0);
          break;
        case 'End':
          evento.preventDefault();
          activar(botones.length - 1);
          break;
        default:
          break;
      }
    });
  });

  return nav;
}
