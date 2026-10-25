import { renderIcons } from '../icons.js';
import { api } from '../api.js';
import { escapar, mensajeDeError } from '../ui.js';

/**
 * Abre el modal de registro de una máquina con Guest Additions.
 *
 * El usuario ingresa sus credenciales y la consola prepara SSH: instala
 * `openssh-server`, la llave de root y deja la máquina registrada.
 *
 * @param {object} opciones Opciones del modal.
 * @param {{nombre: string}} opciones.maquina Máquina por registrar.
 * @param {() => void} [opciones.alRegistrar] Aviso tras registrar.
 * @returns {void}
 */
export function abrirRegistro({ maquina, alRegistrar }) {
  const overlay = document.createElement('div');
  overlay.className = 'fixed inset-0 z-50 bg-black/30 flex items-center justify-center p-4';
  overlay.innerHTML = `
    <section class="card w-[400px] max-w-full" role="dialog" aria-modal="true" aria-labelledby="registro-titulo">
      <header class="px-3 py-1.5 border-b border-vm-border bg-[#f7f7f7] flex items-center justify-between">
        <span id="registro-titulo" class="text-[12px] font-medium">Registrar ${escapar(maquina.nombre)}</span>
        <button id="registro-cerrar" type="button" class="btn h-vm-btn px-1" title="Cerrar" aria-label="Cerrar">
          <i data-lucide="x" class="w-4 h-4"></i>
        </button>
      </header>
      <form id="registro-form" class="p-3 flex flex-col gap-2">
        <p class="text-[11px] text-vm-fgSecondary">
          La máquina tiene Guest Additions pero aún no está registrada. Ingresa un usuario con permisos
          para preparar SSH; las credenciales se guardan solo en este equipo.
        </p>
        <div id="registro-diagnostico" class="border border-vm-border p-2 flex flex-col gap-1 text-[11px]">
          <div class="flex items-center justify-between">
            <span class="text-vm-fgSecondary">Guest Additions</span>
            <span id="diag-ga" class="font-mono">consultando…</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-vm-fgSecondary">SSH (puerto 22)</span>
            <span id="diag-ssh" class="font-mono">consultando…</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-vm-fgSecondary">Usuario sugerido</span>
            <span id="diag-usuario" class="font-mono">—</span>
          </div>
          <ul id="diag-problemas" class="list-disc pl-4 text-vm-accent"></ul>
          <ul id="diag-sugerencias" class="list-disc pl-4 text-vm-fgSecondary"></ul>
        </div>
        <label class="flex flex-col gap-1">
          <span class="label">Usuario</span>
          <input name="usuario" value="root" class="input font-mono" autocomplete="username" required>
        </label>
        <label class="flex flex-col gap-1">
          <span class="label">Contraseña</span>
          <input name="password" type="password" class="input" autocomplete="current-password" required>
        </label>
        <div id="registro-progreso" class="hidden flex-col gap-1.5 pt-1" aria-live="polite">
          <div class="flex items-center justify-between text-[11px]">
            <span class="flex items-center gap-1.5">
              <i data-lucide="loader-2" class="w-3.5 h-3.5 text-vm-accent animate-spin"></i>
              <span id="registro-paso">Guardando credenciales…</span>
            </span>
            <span id="registro-porcentaje" class="font-mono text-vm-fgSecondary">0 %</span>
          </div>
          <div class="h-1.5 w-full bg-vm-bg border border-vm-border overflow-hidden">
            <div id="registro-barra" class="h-full bg-vm-accent transition-all duration-300" style="width: 0%"></div>
          </div>
          <div id="registro-ayuda" class="text-[10px] text-vm-fgSecondary">
            La consola está conectando con la máquina y preparando SSH. Esto puede tardar unos segundos.
          </div>
        </div>
        <div id="registro-error" class="text-[11px] text-vm-accent min-h-[14px]"></div>
        <footer class="flex justify-end gap-2 pt-1">
          <button id="registro-cancelar" type="button" class="btn">Cancelar</button>
          <button type="submit" class="btn btn-primary"><i data-lucide="key-round"></i> Registrar</button>
        </footer>
      </form>
    </section>
  `;

  renderIcons(overlay);

  const cerrarForzado = () => overlay.remove();

  const form = overlay.querySelector('#registro-form');
  const errorEl = overlay.querySelector('#registro-error');
  const boton = form.querySelector('button[type=submit]');
  const progreso = overlay.querySelector('#registro-progreso');
  const barra = overlay.querySelector('#registro-barra');
  const porcentajeEl = overlay.querySelector('#registro-porcentaje');
  const pasoEl = overlay.querySelector('#registro-paso');
  const ayudaEl = overlay.querySelector('#registro-ayuda');
  const cerrarBoton = overlay.querySelector('#registro-cerrar');
  const cancelarBoton = overlay.querySelector('#registro-cancelar');
  const diagGA = overlay.querySelector('#diag-ga');
  const diagSSH = overlay.querySelector('#diag-ssh');
  const diagUsuario = overlay.querySelector('#diag-usuario');
  const diagProblemas = overlay.querySelector('#diag-problemas');
  const diagSugerencias = overlay.querySelector('#diag-sugerencias');
  const campoUsuario = form.querySelector('input[name=usuario]');

  /** Indica que hay una petición en curso y el modal no debe cerrarse. */
  let ocupado = false;

  /** Temporizador que anima la barra mientras el servidor trabaja. */
  let temporizador = 0;

  const cerrar = () => {
    if (!ocupado) {
      cerrarForzado();
    }
  };

  /**
   * Consulta el diagnóstico de la máquina y lo muestra en el modal.
   *
   * @returns {Promise<void>}
   */
  const cargarDiagnostico = async () => {
    try {
      const informe = await api.sesion.diagnostico();

      diagGA.textContent = informe.guest_additions_version || 'no informa';
      diagSSH.textContent = informe.ssh_abierto ? 'abierto' : 'cerrado';
      diagSSH.className = `font-mono ${informe.ssh_abierto ? 'text-vm-success' : 'text-vm-accent'}`;
      diagUsuario.textContent = informe.usuario_sugerido || '—';

      diagProblemas.innerHTML = (informe.problemas ?? [])
        .map((problema) => `<li>${escapar(problema)}</li>`)
        .join('');
      diagSugerencias.innerHTML = (informe.sugerencias ?? [])
        .map((sugerencia) => `<li>${escapar(sugerencia)}</li>`)
        .join('');

      if (informe.usuario_sugerido && campoUsuario.value === 'root') {
        campoUsuario.value = informe.usuario_sugerido;
      }
    } catch {
      diagGA.textContent = 'no disponible';
      diagSSH.textContent = 'no disponible';
      diagSSH.className = 'font-mono text-vm-fgSecondary';
    }
  };

  /**
   * Bloquea o libera el formulario durante la petición.
   *
   * @param {boolean} bloqueado Estado deseado.
   */
  const bloquear = (bloqueado) => {
    ocupado = bloqueado;

    for (const control of form.querySelectorAll('input, button')) {
      control.disabled = bloqueado;
    }

    cerrarBoton.disabled = bloqueado;
    cancelarBoton.disabled = bloqueado;
    boton.innerHTML = bloqueado ? 'Registrando…' : '<i data-lucide="key-round"></i> Registrar';

    if (!bloqueado) {
      renderIcons(boton);
    }
  };

  /** Muestra el progreso y arranca la animación. */
  const iniciarProgreso = () => {
    progreso.classList.remove('hidden');
    progreso.classList.add('flex');

    let porcentaje = 6;

    const pintar = () => {
      barra.style.width = `${porcentaje}%`;
      porcentajeEl.textContent = `${porcentaje} %`;

      if (porcentaje >= 82) {
        pasoEl.textContent = 'Verificando el acceso SSH…';
      } else if (porcentaje >= 54) {
        pasoEl.textContent = 'Preparando SSH y la llave de root…';
      } else if (porcentaje >= 26) {
        pasoEl.textContent = 'Conectando con Guest Additions…';
      }
    };

    pintar();

    temporizador = setInterval(() => {
      porcentaje = Math.min(porcentaje + 5, 88);
      pintar();
    }, 600);
  };

  /** Detiene la animación y fija el resultado final. */
  const terminarProgreso = (exito, texto) => {
    clearInterval(temporizador);
    barra.style.width = '100%';
    porcentajeEl.textContent = '100 %';
    barra.className = `h-full transition-all duration-300 ${exito ? 'bg-vm-success' : 'bg-vm-accent'}`;
    pasoEl.textContent = texto;
    ayudaEl.textContent = exito ? 'La máquina quedó registrada.' : 'Revisa el detalle técnico e inténtalo de nuevo.';
  };

  cerrarBoton.addEventListener('click', cerrar);
  cancelarBoton.addEventListener('click', cerrar);
  overlay.addEventListener('click', (evento) => {
    if (evento.target === overlay) {
      cerrar();
    }
  });
  overlay.addEventListener('keydown', (evento) => {
    if (evento.key === 'Escape') {
      cerrar();
    }
  });

  form.addEventListener('submit', async (evento) => {
    evento.preventDefault();
    errorEl.textContent = '';

    // Los datos se leen antes de deshabilitar los campos: un input
    // deshabilitado no forma parte de FormData.
    const datos = new FormData(form);

    bloquear(true);
    iniciarProgreso();

    try {
      await api.sesion.registrar(String(datos.get('usuario') || ''), String(datos.get('password') || ''));
      terminarProgreso(true, 'Máquina registrada.');

      setTimeout(() => {
        cerrarForzado();

        if (alRegistrar) {
          alRegistrar();
        }
      }, 650);
    } catch (fallo) {
      terminarProgreso(false, 'No fue posible registrar la máquina.');
      errorEl.textContent = mensajeDeError(fallo);
      await cargarDiagnostico();

      if (fallo.tecnico) {
        const detalle = document.createElement('pre');
        detalle.className = 'mt-1 p-1.5 bg-vm-bg border border-vm-border text-[10px] font-mono whitespace-pre-wrap max-h-[120px] overflow-auto';
        detalle.textContent = fallo.tecnico;
        errorEl.appendChild(detalle);
      }

      bloquear(false);
    }
  });

  document.body.appendChild(overlay);
  form.querySelector('input[name=usuario]').focus();
  cargarDiagnostico();
}
