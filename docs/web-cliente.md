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

`main.go` sirve la aplicación en la raíz del mismo origen que la API:
`/assets/` para los archivos con hash y `/` para `index.html`. La interfaz
anterior queda accesible en `/modules/` mientras se retira.

La interfaz se toma de dos fuentes, en este orden:

1. **`VMCONS_WEB`**: si la variable está definida, apunta al directorio que
   contiene `index.html` y `assets/`. Se valida al arrancar y, si falta o no
   parece una aplicación compilada, la consola termina con un mensaje claro.
2. **Copia embebida**: si la variable no está, se usa `web/dist`, que viaja
   dentro del binario con `//go:embed`. Como `web/dist` se versiona, el
   binario compila en un clon limpio sin empaquetar antes el frontend.

La instalación en el entorno del usuario usa las dos: el binario en
`~/.local/bin/vmcons` y la interfaz en `~/.local/share/vmcons/web`, de modo
que actualizar la vista no obliga a recompilar Go.

Las tipografías (IBM Plex Sans y JetBrains Mono, subset latino) y los iconos
viajan dentro del binario: la consola no depende de una red externa ni de una
CDN para verse igual.

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
| Inicio / barra superior | `GET /api/session`, `PUT /api/session`, `GET /api/vms?q=`, `POST /api/vms/{vm}/poweron` |
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

El cliente se maneja con `pnpm` dentro de `web/`; el `Taskfile.yml` solo
conserva las tareas del servidor Go.

```sh
cd web
pnpm install        # instala las dependencias
pnpm dev            # servidor de desarrollo en :5173 con proxy a la API
pnpm test           # pruebas del cliente (Vitest)
pnpm build          # compila en web/dist
```

Desde la raíz del proyecto:

```sh
task install        # compila el cliente y lo instala en ~/.local
task uninstall      # desinstala de ~/.local
```

Tras `task install`, el binario usa la copia instalada cuando se define:

```sh
export VMCONS_WEB="$HOME/.local/share/vmcons/web"
```

El proxy de desarrollo toma el destino de `VMCONS_API` (por omisión
`http://localhost:8080`), de modo que se puede apuntar a otro puerto sin tocar
el archivo de configuración. El nombre que muestra el selector de VM se toma de
`VITE_VM_NOMBRE`.

---

## 5. Pendientes conocidos

- **Persistencia de la vinculación**: la máquina elegida se guarda en memoria
  y se pierde al reiniciar el proceso, que vuelve a `VMCONS_VM`. Guardarla
  corresponde a la tabla de configuración de SQLite diferida en
  `docs/TODO.md` §9.
- **Historial**: fuera de alcance (`docs/TODO.md` §9).
- **Retiro de `web/modules` y `web/static`**: la SPA ya se sirve en la raíz;
  falta borrar la interfaz anterior, sus archivos estáticos y las entradas
  `web/index.html` del `//go:embed` cuando nadie dependa de `/modules/`.
