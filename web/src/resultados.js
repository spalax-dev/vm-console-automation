/**
 * Historial de resultados de operaciones, guardado en el navegador.
 *
 * El panel de resultado se reconstruye al cambiar de módulo y al
 * recargar la página, así que su contenido no puede vivir en el DOM: se
 * guarda en `localStorage` y se lee al montar. Se conservan los últimos
 * resultados, del más antiguo al más reciente.
 */

/** Clave con la que se guarda el historial. */
const CLAVE = 'vmcons:resultados';

/** Número máximo de resultados conservados. */
const MAXIMO = 50;

/** Respaldo en memoria cuando `localStorage` no está disponible. */
let memoria = [];

/** Máquina activa para separar los resultados. */
let maquinaActiva = '';

/**
 * Establece la máquina a la que pertenecen los resultados nuevos.
 *
 * @param {string} vm Nombre de la máquina.
 */
export function setActiveVM(vm) {
  maquinaActiva = vm ?? '';
}

/**
 * Devuelve la máquina activa.
 *
 * @returns {string} Nombre de la máquina.
 */
export function getActiveVM() {
  return maquinaActiva;
}

/**
 * Devuelve el almacenamiento local, o `null` si el navegador lo bloquea.
 *
 * @returns {Storage|null} Almacenamiento disponible.
 */
function almacen() {
  try {
    return window.localStorage;
  } catch {
    return null;
  }
}

/**
 * Comprueba que un valor tenga la forma de un resultado guardado.
 *
 * @param {unknown} valor Valor leído.
 * @returns {boolean} `true` si se puede mostrar.
 */
function esResultado(valor) {
  return Boolean(valor) && typeof valor === 'object' && typeof valor.mensaje === 'string';
}

/**
 * Lee el historial guardado. Si el contenido está corrupto o el
 * almacenamiento no está disponible, devuelve el respaldo en memoria.
 *
 * @returns {Array<object>} Resultados del más antiguo al más reciente.
 */
export function leer() {
  const guardado = almacen();

  if (!guardado) {
    return [...memoria];
  }

  try {
    const crudo = JSON.parse(guardado.getItem(CLAVE) ?? '[]');

    if (!Array.isArray(crudo)) {
      return [];
    }

    return crudo.filter(esResultado);
  } catch {
    return [];
  }
}

/**
 * Añade un resultado al final del historial y lo persiste.
 *
 * @param {object} resultado Resultado de la operación.
 * @returns {Array<object>} Historial resultante.
 */
export function agregar(resultado) {
  const entrada = { ...resultado, vm: resultado.vm ?? maquinaActiva };
  const historial = [...leer(), entrada].slice(-MAXIMO);

  return escribir(historial);
}

/**
 * Vacía el historial.
 *
 * @returns {Array<object>} Historial vacío.
 */
export function limpiar(vm = maquinaActiva) {
  return escribir(leer().filter((resultado) => (resultado.vm ?? '') !== vm));
}

/**
 * Guarda el historial en el almacenamiento y devuelve lo guardado.
 *
 * @param {Array<object>} historial Resultados a guardar.
 * @returns {Array<object>} Historial guardado.
 */
function escribir(historial) {
  memoria = historial;

  const guardado = almacen();

  if (guardado) {
    try {
      guardado.setItem(CLAVE, JSON.stringify(historial));
    } catch {
      // Cuota llena o almacenamiento bloqueado: queda el respaldo.
    }
  }

  return historial;
}
