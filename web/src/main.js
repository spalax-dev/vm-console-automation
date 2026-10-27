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
import { ResultPanel, refreshResultPanel } from './components/ResultPanel.js';
import { api } from './api.js';
import { escapar } from './ui.js';
import { renderAcceso } from './views/acceso.js';
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

/** Usuario de la consola con la sesión iniciada. */
let usuarioActual = '';

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

  // El armazón ocupa la ventana: la barra superior y las pestañas quedan
  // fijas y solo el área de contenido se desplaza.
  app.className = 'flex h-dvh flex-col overflow-hidden';

  const barra = document.createElement('div');
  barra.className = 'shrink-0';

  const principal = document.createElement('main');
  principal.className = 'flex-1 min-h-0 overflow-y-auto';

  const contenido = document.createElement('div');
  contenido.className = 'max-w-vm mx-auto px-3 py-3';

  principal.appendChild(contenido);
  app.append(Header({ usuario: usuarioActual, alSalir: salir }), barra, principal);

  armazon = { barra, principal: contenido };

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

window.addEventListener('hashchange', () => {
  if (armazon) {
    montar();
  }
});

// Al vincular otra máquina o encenderla, el servidor invalida su caché y
// la vista activa se vuelve a montar para pedir los datos de la nueva
// máquina; la barra superior se conserva.
document.addEventListener('vmcons:vm-activa', () => {
  refreshResultPanel();
});

document.addEventListener('vmcons:maquina', () => {
  if (armazon) {
    montar();
  }
});

/**
 * Muestra la pantalla de acceso o un aviso cuando la consulta falla.
 *
 * @param {object|null} estado Estado de la autenticación.
 * @param {string} [error] Mensaje de fallo al consultar el estado.
 */
function mostrarAcceso(estado, error) {
  const app = document.getElementById('app');

  armazon = null;
  app.className = '';

  if (error) {
    app.innerHTML = `
      <div class="min-h-dvh flex items-center justify-center bg-vm-bg px-4">
        <div class="card p-3 max-w-[420px] flex flex-col gap-2">
          <p class="text-[13px] font-medium">No fue posible comprobar la sesión de la consola</p>
          <p class="text-[11px] text-vm-accent">${escapar(error)}</p>
          <button id="acceso-reintentar" type="button" class="btn btn-primary justify-center">
            Reintentar
          </button>
        </div>
      </div>
    `;
    app.querySelector('#acceso-reintentar').addEventListener('click', iniciar);

    return;
  }

  app.appendChild(
    renderAcceso({
      configurada: Boolean(estado && estado.configurada),
      alEntrar: (sesion) => {
        usuarioActual = (sesion && sesion.usuario) || usuarioActual;
        iniciar();
      },
    }),
  );
}

/**
 * Cierra la sesión de la consola y vuelve a la pantalla de acceso.
 */
async function salir() {
  try {
    await api.auth.salir();
  } catch {
    // Si la sesión ya venció, el servidor responde 401 igualmente.
  }

  usuarioActual = '';
  iniciar();
}

/**
 * Consulta el estado de la autenticación y monta la pantalla que
 * corresponde.
 */
async function iniciar() {
  try {
    const estado = await api.auth.estado();

    if (!estado.autenticado) {
      mostrarAcceso(estado);

      return;
    }

    usuarioActual = estado.usuario || '';
    montar();
  } catch (error) {
    mostrarAcceso(null, error.message);
  }
}

iniciar();
