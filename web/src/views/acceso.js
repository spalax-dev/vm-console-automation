/**
 * Pantalla de acceso a la consola.
 *
 * Se muestra antes que el armazón: si todavía no existe una cuenta, ofrece
 * crearla; si existe, pide usuario y contraseña. La cookie de sesión la
 * instala el servidor, de modo que aquí solo se envía el formulario.
 */

import { renderIcons } from '../icons.js';
import { api } from '../api.js';
import { mensajeDeError } from '../ui.js';
import { validarPasswordConsola, validarUsuarioConsola } from '../validacion.js';

/**
 * Construye la pantalla de acceso.
 *
 * @param {object} [opciones] Opciones de la pantalla.
 * @param {boolean} [opciones.configurada] Ya existe una cuenta de la consola.
 * @param {(estado: object) => void} [opciones.alEntrar] Aviso tras entrar.
 * @returns {HTMLElement} Pantalla lista para insertarse.
 */
export function renderAcceso({ configurada = false, alEntrar } = {}) {
  const el = document.createElement('div');
  el.className = 'min-h-dvh flex items-center justify-center bg-vm-bg px-4';

  const titulo = configurada ? 'Iniciar sesión' : 'Crear la cuenta de la consola';
  const ayuda = configurada
    ? 'La consola pide usuario y contraseña para administrar la máquina virtual.'
    : 'Es la primera vez que se abre la consola: define la cuenta con la que se administrará.';

  el.innerHTML = `
    <div class="w-full max-w-[380px] card p-0">
      <header class="px-3 py-2 border-b border-vm-border bg-[#f7f7f7] flex items-center gap-2">
        <span class="w-2.5 h-2.5 rounded-full bg-vm-accent"></span>
        <span class="font-semibold text-[15px] tracking-tight">vmcons</span>
      </header>
      <form id="acceso-form" class="p-3 flex flex-col gap-2.5" novalidate>
        <div class="flex flex-col gap-0.5">
          <h1 class="text-[14px] font-medium">${titulo}</h1>
          <p class="text-[11px] text-vm-fgSecondary">${ayuda}</p>
        </div>
        <div class="flex flex-col gap-1">
          <label class="label" for="acceso-usuario">Usuario</label>
          <input id="acceso-usuario" name="usuario" class="input font-mono" autocomplete="username"
            spellcheck="false" maxlength="32">
        </div>
        <div class="flex flex-col gap-1">
          <label class="label" for="acceso-password">Contraseña</label>
          <input id="acceso-password" name="password" type="password" class="input"
            autocomplete="${configurada ? 'current-password' : 'new-password'}" maxlength="128">
        </div>
        ${
          configurada
            ? ''
            : `<div class="flex flex-col gap-1">
          <label class="label" for="acceso-repetir">Repetir la contraseña</label>
          <input id="acceso-repetir" name="repetir" type="password" class="input"
            autocomplete="new-password" maxlength="128">
        </div>`
        }
        <p id="acceso-aviso" class="text-[11px] text-vm-accent min-h-[14px]" role="alert"></p>
        <button id="acceso-enviar" type="submit" class="btn btn-primary justify-center">
          ${configurada ? 'Entrar' : 'Crear cuenta y entrar'}
        </button>
      </form>
    </div>
  `;

  renderIcons(el);

  const formulario = el.querySelector('#acceso-form');
  const aviso = el.querySelector('#acceso-aviso');
  const enviar = el.querySelector('#acceso-enviar');

  /**
   * Muestra un aviso en la pantalla.
   *
   * @param {string} texto Mensaje para el usuario.
   */
  const mostrar = (texto) => {
    aviso.textContent = texto;
  };

  formulario.addEventListener('submit', async (evento) => {
    evento.preventDefault();

    const datos = new FormData(formulario);
    const usuario = String(datos.get('usuario') ?? '').trim();
    const password = String(datos.get('password') ?? '');
    const repetir = String(datos.get('repetir') ?? '');

    const errorUsuario = validarUsuarioConsola(usuario);
    if (errorUsuario) {
      mostrar(errorUsuario);

      return;
    }

    const errorPassword = validarPasswordConsola(password);
    if (errorPassword) {
      mostrar(errorPassword);

      return;
    }

    if (!configurada && password !== repetir) {
      mostrar('Las contraseñas no coinciden.');

      return;
    }

    mostrar('');
    enviar.disabled = true;

    try {
      const estado = configurada
        ? await api.auth.entrar(usuario, password)
        : await api.auth.configurar(usuario, password);

      if (alEntrar) {
        alEntrar(estado);
      }
    } catch (error) {
      mostrar(mensajeDeError(error));
    } finally {
      enviar.disabled = false;
    }
  });

  return el;
}
