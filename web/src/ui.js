/**
 * Utilidades de presentación compartidas por las vistas.
 *
 * Reúne el escapado de texto, la traducción de estados del contrato y la
 * construcción del objeto que consume el panel de resultado.
 */

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
 * Aplica el atributo `disabled` a todos los controles de un contenedor.
 *
 * @param {HTMLElement} raiz Contenedor con los controles.
 * @param {boolean} deshabilitado Estado que se aplica.
 */
export function inhabilitar(raiz, deshabilitado) {
  raiz.querySelectorAll('button, input, select, textarea').forEach((control) => {
    control.disabled = deshabilitado;
  });
}

/**
 * Ejecuta una operación mostrando su avance en el panel de resultado.
 *
 * @param {object} opciones Opciones de la ejecución.
 * @param {() => Promise<object>} opciones.accion Operación que produce un `ResultadoOperacion`.
 * @param {string} opciones.operacion Descripción de la orden ejecutada.
 * @param {() => void} [opciones.recargar] Consulta que se repite si la operación tiene éxito.
 * @param {() => void} opciones.informar Función que entrega el resultado al panel.
 * @returns {Promise<boolean>} `true` si la operación terminó correctamente.
 */
export async function ejecutar({ accion, operacion, recargar, informar }) {
  try {
    const resultado = await accion();

    informar({
      ok: true,
      mensaje: (resultado && resultado.mensaje) || 'La operación terminó correctamente.',
      operacion: (resultado && resultado.operacion) || operacion,
      codigo: (resultado && resultado.codigo) || 0,
      salida: (resultado && resultado.salida) || '',
      stderr: (resultado && resultado.stderr) || '',
      tecnico: (resultado && resultado.tecnico) || '',
    });

    if (recargar) {
      recargar();
    }

    return true;
  } catch (error) {
    informar(resultadoDeError(error, operacion));

    return false;
  }
}
