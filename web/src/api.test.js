import { afterEach, describe, expect, it, vi } from 'vitest';

import { ApiError, api } from './api.js';

/**
 * Sustituye `fetch` por un doble que devuelve la respuesta indicada y
 * registra la petición recibida.
 *
 * @param {object} respuesta Respuesta simulada: `estado`, `cuerpo`, `tipo`, `blob`.
 * @returns {{llamada: () => {url: string, opciones: object}}}
 */
function simularFetch(respuesta) {
  const llamadas = [];

  vi.stubGlobal('fetch', async (url, opciones) => {
    llamadas.push({ url, opciones: opciones ?? {} });

    const estado = respuesta.estado ?? 200;
    const tipo = respuesta.tipo ?? 'application/json';
    const cuerpo = respuesta.cuerpo;

    return {
      ok: estado >= 200 && estado < 300,
      status: estado,
      headers: { get: (clave) => (clave.toLowerCase() === 'content-type' ? tipo : null) },
      json: async () => cuerpo,
      text: async () => (typeof cuerpo === 'string' ? cuerpo : JSON.stringify(cuerpo)),
      blob: async () => respuesta.blob ?? new Blob(['llave']),
    };
  });

  return { llamada: () => llamadas[0] };
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('cliente de la API', () => {
  it('busca máquinas en el inventario con el filtro en la consulta', async () => {
    const { llamada } = simularFetch({ cuerpo: { maquinas: [] } });

    await api.maquinas.listar('deb');

    expect(llamada().url).toBe('/api/vms?q=deb');
    expect(llamada().opciones.method).toBe('GET');
  });

  it('pide el inventario completo cuando no hay filtro', async () => {
    const { llamada } = simularFetch({ cuerpo: { maquinas: [] } });

    await api.maquinas.listar('   ');

    expect(llamada().url).toBe('/api/vms');
  });

  it('vincula la consola con la máquina elegida', async () => {
    const { llamada } = simularFetch({
      cuerpo: { nombre: 'debian-1', uuid: 'uuid-1', estado: 'running', activa: true },
    });

    const maquina = await api.sesion.vincular('debian-1');

    expect(maquina.activa).toBe(true);
    expect(llamada().url).toBe('/api/session');
    expect(llamada().opciones.method).toBe('PUT');
    expect(llamada().opciones.body).toBe(JSON.stringify({ vm: 'debian-1' }));
  });

  it('enciende una máquina con POST y el nombre en la ruta', async () => {
    const { llamada } = simularFetch({
      cuerpo: { nombre: 'debian-1', uuid: 'uuid-1', estado: 'running', activa: true },
    });

    const maquina = await api.maquinas.encender('debian-1');

    expect(maquina.estado).toBe('running');
    expect(llamada().url).toBe('/api/vms/debian-1/poweron');
    expect(llamada().opciones.method).toBe('POST');
  });

  it('consulta la sesión con GET /api/session', async () => {
    const { llamada } = simularFetch({
      cuerpo: { conector: 'ssh', ip: '192.168.1.50', estado: 'running' },
    });

    const sesion = await api.sesion.obtener();

    expect(sesion.estado).toBe('running');
    expect(llamada().url).toBe('/api/session');
    expect(llamada().opciones.method).toBe('GET');
  });

  it('lista los paquetes con GET /api/packages', async () => {
    const { llamada } = simularFetch({ cuerpo: { paquetes: [] } });

    const lista = await api.paquetes.listar();

    expect(lista.paquetes).toEqual([]);
    expect(llamada().url).toBe('/api/packages');
  });

  it('instala un paquete con POST y el nombre en el cuerpo', async () => {
    const { llamada } = simularFetch({ estado: 201, cuerpo: { ok: true, mensaje: 'Instalado' } });

    const resultado = await api.paquetes.instalar('python3-pip');

    expect(resultado.ok).toBe(true);
    expect(llamada().url).toBe('/api/packages');
    expect(llamada().opciones.method).toBe('POST');
    expect(llamada().opciones.body).toBe(JSON.stringify({ nombre: 'python3-pip' }));
    expect(llamada().opciones.headers['Content-Type']).toBe('application/json');
  });

  it('elimina un paquete con el nombre en la ruta y resuelve null en 204', async () => {
    const { llamada } = simularFetch({ estado: 204, tipo: '' });

    const resultado = await api.paquetes.eliminar('nano');

    expect(resultado).toBeNull();
    expect(llamada().url).toBe('/api/packages/nano');
    expect(llamada().opciones.method).toBe('DELETE');
  });

  it('escapa el identificador del servidor del back-end', async () => {
    const { llamada } = simularFetch({ estado: 204, tipo: '' });

    await api.haproxy.eliminar('192.168.1.61:8080');

    expect(llamada().url).toBe('/api/haproxy/backend/servers/192.168.1.61%3A8080');
    expect(llamada().opciones.method).toBe('DELETE');
  });

  it('envía los indicadores de la eliminación de usuario como consulta', async () => {
    const { llamada } = simularFetch({ estado: 204, tipo: '' });

    await api.usuarios.eliminar('jorge', { archivos: true });

    expect(llamada().url).toBe('/api/users/jorge?files=true');
  });

  it('omite la consulta cuando la opción está desactivada', async () => {
    const { llamada } = simularFetch({ estado: 204, tipo: '' });

    await api.usuarios.eliminar('jorge', { archivos: false });

    expect(llamada().url).toBe('/api/users/jorge');
  });

  it('cambia el estado de una unidad con PATCH y una sola propiedad', async () => {
    const { llamada } = simularFetch({
      cuerpo: { unidad: 'catalogo.service', activo: false, habilitado: true, estado: 'inactive' },
    });

    const estado = await api.unidades.cambiarEstado('catalogo.service', { activo: false });

    expect(estado.estado).toBe('inactive');
    expect(llamada().opciones.method).toBe('PATCH');
    expect(llamada().opciones.body).toBe(JSON.stringify({ activo: false }));
  });

  it('despliega una unidad con PUT y el cuerpo de archivos', async () => {
    const { llamada } = simularFetch({ cuerpo: { ok: true, mensaje: 'Desplegado' } });
    const solicitud = { archivos: [{ origen: './app', destino: '/opt/app', modo: '755' }] };

    await api.unidades.desplegar('catalogo.service', solicitud);

    expect(llamada().opciones.method).toBe('PUT');
    expect(llamada().opciones.body).toBe(JSON.stringify(solicitud));
  });

  it('reinicia una unidad con POST al subrecurso de reinicios', async () => {
    const { llamada } = simularFetch({ estado: 202, cuerpo: { ok: true, mensaje: 'Reiniciado' } });

    const resultado = await api.unidades.reiniciar('catalogo.service');

    expect(resultado.ok).toBe(true);
    expect(llamada().url).toBe('/api/units/catalogo.service/restarts');
    expect(llamada().opciones.method).toBe('POST');
  });

  it('entrega la llave privada como Blob', async () => {
    const blob = new Blob(['-----BEGIN OPENSSH PRIVATE KEY-----']);
    const { llamada } = simularFetch({ tipo: 'application/octet-stream', blob });

    const archivo = await api.llaves.descargarRoot();

    expect(archivo).toBe(blob);
    expect(llamada().url).toBe('/api/keys/root');
    expect(llamada().opciones.method).toBe('GET');
  });

  it('convierte una respuesta 400 en ApiError con el campo rechazado', async () => {
    simularFetch({
      estado: 400,
      cuerpo: { ok: false, mensaje: 'el campo "nombre" no es válido', campo: 'nombre' },
    });

    const error = await api.paquetes.instalar('Nano!').catch((e) => e);

    expect(error).toBeInstanceOf(ApiError);
    expect(error.estado).toBe(400);
    expect(error.campo).toBe('nombre');
    expect(error.mensaje).toContain('no es válido');
    expect(error.noImplementado).toBe(false);
  });

  it('convierte una respuesta 501 en ApiError marcado como no implementado', async () => {
    simularFetch({ estado: 501, cuerpo: { ok: false, mensaje: 'La operación aún no está conectada.' } });

    const error = await api.usuarios.listar().catch((e) => e);

    expect(error).toBeInstanceOf(ApiError);
    expect(error.noImplementado).toBe(true);
    expect(error.mensaje).toBe('La operación aún no está conectada.');
  });

  it('conserva el detalle técnico de una respuesta 503', async () => {
    simularFetch({
      estado: 503,
      cuerpo: { ok: false, mensaje: 'No hay una sesión disponible con la máquina: enciéndela o revisa la red.', tecnico: 'sin conector' },
    });

    const error = await api.sesion.obtener().catch((e) => e);

    expect(error.estado).toBe(503);
    expect(error.tecnico).toBe('sin conector');
  });
});
