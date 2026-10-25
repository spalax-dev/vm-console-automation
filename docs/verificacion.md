# Verificación de alcance: informe, API e interfaz

Esta matriz comprueba que cada capacidad descrita en el informe
(`docs/informe/body/03-diseno-solucion.tex`, tabla de capacidades) tiene
un endpoint en `docs/openapi.yaml` y una vista que lo consume en `web/`.

| Capacidad | API | Interfaz | Estado |
| --- | --- | --- | --- |
| Consultar sesión, conector e IP | `GET /api/session` | `Header`, `views/inicio.js` | Completo |
| Vincular una máquina | `PUT /api/session` | `components/VMSelector.js` | Completo |
| Registrar una máquina (modal) | `POST /api/session/register` | `components/RegistroModal.js` | Completo |
| Diagnosticar el registro | `GET /api/session/diagnostico` | `components/RegistroModal.js` | Completo |
| Listar y encender máquinas | `GET /api/vms`, `POST /api/vms/{vm}/poweron` | `components/VMSelector.js` | Completo |
| Listar, instalar y eliminar paquetes | `GET/POST /api/packages`, `DELETE /api/packages/{name}` | `views/paquetes.js` | Completo |
| Generar y descargar la llave de root | `POST/GET /api/keys/root` | `views/llaves.js` | Completo |
| Consultar y personalizar `/etc/issue` | `GET/PUT /api/issue` | `views/issue.js` | Completo |
| Consultar y cambiar el nombre de host | `GET/PUT /api/hostname` | `views/hostname.js` | Completo |
| Listar, crear, consultar y eliminar usuarios | `/api/users`, `/api/users/{name}` | `views/usuarios.js` | Completo |
| Descargar la llave privada de un usuario | `GET /api/users/{name}/key` | `views/usuarios.js` | Completo |
| Listar, consultar y desplegar unidades | `GET /api/units`, `/api/units/{unit}` | `views/systemd.js` | Completo |
| Iniciar, detener, habilitar, deshabilitar y reiniciar | `PATCH /api/units/{unit}`, `POST /api/units/{unit}/restarts` | `views/systemd.js` | Completo |
| Listar, agregar y eliminar servidores del back-end | `/api/haproxy/backend/servers` | `views/haproxy.js` | Completo |
| Consultar el historial con filtros y exportarlo | `GET /api/historial` | `views/historial.js` | Completo |

## Comprobaciones automáticas

```bash
task check
task openapi
```

- `task check`: formato, lint, pruebas Go, documentación generada.
- `task openapi`: valida el contrato; no queda ninguna respuesta `501`.
- `pnpm test` en `web/`: 67 pruebas del cliente y las vistas.

## Diferencias conocidas

- `DELETE /api/session/register` existe para olvidar una máquina, pero la
  interfaz no ofrece todavía un botón de «olvidar registro»; no forma
  parte de las capacidades de la tabla del informe.
- `GET /api/users/{name}` se usa para consultar una cuenta, pero la vista
  de usuarios trabaja con el listado; el endpoint está cubierto por la
  API y las pruebas.
- La autenticación de la consola no está implementada: es opcional según
  `docs/TODO.md` §9 y no se declara en el informe.
