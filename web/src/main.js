/**
 * Punto de entrada de la SPA: enrutado por hash y montaje del armazón.
 *
 * La barra superior se monta una sola vez: contiene el selector de
 * máquina, que consulta el inventario y la sesión. Reconstruirla en cada
 * cambio de pestaña multiplicaría esas consultas al hipervisor sin
 * aportar nada, así que solo se sustituyen las pestañas y el contenido.
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

/** Partes del armazón que se reemplazan al navegar. */
let armazon = null;

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
 * Construye el armazón la primera vez y devuelve sus partes móviles.
 *
 * @returns {{barra: HTMLElement, principal: HTMLElement}} Contenedores.
 */
function montarArmazon() {
  const app = document.getElementById('app');

  app.innerHTML = '';

  const barra = document.createElement('div');
  const principal = document.createElement('main');

  principal.className = 'max-w-vm mx-auto px-3 py-3';
  app.append(Header(), barra, principal);

  armazon = { barra, principal };

  return armazon;
}

/**
 * Monta las pestañas y la vista de la ruta activa.
 */
function montar() {
  const { barra, principal } = armazon ?? montarArmazon();
  const actual = rutaActual();

  barra.replaceChildren(Tabs(actual));

  const vista = rutas[actual]();
  vista.classList.add('fade-slide');
  principal.replaceChildren(vista);

  const ranura = document.getElementById('result-slot');

  if (ranura && !document.getElementById('resultado-global')) {
    const panel = ResultPanel();
    panel.id = 'resultado-global';
    ranura.appendChild(panel);
  }
}

window.addEventListener('hashchange', montar);

// Al vincular otra máquina o encenderla, el servidor invalida su caché y
// la vista activa se vuelve a montar para pedir los datos de la nueva
// máquina; la barra superior se conserva.
document.addEventListener('vmcons:maquina', montar);

montar();
