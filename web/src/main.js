/**
 * Punto de entrada de la SPA: enrutado por hash y montaje del armazón.
 *
 * Cada módulo es una vista independiente que consulta su recurso del
 * contrato OpenAPI al montarse. La navegación no recarga la página.
 */

import './styles.css';
import '@fontsource/ibm-plex-sans/latin-400.css';
import '@fontsource/ibm-plex-sans/latin-500.css';
import '@fontsource/ibm-plex-sans/latin-600.css';
import '@fontsource/ibm-plex-sans/latin-700.css';
import '@fontsource/jetbrains-mono/latin-400.css';
import '@fontsource/jetbrains-mono/latin-500.css';
import { Header } from './components/Header.js';
import { Tabs } from './components/Tabs.js';
import { ResultPanel } from './components/ResultPanel.js';
import { renderInicio } from './views/inicio.js';
import { renderPaquetes } from './views/paquetes.js';
import { renderLlaves } from './views/llaves.js';
import { renderIssue } from './views/issue.js';
import { renderHostname } from './views/hostname.js';
import { renderUsuarios } from './views/usuarios.js';
import { renderSystemd } from './views/systemd.js';
import { renderHaproxy } from './views/haproxy.js';
import { renderHistorial } from './views/historial.js';

/** Vistas disponibles, indexadas por el fragmento de la URL. */
const rutas = {
  inicio: renderInicio,
  paquetes: renderPaquetes,
  llaves: renderLlaves,
  issue: renderIssue,
  hostname: renderHostname,
  usuarios: renderUsuarios,
  systemd: renderSystemd,
  haproxy: renderHaproxy,
  historial: renderHistorial,
};

/** Ruta que se muestra cuando el fragmento no corresponde a una vista. */
const RUTA_INICIAL = 'inicio';

/**
 * Devuelve la ruta activa a partir del fragmento de la URL.
 *
 * @returns {string} Clave de la vista.
 */
function rutaActual() {
  const fragmento = window.location.hash.replace(/^#\/?/, '');

  return Object.hasOwn(rutas, fragmento) ? fragmento : RUTA_INICIAL;
}

/**
 * Monta el armazón y la vista de la ruta activa.
 */
function montar() {
  const app = document.getElementById('app');
  const actual = rutaActual();

  app.innerHTML = '';
  app.appendChild(Header());
  app.appendChild(Tabs(actual));

  const principal = document.createElement('main');
  principal.className = 'max-w-vm mx-auto px-3 py-3';

  const vista = rutas[actual]();
  vista.classList.add('fade-slide');
  principal.appendChild(vista);
  app.appendChild(principal);

  const ranura = document.getElementById('result-slot');

  if (ranura && !document.getElementById('resultado-global')) {
    const panel = ResultPanel();
    panel.id = 'resultado-global';
    ranura.appendChild(panel);
  }
}

window.addEventListener('hashchange', montar);
montar();
