/**
 * Utilidades de presentación compartidas por las vistas.
 *
 * Reúne el escapado de texto, la traducción de estados del contrato y la
 * construcción del objeto que consume el panel de resultado.
 */

import { renderIcons } from './icons.js';

/**
 * Escapa un valor para insertarlo como texto en una plantilla HTML.
 *
 * @param {unknown} valor Valor que se muestra.
 * @returns {string} Texto seguro para HTML.
 */
export function escapar(valor) {
  if (valor === null || valor === undefined) {
    return '';
  }

  return String(valor)
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');
}

/**
 * Traduce el estado de encendido de la máquina a una etiqueta y una clase.
 *
 * @param {string} estado Estado del contrato: `running`, `poweroff`, `paused`, `saved` o `unknown`.
 * @returns {{texto: string, clase: string, punto: string}} Presentación del estado.
 */
export function presentarEstado(estado) {
  switch (estado) {
    case 'running':
      return { texto: 'Encendida', clase: 'text-vm-success border-vm-success bg-[#e8f5e9]', punto: 'bg-vm-success' };
    case 'paused':
      return { texto: 'Pausada', clase: 'text-[#b9770e] border-vm-warning bg-[#fdf3e3]', punto: 'bg-vm-warning' };
    case 'poweroff':
    case 'saved':
      return { texto: 'Apagada', clase: 'text-vm-muted border-vm-muted bg-[#f2f4f4]', punto: 'bg-vm-muted' };
    default:
      return { texto: 'Desconocido', clase: 'text-vm-fgSecondary border-vm-border bg-vm-bg', punto: 'bg-vm-muted' };
  }
}

/**
 * Traduce el estado de una unidad systemd a una etiqueta y una clase.
 *
 * @param {string} estado Estado del contrato: `active`, `inactive`, `failed` o `unknown`.
 * @returns {{texto: string, clase: string, punto: string}} Presentación del estado.
 */
export function presentarEstadoUnidad(estado) {
  switch (estado) {
    case 'active':
      return { texto: 'activa', clase: 'text-vm-success border-vm-success bg-[#e8f5e9]', punto: 'bg-vm-success' };
    case 'failed':
      return { texto: 'falló', clase: 'text-vm-accent border-vm-accent bg-[#fdf1ef]', punto: 'bg-vm-accent' };
    case 'inactive':
      return { texto: 'inactiva', clase: 'text-vm-fgSecondary border-vm-border bg-vm-bg', punto: 'bg-vm-muted' };
    default:
      return { texto: 'desconocida', clase: 'text-vm-fgSecondary border-vm-border bg-vm-bg', punto: 'bg-vm-muted' };
  }
}

/**
 * Traduce el estado de un servidor del back-end a una etiqueta.
 *
 * @param {string} estado Estado informado por HAProxy: `up`, `down` o `unknown`.
 * @returns {{texto: string, clase: string, punto: string}} Presentación del estado.
 */
export function presentarEstadoServidor(estado) {
  switch (estado) {
    case 'up':
      return { texto: 'UP', clase: 'text-vm-success border-vm-success bg-[#e8f5e9]', punto: 'bg-vm-success' };
    case 'down':
      return { texto: 'DOWN', clase: 'text-vm-accent border-vm-accent bg-[#fdf1ef]', punto: 'bg-vm-accent' };
    default:
      return { texto: 'DESCONOCIDO', clase: 'text-vm-fgSecondary border-vm-border bg-vm-bg', punto: 'bg-vm-muted' };
  }
}

/**
 * Construye el mensaje que se muestra al usuario para un error del cliente.
 *
 * @param {unknown} error Error recibido.
 * @returns {string} Mensaje comprensible.
 */
export function mensajeDeError(error) {
  if (error && error.noImplementado) {
    return 'Este módulo aún no está conectado con la máquina.';
  }

  if (error && error.mensaje) {
    return error.mensaje;
  }

  if (error instanceof TypeError) {
    return 'No fue posible contactar con la consola. Verifica que el servidor esté en ejecución.';
  }

  return 'Ocurrió un error inesperado al ejecutar la operación.';
}

/**
 * Convierte un error del cliente al objeto que consume el panel de resultado.
 *
 * @param {unknown} error Error recibido.
 * @param {string} [operacion] Orden que se intentó ejecutar.
 * @returns {{ok: boolean, mensaje: string, operacion: string, tecnico: string, codigo: number|undefined, stderr: string, solicitud: string}} Resultado fallido.
 */
export function resultadoDeError(error, operacion = '') {
  return {
    ok: false,
    mensaje: mensajeDeError(error),
    operacion,
    codigo: error && error.codigo,
    stderr: (error && error.stderr) || '',
    tecnico: (error && (error.tecnico || error.campo)) || '',
    solicitud: (error && error.solicitud) || '',
  };
}

/**
 * Pinta una fila de tabla que ocupa todas las columnas.
 *
 * @param {number} columnas Número de columnas de la tabla.
 * @param {string} texto Texto de la fila.
 * @param {string} [clase] Clases adicionales.
 * @returns {string} Fila HTML.
 */
export function filaTabla(columnas, texto, clase = 'text-vm-fgSecondary') {
  return `<tr><td colspan="${columnas}" class="px-3 py-3 text-center text-[12px] ${clase}">${escapar(texto)}</td></tr>`;
}

/**
 * Muestra una alerta flotante temporal en la pantalla para notificar el inicio y fin de una operación.
 *
 * @param {object} opciones Parámetros de la alerta.
 * @param {'progreso'|'exito'|'error'|'info'} [opciones.tipo] Tipo visual de alerta.
 * @param {string} opciones.titulo Título destacado de la alerta.
 * @param {string} [opciones.mensaje] Mensaje explicativo o detalle.
 * @param {number} [opciones.duracion] Tiempo en ms antes de ocultarse (0 = persistente).
 * @returns {{ actualizar: (nuevasOpciones: object) => void, cerrar: () => void }} Manejador de la alerta.
 */
export function mostrarAlerta({ tipo = 'info', titulo = '', mensaje = '', duracion = 0 } = {}) {
  if (typeof document === 'undefined') {
    return { actualizar: () => {}, cerrar: () => {} };
  }

  let contenedor = document.getElementById('toast-container');
  if (!contenedor) {
    contenedor = document.createElement('div');
    contenedor.id = 'toast-container';
    contenedor.className = 'fixed bottom-5 right-5 z-50 flex flex-col gap-2 max-w-sm pointer-events-none';
    document.body.appendChild(contenedor);
  }

  const toast = document.createElement('div');
  let timer = null;

  const aplicarEstilos = (t) => {
    switch (t) {
      case 'progreso':
        toast.className = 'pointer-events-auto flex items-start gap-2.5 p-3 rounded-[3px] border border-[#3B82F6] bg-[#222E3E] text-[#E5E5E5] text-[12px] shadow-2xl transition-all duration-200';
        return { icono: 'loader-2', claseIcono: 'text-[#60A5FA] animate-spin' };
      case 'exito':
        toast.className = 'pointer-events-auto flex items-start gap-2.5 p-3 rounded-[3px] border border-[#2ECC71] bg-[#1E3024] text-[#E5E5E5] text-[12px] shadow-2xl transition-all duration-200';
        return { icono: 'check-circle-2', claseIcono: 'text-[#2ECC71]' };
      case 'error':
        toast.className = 'pointer-events-auto flex items-start gap-2.5 p-3 rounded-[3px] border border-vm-accent bg-[#352120] text-[#E5E5E5] text-[12px] shadow-2xl transition-all duration-200';
        return { icono: 'alert-triangle', claseIcono: 'text-vm-accent' };
      default:
        toast.className = 'pointer-events-auto flex items-start gap-2.5 p-3 rounded-[3px] border border-[#404040] bg-[#2B2B2B] text-[#E5E5E5] text-[12px] shadow-2xl transition-all duration-200';
        return { icono: 'info', claseIcono: 'text-[#60A5FA]' };
    }
  };

  const renderContenido = (t, tit, msg) => {
    const { icono, claseIcono } = aplicarEstilos(t);
    toast.innerHTML = `
      <i data-lucide="${icono}" class="w-4 h-4 shrink-0 mt-0.5 ${claseIcono}"></i>
      <div class="flex-1 min-w-0 pr-1">
        <div class="font-medium text-[#F0F0F0] leading-tight">${escapar(tit)}</div>
        ${msg ? `<div class="text-[11px] text-[#A0AEC0] mt-1 break-words leading-normal">${escapar(msg)}</div>` : ''}
      </div>
      <button type="button" class="toast-cerrar shrink-0 text-[#888888] hover:text-[#FFFFFF] transition-colors p-0.5 ml-1" title="Cerrar">
        <i data-lucide="x" class="w-3.5 h-3.5"></i>
      </button>
    `;
    renderIcons(toast);
    toast.querySelector('.toast-cerrar')?.addEventListener('click', cerrar);
  };

  const cerrar = () => {
    if (timer) clearTimeout(timer);
    toast.classList.add('opacity-0', 'translate-y-2');
    setTimeout(() => toast.remove(), 250);
  };

  const programarCierre = (ms) => {
    if (timer) clearTimeout(timer);
    if (ms > 0) {
      timer = setTimeout(cerrar, ms);
    }
  };

  renderContenido(tipo, titulo, mensaje);
  contenedor.appendChild(toast);

  if (typeof requestAnimationFrame === 'function') {
    requestAnimationFrame(() => {
      toast.classList.remove('opacity-0', 'translate-y-2');
    });
  }

  programarCierre(duracion);

  return {
    actualizar({ tipo: nuevoTipo = tipo, titulo: nuevoTitulo = titulo, mensaje: nuevoMensaje = mensaje, duracion: nuevaDuracion = 0 }) {
      renderContenido(nuevoTipo, nuevoTitulo, nuevoMensaje);
      programarCierre(nuevaDuracion);
    },
    cerrar,
  };
}

/**
 * Aplica el atributo `disabled` a todos los controles de un contenedor y
 * muestra un indicador de carga en el botón de acción principal.
 *
 * @param {HTMLElement} raiz Contenedor con los controles.
 * @param {boolean} deshabilitado Estado que se aplica.
 */
export function inhabilitar(raiz, deshabilitado) {
  if (!raiz || typeof raiz.querySelectorAll !== 'function') {
    return;
  }

  raiz.querySelectorAll('button, input, select, textarea').forEach((control) => {
    control.disabled = deshabilitado;

    if (control.tagName === 'BUTTON' && (control.type === 'submit' || control.classList.contains('btn-primary'))) {
      if (deshabilitado) {
        if (!control.dataset.prevHtml) {
          control.dataset.prevHtml = control.innerHTML;
        }
        control.innerHTML = '<i data-lucide="loader-2" class="w-3.5 h-3.5 animate-spin inline-block mr-1"></i><span>Ejecutando…</span>';
        renderIcons(control);
      } else if (control.dataset.prevHtml) {
        control.innerHTML = control.dataset.prevHtml;
        delete control.dataset.prevHtml;
        renderIcons(control);
      }
    }
  });
}

/**
 * Ejecuta una operación mostrando su avance en el panel de resultado y
 * notificando al usuario mediante una alerta flotante inmediata.
 *
 * @param {object} opciones Opciones de la ejecución.
 * @param {() => Promise<object>} opciones.accion Operación que produce un `ResultadoOperacion`.
 * @param {string} opciones.operacion Descripción de la orden ejecutada.
 * @param {() => void} [opciones.recargar] Consulta que se repite si la operación tiene éxito.
 * @param {() => void} opciones.informar Función que entrega el resultado al panel.
 * @returns {Promise<boolean>} `true` si la operación terminó correctamente.
 */
export async function ejecutar({ accion, operacion, recargar, informar }) {
  if (typeof document !== 'undefined') {
    document.dispatchEvent(new CustomEvent('vmcons:progreso', { detail: { operacion } }));
  }

  const alerta = mostrarAlerta({
    tipo: 'progreso',
    titulo: 'Operación iniciada',
    mensaje: operacion ? `Ejecutando «${operacion}»…` : 'Procesando en la máquina virtual…',
  });

  try {
    const resultado = await accion();

    const resultadoFinal = {
      ok: true,
      // Si el servidor no explicó el resultado (por ejemplo, un 204 sin
      // cuerpo), se usa la orden ejecutada para no dejar un mensaje vago.
      mensaje: (resultado && resultado.mensaje) || '',
      operacion: (resultado && resultado.operacion) || operacion,
      codigo: (resultado && resultado.codigo) || 0,
      salida: (resultado && resultado.salida) || '',
      stderr: (resultado && resultado.stderr) || '',
      tecnico: (resultado && resultado.tecnico) || '',
    };

    informar(resultadoFinal);

    alerta.actualizar({
      tipo: 'exito',
      titulo: 'Operación completada',
      mensaje: resultadoFinal.mensaje || (operacion ? `Se completó la orden «${operacion}».` : 'La operación concluyó correctamente.'),
      duracion: 3500,
    });

    if (recargar) {
      recargar();
    }

    return true;
  } catch (error) {
    const errorResultado = resultadoDeError(error, operacion);
    informar(errorResultado);

    alerta.actualizar({
      tipo: 'error',
      titulo: 'Error en la operación',
      mensaje: errorResultado.mensaje || (error && error.message) || 'Ocurrió un fallo al ejecutar la orden.',
      duracion: 6000,
    });

    return false;
  }
}

