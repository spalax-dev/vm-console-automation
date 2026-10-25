# Cliente web de `vmcons`

Este documento describe la aplicación web que consume el contrato
`docs/openapi.yaml`. La interfaz es una SPA construida con Vite, JavaScript
sin framework y Tailwind CSS; el servidor Go la embebe desde `web/dist` y la
sirve en la raíz del mismo origen que la API.

---

## 1. Estructura

```
web/
├── index.html            punto de entrada de Vite
├── vite.config.js        proxy /api → :8080 en desarrollo
├── src/
│   ├── main.js           enrutador por hash y montaje del armazón
│   ├── api.js            cliente HTTP del contrato OpenAPI
│   ├── icons.js          registro de iconos Lucide
│   ├── ui.js             utilidades de presentación compartidas
│   ├── styles.css        tokens de diseño y clases base
│   ├── components/       armazón (barra superior, pestañas, resultado)
│   └── views/            una vista por módulo
├── dist/                 compilación que se embebe en el binario
└── modules/, static/     interfaz anterior, se retira al cerrar la migración
```

`main.go` embebe `web/dist`, `web/modules`, `web/static` y `web/index.html`.
Mientras convivan las dos interfaces, la raíz sigue redirigiendo a
`/modules/index.html`; el cambio a `web/dist/index.html` se hace cuando la SPA
cubra los ocho módulos.

---

## 2. Cliente de la API

`src/api.js` concentra todas las llamadas. Ninguna vista usa `fetch` de forma
directa, de modo que un cambio del contrato se resuelve en un solo archivo.

### 2.1 Convenciones del contrato

| Elemento | Regla |
| --- | --- |
| Identificadores | viajan en la ruta (`/api/packages/{name}`), no en el cuerpo |
| Opciones | viajan en la cadena de consulta (`?files=true`) |
| Cuerpo | JSON, sin campos desconocidos |
| Éxito con cuerpo | `ResultadoOperacion`: `{ok, mensaje, operacion, codigo, salida, stderr, tecnico, campo}` |
| `POST` | `201` con `Location` |
| `PUT` | `200` |
| `PATCH` | `200` con el recurso actualizado |
| `DELETE` | `204` sin cuerpo |

### 2.2 Manejo de errores

Los errores se normalizan en `ApiError`, que conserva:

- `estado`: código HTTP recibido.
- `mensaje`: texto de la envoltura `{ok, mensaje, campo, tecnico}`.
- `campo`: campo rechazado por la validación, cuando aplica.
- `tecnico`: detalle para el panel de resultado.

Las respuestas `501` se muestran como «módulo aún no conectado con la
máquina», no como error de red: el contrato ya existe y el caso de uso del
servidor todavía no está implementado.

`204` se resuelve con `null`; las descargas de llaves se resuelven con un
`Blob` y se entregan al navegador con `Content-Disposition`.

### 2.3 Correspondencia con los módulos

| Vista | Operaciones |
| --- | --- |
| Inicio / barra superior | `GET /api/session` |
| Paquetes | `GET /api/packages`, `POST /api/packages`, `DELETE /api/packages/{name}` |
| Llaves SSH | `POST /api/keys/root`, `GET /api/keys/root` |
| `/etc/issue` | `GET /api/issue`, `PUT /api/issue` |
| Hostname | `GET /api/hostname`, `PUT /api/hostname` |
| Usuarios | `GET /api/users`, `POST /api/users`, `DELETE /api/users/{name}?files=`, `GET /api/users/{name}/key` |
| systemd | `GET /api/units`, `GET /api/units/{unit}`, `PUT /api/units/{unit}`, `PATCH /api/units/{unit}`, `POST /api/units/{unit}/restarts` |
| HAProxy | `GET /api/haproxy/backend/servers`, `POST /api/haproxy/backend/servers`, `DELETE /api/haproxy/backend/servers/{server}` |

---

## 3. Estado de las vistas

Toda vista con datos remotos sigue el mismo ciclo:

1. **Carga**: se pinta la estructura y una fila «Consultando…».
2. **Vacío**: se explica que la respuesta llegó sin elementos.
3. **Error**: se muestra el mensaje de `ApiError` y se ofrece reintentar.
4. **Datos**: se pinta la tabla o el formulario con los valores devueltos.

Las operaciones de escritura envían su resultado al panel lateral de
resultado, que muestra el mensaje, la orden ejecutada, el código de salida y
`stderr`. La lista afectada se vuelve a consultar tras una operación exitosa.

### 3.1 Validación previa

El navegador valida con las mismas expresiones del contrato antes de enviar:
nombre de paquete, nombre POSIX de usuario, nombre de host RFC 1123, unidad
`*.service`, IPv4 y puerto. La validación del servidor sigue siendo la
autoridad; la del cliente solo evita un viaje de ida y vuelta.

---

## 4. Comandos

```sh
cd web
pnpm install        # instala las dependencias
pnpm dev            # servidor de desarrollo en :5173 con proxy a :8080
pnpm test           # pruebas del cliente (vitest)
pnpm build          # compila en web/dist
```

El proxy de desarrollo toma el destino de `VMCONS_API` (por omisión
`http://localhost:8080`), de modo que se puede apuntar a otro puerto sin tocar
el archivo de configuración. El nombre que muestra el selector de VM se toma de
`VITE_VM_NOMBRE`.

---

## 5. Pendientes conocidos

- **Lista de VM**: el selector de la barra superior necesita un recurso
  `GET /api/vms` que aún no existe en el contrato. Hasta entonces muestra la
  VM configurada por `VMCONS_VM` y el estado real de `GET /api/session`.
- **Historial**: fuera de alcance (`docs/TODO.md` §9).
- **Retiro de `web/modules` y `web/static`**: cuando la SPA cubra los ocho
  módulos y `main.go` sirva `web/dist/index.html` en la raíz. Como `web/dist`
  está en `.gitignore`, el `//go:embed` de esa carpeta exige compilar el
  frontend (`cd web && pnpm build`) antes de `go build`; por eso el cableado se
  hace en la tarea `build` de `Taskfile.yml` y no todavía en `main.go`.
