const API_BASE = '/api';

async function request(method, url, body = null) {
  const opts = {
    method,
    headers: {
      'Accept': 'application/json',
    },
  };
  if (body) {
    if (body instanceof FormData) {
      opts.body = body;
    } else {
      opts.headers['Content-Type'] = 'application/json';
      opts.body = JSON.stringify(body);
    }
  }
  const res = await fetch(API_BASE + url, opts);
  const contentType = res.headers.get('content-type') || '';
  if (contentType.includes('application/json')) {
    const json = await res.json();
    if (!res.ok) throw { status: res.status, ...json };
    return json;
  }
  if (!res.ok) throw { status: res.status, text: await res.text() };
  return res;
}

// Cliente REST de la consola. Los nombres de las funciones conservan la
// forma que usan las vistas; las rutas siguen docs/openapi.yaml.
export const api = {
  getSesion: () => request('GET', '/session'),

  listarPaquetes: () => request('GET', '/packages'),
  paquetes: ({ accion, paquete }) =>
    accion === 'remove'
      ? request('DELETE', `/packages/${encodeURIComponent(paquete)}`)
      : request('POST', '/packages', { nombre: paquete }),

  sshRootGenerar: () => request('POST', '/keys/root'),
  getSshRootLlave: async () => {
    const res = await request('GET', '/keys/root');
    return res.blob();
  },

  getIssue: () => request('GET', '/issue'),
  setIssue: (data) => request('PUT', '/issue', data),

  getHostname: () => request('GET', '/hostname'),
  setHostname: (data) => request('PUT', '/hostname', data),

  listarUsuarios: () => request('GET', '/users'),
  crearUsuario: (data) => request('POST', '/users', data),
  getUsuario: (nombre) => request('GET', `/users/${encodeURIComponent(nombre)}`),
  eliminarUsuario: ({ nombre, archivos = false }) =>
    request('DELETE', `/users/${encodeURIComponent(nombre)}?files=${archivos ? 'true' : 'false'}`),
  getUsuarioLlave: async (nombre) => {
    const res = await request('GET', `/users/${encodeURIComponent(nombre)}/key`);
    return res.blob();
  },

  listarUnidades: () => request('GET', '/units'),
  desplegarSystemd: ({ unidad, ...data }) => {
    const archivos = (data.archivos || []).map((a) => ({
      origen: a.origen,
      destino: a.destino,
      modo: a.modo,
    }));
    return request('PUT', `/units/${encodeURIComponent(unidad)}`, {
      archivos,
      usuario: data.usuario,
      grupo: data.grupo,
    });
  },
  systemdAccion: (unidad, accion) => {
    const ruta = `/units/${encodeURIComponent(unidad)}`;
    switch (accion) {
      case 'start':
        return request('PATCH', ruta, { activo: true });
      case 'stop':
        return request('PATCH', ruta, { activo: false });
      case 'enable':
        return request('PATCH', ruta, { habilitado: true });
      case 'disable':
        return request('PATCH', ruta, { habilitado: false });
      case 'restart':
        return request('POST', `${ruta}/restarts`);
      default:
        return Promise.reject(new Error(`Acción desconocida: ${accion}`));
    }
  },
  systemdStatus: (unidad) => request('GET', `/units/${encodeURIComponent(unidad)}`),

  getHaproxy: () => request('GET', '/haproxy/backend/servers'),
  addHaproxy: (data) => request('POST', '/haproxy/backend/servers', data),
  delHaproxy: ({ direccion, puerto }) =>
    request('DELETE', `/haproxy/backend/servers/${direccion}:${puerto}`),
};
