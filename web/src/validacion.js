/**
 * Validaciones del contrato OpenAPI (`docs/openapi.yaml`).
 *
 * Cada función devuelve `null` cuando el valor cumple la regla o un mensaje
 * comprensible cuando no. El servidor sigue siendo la autoridad: estas
 * comprobaciones solo evitan un viaje de ida y vuelta innecesario.
 */

/** Patrón de nombre de paquete APT. */
const PAQUETE = /^[a-z0-9][a-z0-9+.-]*$/;

/** Patrón de nombre de usuario POSIX. */
const USUARIO = /^[a-z_][a-z0-9_-]{0,31}$/;

/** Patrón de nombre de host según RFC 1123. */
const HOST = /^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$/;

/** Patrón de unidad systemd con extensión `.service`. */
const UNIDAD = /^[A-Za-z0-9_.@-]+\.service$/;

/** Patrón de permisos en octal. */
const MODO = /^[0-7]{3,4}$/;

/**
 * Valida un nombre de paquete APT.
 *
 * @param {string} valor Nombre propuesto.
 * @returns {string|null} Mensaje de error o `null`.
 */
export function validarNombrePaquete(valor) {
  if (typeof valor !== 'string' || !PAQUETE.test(valor)) {
    return 'El nombre del paquete solo admite minúsculas, dígitos y los signos + . -';
  }

  return null;
}

/**
 * Valida un nombre de usuario POSIX.
 *
 * @param {string} valor Nombre propuesto.
 * @returns {string|null} Mensaje de error o `null`.
 */
export function validarNombreUsuario(valor) {
  if (typeof valor !== 'string' || !USUARIO.test(valor)) {
    return 'El nombre de usuario debe iniciar con minúscula o guion bajo y usar hasta 32 caracteres';
  }

  return null;
}

/**
 * Valida un nombre de host.
 *
 * @param {string} valor Nombre propuesto.
 * @returns {string|null} Mensaje de error o `null`.
 */
export function validarNombreHost(valor) {
  if (typeof valor !== 'string' || !HOST.test(valor)) {
    return 'El nombre de host debe cumplir RFC 1123: minúsculas, dígitos y guiones internos';
  }

  return null;
}

/**
 * Valida una unidad systemd.
 *
 * @param {string} valor Nombre de la unidad.
 * @returns {string|null} Mensaje de error o `null`.
 */
export function validarNombreUnidad(valor) {
  if (typeof valor !== 'string' || !UNIDAD.test(valor)) {
    return 'La unidad debe tener una extensión .service y usar letras, dígitos o _ . @ -';
  }

  return null;
}

/**
 * Comprueba que una cadena sea una dirección IPv4 con octetos válidos.
 *
 * @param {string} valor Dirección propuesta.
 * @returns {boolean} `true` si es una IPv4 válida.
 */
export function esIpv4(valor) {
  if (typeof valor !== 'string') {
    return false;
  }

  const partes = valor.split('.');

  if (partes.length !== 4) {
    return false;
  }

  return partes.every((parte) => /^\d{1,3}$/.test(parte) && Number(parte) <= 255);
}

/**
 * Comprueba que un valor sea un puerto TCP en el rango 1–65535.
 *
 * @param {number} valor Puerto propuesto.
 * @returns {boolean} `true` si el puerto es válido.
 */
export function esPuerto(valor) {
  return typeof valor === 'number' && Number.isInteger(valor) && valor >= 1 && valor <= 65535;
}

/**
 * Comprueba que una cadena sea un modo octal de tres o cuatro dígitos.
 *
 * @param {string} valor Permisos propuestos.
 * @returns {boolean} `true` si los permisos son válidos.
 */
export function esModo(valor) {
  return typeof valor === 'string' && MODO.test(valor);
}

/**
 * Comprueba que una cadena sea una ruta absoluta.
 *
 * @param {string} valor Ruta propuesta.
 * @returns {boolean} `true` si la ruta es absoluta.
 */
export function esRutaAbsoluta(valor) {
  return typeof valor === 'string' && valor.startsWith('/') && valor.length > 1;
}

/**
 * Compone el identificador del servidor del back-end.
 *
 * @param {string} direccion Dirección IPv4.
 * @param {number} puerto Puerto TCP.
 * @returns {string} Identificador `direccion:puerto`.
 */
export function identificadorServidor(direccion, puerto) {
  return `${direccion}:${puerto}`;
}
