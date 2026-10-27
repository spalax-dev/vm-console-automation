/**
 * Cliente HTTP del contrato OpenAPI de `vmcons` (`docs/openapi.yaml`).
 *
 * Ninguna vista llama a `fetch` directamente: todas las rutas, los cuerpos y
 * los errores del contrato se resuelven aquí. La base es relativa (`/api`)
 * porque la SPA se sirve desde el mismo origen que la API; el proxy de Vite
 * redirige `/api` al servidor Go durante el desarrollo.
 */

/** Prefijo común de todas las rutas de la API. */
const BASE = '/api';

/** Cuerpo JSON que se envía por omisión. */
const CABECERAS_JSON = {
  Accept: 'application/json',
  'Content-Type': 'application/json',
};

/**
 * Error normalizado de una respuesta fallida de la consola.
 *
 * Conserva el código HTTP, el mensaje de la envoltura
 * `{ok, mensaje, campo, tecnico}` y el detalle técnico para el panel de
 * resultado.
 */
export class ApiError extends Error {
  /**
   * Construye el error con la respuesta recibida.
   *
   * @param {number} estado Código HTTP de la respuesta.
   * @param {{mensaje?: string, campo?: string, tecnico?: string}|null} cuerpo Cuerpo de la respuesta.
   */
  constructor(estado, cuerpo) {
    super((cuerpo && cuerpo.mensaje) || `La consola respondió con el código ${estado}.`);

    /** Código HTTP recibido. */
    this.name = 'ApiError';
    /** Código HTTP recibido. */
    this.estado = estado;
    /** Mensaje comprensible para el usuario. */
    this.mensaje = this.message;
    /** Campo rechazado por la validación, cuando aplica. */
    this.campo = cuerpo && cuerpo.campo;
    /** Detalle técnico de la respuesta. */
    this.tecnico = cuerpo && cuerpo.tecnico;
    /** Identificador de la solicitud para correlacionar con los logs. */
    this.solicitud = '';
  }

  /**
   * Indica si la operación existe en el contrato pero aún no está conectada
   * con la máquina.
   *
   * @returns {boolean} `true` para una respuesta `501`.
   */
  get noImplementado() {
    return this.estado === 501;
  }
}

/**
 * Añade la cadena de consulta a una ruta, omitiendo los valores vacíos y los
 * indicadores en `false`.
 *
 * @param {string} ruta Ruta relativa al prefijo de la API.
 * @param {Record<string, string|number|boolean|undefined>|undefined} consulta Parámetros.
 * @returns {string} URL completa.
 */
function conConsulta(ruta, consulta) {
  if (!consulta) {
    return BASE + ruta;
  }

  const partes = Object.entries(consulta)
    .filter(([, valor]) => valor !== undefined && valor !== null && valor !== false)
    .map(([clave, valor]) => `${encodeURIComponent(clave)}=${encodeURIComponent(String(valor))}`);

  return partes.length > 0 ? `${BASE}${ruta}?${partes.join('&')}` : BASE + ruta;
}

/**
 * Ejecuta una petición y traduce la respuesta al valor del contrato.
 *
 * @param {string} metodo Verbo HTTP.
 * @param {string} ruta Ruta relativa al prefijo de la API.
 * @param {object} [opciones] Opciones de la petición.
 * @param {unknown} [opciones.cuerpo] Cuerpo JSON que se serializa.
 * @param {Record<string, string|number|boolean|undefined>} [opciones.consulta] Cadena de consulta.
 * @param {boolean} [opciones.descarga] Si la respuesta es un archivo binario.
 * @returns {Promise<unknown>} Cuerpo JSON, `Blob` o `null` para `204`.
 * @throws {ApiError} Cuando la respuesta no es satisfactoria.
 */
async function pedir(metodo, ruta, { cuerpo, consulta, descarga = false } = {}) {
  const opciones = { method: metodo, headers: { Accept: 'application/json' } };

  if (cuerpo !== undefined) {
    opciones.headers = CABECERAS_JSON;
    opciones.body = JSON.stringify(cuerpo);
  }

  const respuesta = await fetch(conConsulta(ruta, consulta), opciones);
  const solicitud = respuesta.headers.get('X-Request-ID') || '';

  if (respuesta.status === 204) {
    return null;
  }

  const tipo = respuesta.headers.get('content-type') || '';

  if (!respuesta.ok) {
    let detalle = null;

    if (tipo.includes('application/json')) {
      try {
        detalle = await respuesta.json();
      } catch {
        detalle = null;
      }
    }

    const error = new ApiError(respuesta.status, detalle);
    error.solicitud = solicitud;

    throw error;
  }

  if (descarga) {
    return respuesta.blob();
  }

  if (tipo.includes('application/json')) {
    return respuesta.json();
  }

  return respuesta.text();
}

/**
 * Descarga un archivo y lo entrega al navegador con el nombre indicado.
 *
 * @param {Blob} archivo Contenido descargado.
 * @param {string} nombre Nombre sugerido para guardar.
 */
export function guardarArchivo(archivo, nombre) {
  const url = URL.createObjectURL(archivo);
  const enlace = document.createElement('a');

  enlace.href = url;
  enlace.download = nombre;
  document.body.appendChild(enlace);
  enlace.click();
  enlace.remove();
  URL.revokeObjectURL(url);
}

/** Operaciones del contrato, agrupadas por recurso. */
export const api = {
  /** Autenticación de la consola (`/api/auth`). */
  auth: {
    /** @returns {Promise<{configurada: boolean, autenticado: boolean, usuario?: string}>} */
    estado: () => pedir('GET', '/auth/estado'),
    /**
     * Crea la primera cuenta de la consola y deja la sesión iniciada.
     *
     * @param {string} usuario Usuario de la consola.
     * @param {string} password Contraseña de la consola.
     * @returns {Promise<object>} Estado de la autenticación.
     */
    configurar: (usuario, password) =>
      pedir('POST', '/auth/setup', { cuerpo: { usuario, password } }),
    /**
     * Inicia sesión con la cuenta existente.
     *
     * @param {string} usuario Usuario de la consola.
     * @param {string} password Contraseña de la consola.
     * @returns {Promise<object>} Estado de la autenticación.
     */
    entrar: (usuario, password) =>
      pedir('POST', '/auth/login', { cuerpo: { usuario, password } }),
    /** @returns {Promise<null>} Sin cuerpo; vence la cookie. */
    salir: () => pedir('POST', '/auth/logout'),
  },

  /** Sesión con la máquina virtual (`GET /api/session`). */
  sesion: {
    /** @returns {Promise<{vm: string, conector: string, ip: string, estado: string}>} */
    obtener: () => pedir('GET', '/session'),
    /**
     * Vincula la consola con una máquina del inventario.
     *
     * @param {string} vm Nombre o UUID de la máquina.
     * @returns {Promise<object>} Máquina vinculada.
     */
    vincular: (vm) => pedir('PUT', '/session', { cuerpo: { vm } }),
    /**
     * Registra la máquina vinculada: guarda sus credenciales y prepara SSH.
     *
     * @param {string} usuario Usuario con permisos en la máquina.
     * @param {string} password Contraseña del usuario.
     * @returns {Promise<object>} Resultado del registro.
     */
    registrar: (usuario, password) =>
      pedir('POST', '/session/register', { cuerpo: { usuario, password } }),
    /** @returns {Promise<null>} Sin cuerpo. */
    olvidar: () => pedir('DELETE', '/session/register'),
    /** @returns {Promise<object>} Diagnóstico de la máquina vinculada. */
    diagnostico: () => pedir('GET', '/session/diagnostico'),
  },

  /** Historial de operaciones (`GET /api/historial`). */
  historial: {
    /**
     * @param {object} filtros Filtros de la consulta.
     * @returns {Promise<{operaciones: Array<object>}>}
     */
    listar: (filtros) => pedir('GET', '/historial', { consulta: filtros }),
  },

  /** Inventario de máquinas del hipervisor (`GET /api/vms`). */
  maquinas: {
    /**
     * @param {string} [filtro] Texto de búsqueda sobre el nombre o el UUID.
     * @returns {Promise<{maquinas: Array<object>}>}
     */
    listar: (filtro) =>
      pedir('GET', '/vms', { consulta: { q: filtro && filtro.trim() ? filtro.trim() : undefined } }),
    /**
     * Enciende la máquina en modo sin interfaz gráfica.
     *
     * @param {string} nombre Nombre o UUID de la máquina.
     * @returns {Promise<object>} Máquina ya en ejecución.
     */
    encender: (nombre) => pedir('POST', `/vms/${encodeURIComponent(nombre)}/poweron`),
  },

  /** Paquetes APT (numeral 3.1.1). */
  paquetes: {
    /** @returns {Promise<{paquetes: Array<object>}>} */
    listar: () => pedir('GET', '/packages'),
    /**
     * @param {string} nombre Nombre del paquete.
     * @returns {Promise<object>} Resultado de la operación.
     */
    instalar: (nombre) => pedir('POST', '/packages', { cuerpo: { nombre } }),
    /**
     * @param {string} nombre Nombre del paquete.
     * @returns {Promise<null>} Sin cuerpo.
     */
    eliminar: (nombre) => pedir('DELETE', `/packages/${encodeURIComponent(nombre)}`),
  },

  /** Llaves SSH de root (numeral 3.1.2). */
  llaves: {
    /** @returns {Promise<object>} Resultado de la generación. */
    generarRoot: () => pedir('POST', '/keys/root'),
    /** @returns {Promise<{existe: boolean}>} Estado de la llave de root. */
    estadoRoot: () => pedir('GET', '/keys/root/status'),
    /** @returns {Promise<Blob>} Llave privada. */
    descargarRoot: () => pedir('GET', '/keys/root', { descarga: true }),
  },

  /** Banner `/etc/issue` (numeral 3.1.3). */
  issue: {
    /** @returns {Promise<object>} Configuración actual. */
    obtener: () => pedir('GET', '/issue'),
    /**
     * @param {{mensaje: string, nombre_host: boolean, fecha: boolean, hora: boolean, ip: boolean}} configuracion Configuración.
     * @returns {Promise<object>} Resultado de la operación.
     */
    reemplazar: (configuracion) => pedir('PUT', '/issue', { cuerpo: configuracion }),
  },

  /** Nombre de host (numeral 3.1.4). */
  hostname: {
    /** @returns {Promise<object>} Configuración actual. */
    obtener: () => pedir('GET', '/hostname'),
    /**
     * @param {{nombre: string, ip: string, actualizar_hosts: boolean}} configuracion Configuración.
     * @returns {Promise<object>} Resultado de la operación.
     */
    reemplazar: (configuracion) => pedir('PUT', '/hostname', { cuerpo: configuracion }),
  },

  /** Cuentas locales (numeral 3.2). */
  usuarios: {
    /** @returns {Promise<{usuarios: Array<object>}>} */
    listar: () => pedir('GET', '/users'),
    /**
     * @param {{nombre: string, sudo: boolean, llaves: boolean}} solicitud Datos del usuario.
     * @returns {Promise<object>} Resultado de la operación.
     */
    crear: (solicitud) => pedir('POST', '/users', { cuerpo: solicitud }),
    /**
     * @param {string} nombre Nombre del usuario.
     * @returns {Promise<object>} Datos del usuario.
     */
    obtener: (nombre) => pedir('GET', `/users/${encodeURIComponent(nombre)}`),
    /**
     * @param {string} nombre Nombre del usuario.
     * @param {{archivos?: boolean}} [opciones] Eliminar también el directorio personal.
     * @returns {Promise<null>} Sin cuerpo.
     */
    eliminar: (nombre, opciones = {}) =>
      pedir('DELETE', `/users/${encodeURIComponent(nombre)}`, {
        consulta: { files: opciones.archivos ? true : undefined },
      }),
    /**
     * @param {string} nombre Nombre del usuario.
     * @returns {Promise<Blob>} Llave privada.
     */
    descargarLlave: (nombre) => pedir('GET', `/users/${encodeURIComponent(nombre)}/key`, { descarga: true }),
    /**
     * Genera e instala un par de llaves para el usuario.
     *
     * @param {string} nombre Nombre del usuario.
     * @param {{regenerar?: boolean}} [opciones] Reemplazar el par existente.
     * @returns {Promise<object>} Resultado de la operación.
     */
    generarLlave: (nombre, opciones = {}) =>
      pedir('POST', `/users/${encodeURIComponent(nombre)}/key`, {
        consulta: { regenerar: opciones.regenerar ? true : undefined },
      }),
  },

  /** Unidades systemd (numeral 3.3). */
  unidades: {
    /** @returns {Promise<{unidades: Array<object>}>} */
    listar: () => pedir('GET', '/units'),
    /**
     * @param {string} unidad Nombre de la unidad.
     * @returns {Promise<object>} Estado de la unidad.
     */
    obtener: (unidad) => pedir('GET', `/units/${encodeURIComponent(unidad)}`),
    /**
     * Devuelve la salida de `systemctl status` para una unidad.
     *
     * @param {string} unidad Nombre de la unidad.
     * @returns {Promise<object>} Resultado de la orden.
     */
    estadoDetallado: (unidad) => pedir('GET', `/units/${encodeURIComponent(unidad)}/status`),
    /**
     * @param {string} unidad Nombre de la unidad.
     * @param {object} solicitud Archivos que se despliegan.
     * @returns {Promise<object>} Resultado de la operación.
     */
    desplegar: (unidad, solicitud) => pedir('PUT', `/units/${encodeURIComponent(unidad)}`, { cuerpo: solicitud }),
    /**
     * @param {string} unidad Nombre de la unidad.
     * @param {{activo?: boolean, habilitado?: boolean}} cambio Una sola propiedad.
     * @returns {Promise<object>} Estado actualizado.
     */
    cambiarEstado: (unidad, cambio) => pedir('PATCH', `/units/${encodeURIComponent(unidad)}`, { cuerpo: cambio }),
    /**
     * @param {string} unidad Nombre de la unidad.
     * @returns {Promise<object>} Resultado de la operación.
     */
    reiniciar: (unidad) => pedir('POST', `/units/${encodeURIComponent(unidad)}/restarts`),
  },

  /** Servidores del back-end de HAProxy (numeral 3.4). */
  haproxy: {
    /** @returns {Promise<{servidores: Array<object>}>} */
    listar: () => pedir('GET', '/haproxy/backend/servers'),
    /** @returns {Promise<object>} Estado de HAProxy. */
    estado: () => pedir('GET', '/haproxy/status'),
    /** @returns {Promise<object>} Resultado de la configuración rápida. */
    configurar: () => pedir('POST', '/haproxy/setup'),
    /**
     * @param {{direccion: string, puerto: number}} servidor Servidor que se agrega.
     * @returns {Promise<object>} Resultado de la operación.
     */
    agregar: (servidor) => pedir('POST', '/haproxy/backend/servers', { cuerpo: servidor }),
    /**
     * @param {string} identificador Servidor como `direccion:puerto`.
     * @returns {Promise<null>} Sin cuerpo.
     */
    eliminar: (identificador) =>
      pedir('DELETE', `/haproxy/backend/servers/${encodeURIComponent(identificador)}`),
  },
};
