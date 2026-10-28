# Arquitectura de la consola vmcons

Este documento define cómo funciona la comunicación entre la consola y la máquina virtual administrada, y la arquitectura de puertos y adaptadores que mantiene el núcleo de la aplicación libre de dependencias de VirtualBox, Guest Additions o SSH.

## Flujo general de conexión

1. La consola se ejecuta en el equipo anfitrión (Windows o GNU/Linux).
2. Para obtener la dirección IP de la máquina virtual se enumeran las propiedades de Guest Additions (`VBoxManage guestproperty enumerate <vm>`) y se toma la primera `/VirtualBox/GuestInfo/Net/<n>/V4/IP` utilizable. Si no hay ninguna, se consulta el estado de las interfaces: cuando el cable está desconectado (`cableconnected<n>="off"`) el error lo indica en lugar de culpar a Guest Additions.
3. Con la IP descubierta se establece la primera conexión a la máquina virtual y se instala/configura el acceso por SSH con llaves.
4. A partir de ese momento, todas las operaciones sobre la máquina se realizan exclusivamente por SSH. Si una máquina registrada pierde el acceso SSH, la consola no vuelve a Guest Additions: responde `503` para no ocultar el fallo.

## Estructura del código

```
internal/
├── core/
│   ├── domain/    # Command, Result, Target, MachineState, ExitError
│   └── ports/     # Executor, Connector, Session, Machine (interfaces)
├── adapters/
│   ├── localproc/ # ports.Executor sobre os/exec (ejecución local)
│   ├── vbox/      # ports.Machine y ports.Connector sobre VBoxManage + Guest Additions
│   └── ssh/       # ports.Connector sobre el cliente OpenSSH
├── config/        # variables de entorno, rutas de datos y llaves
├── handlers/      # adaptadores primarios (HTTP): enrutan y validan
├── httpcache/     # caché de lecturas de la API con vigencia por recurso
├── services/      # casos de uso; dependen solo de core/ports
├── terminal/      # banner de arranque y handler de log para la consola
└── webui/         # resolución y montaje de la interfaz web
```

La raíz del módulo solo conserva `main.go`, que declara `//go:embed` (el
patrón no admite rutas con `..`) y compone los adaptadores, los servicios y
los manejadores; `main_test.go` prueba ese ensamblaje. El resto del código
vive en `internal/`, de modo que ninguna carpeta de la raíz acumula lógica.

Dentro de `handlers` conviven los manejadores de cada módulo y los
middlewares que envuelven el enrutador: `observabilidad.go` (identificador
de solicitud y log), `historial.go` (registro de operaciones) y
`auth.go` (exigencia de la cookie de sesión). Los dos primeros son
transversales; el tercero protege toda la API salvo `/api/auth/...`.

`main.go` es el único lugar que conoce a la vez las interfaces y sus implementaciones: ahí se instancian los adaptadores y se inyectan a los servicios.

## Puertos (interfaces definidas por el núcleo)

| Puerto      | Responsabilidad                                                        |
| ----------- | ---------------------------------------------------------------------- |
| `Executor`  | Ejecuta un programa en el sistema operativo anfitrión y captura salida, error y código de salida. |
| `Machine`   | Expone información de la máquina virtual (dirección IP, estado de encendido). |
| `Connector` | Establece sesiones con la máquina administrada (`Probe`, `Connect`, `Name`). |
| `Session`   | Sesión establecida: ejecutar órdenes (`Run`), transferir archivos (`Put`), cerrar (`Close`). |

## Adaptadores (implementaciones concretas)

| Adaptador   | Implementa         | Detalle                                                        |
| ----------- | ------------------ | -------------------------------------------------------------- |
| `localproc` | `Executor`         | Usa `os/exec`; es el único punto del código que lanza procesos del host. |
| `vbox`      | `Machine`, `Connector` | Habla con `VBoxManage`; la sesión «guest-additions» ejecuta órdenes dentro de la VM con `guestcontrol run` y transfiere archivos con `guestcontrol copyto`. |
| `ssh`       | `Connector`        | Usa el cliente OpenSSH (`ssh`/`scp`); la sesión ejecuta órdenes remotas y transfiere con `scp`. |

El conector `vbox` publica el nombre `guest-additions` y el conector `ssh` publica `ssh`. Los servicios eligen el conector sin que el núcleo conozca sus implementaciones.

## Criterio de diseño: el núcleo no conoce la infraestructura

- `core/domain` y `core/ports` no importan nada de `adapters`, `os/exec`, `net/ssh` ni librerías de VirtualBox.
- Los tipos del dominio usan la stdlib mínima (`net/netip`, `io/fs`), que es multiplataforma.
- Cada adaptador traduce su vocabulario al dominio: por ejemplo, `vbox` convierte el estado `poweroff` de VirtualBox en `domain.MachinePoweredOff`.

## Autenticación de la consola

La consola exige una cuenta local antes de administrar la máquina. El
servicio `services/Consola` (`consola.go`) guarda el hash `bcrypt` de la
contraseña en la tabla `usuarios_consola`, emite una cookie firmada con
HMAC-SHA256 y valida su vigencia; el secreto se genera en el primer uso y
se guarda en `configuracion`. `handlers/auth.go` expone las rutas de
estado, creación de la cuenta, acceso y salida, y su middleware devuelve
`401` para cualquier `/api/...` que no traiga una cookie válida. La
interfaz no monta el armazón hasta que `GET /api/auth/estado` confirma la
sesión.

## Convenciones de nombres

- Los identificadores de Go ---paquetes, tipos, funciones, métodos, variables,
  campos y constantes--- se escriben en inglés.
- Los comentarios, la documentación y los mensajes dirigidos al usuario se
  escriben en español.
- Los nombres del contrato JSON y de las rutas de la API se mantienen en
  español para no romper el cliente; los tipos que los representan usan
  nombres en inglés con la etiqueta `json` correspondiente.
- Los nombres de las pruebas y sus datos auxiliares se escriben en inglés; no
  se usa el nombre de una distribución como sinónimo de máquina virtual.

## Registro y observabilidad

Cada petición a `/api/` recibe un identificador `X-Request-ID` y se
registra con un formato compacto: hora, nivel coloreado, método, ruta,
descripción legible de la operación, estado, duración, máquina y
solicitud. En los errores 4xx y en el `503` se añade el motivo
comprensible devuelto al cliente. Los 4xx y el `503` se anotan como
advertencia; los 5xx internos, como error. El formato se
elige con `VMCONS_LOG` (`text` por omisión o `json`) y el nivel con
`VMCONS_LOG_LEVEL` (`debug`, `info`, `warn`, `error`).

Los adaptadores `vbox` y `ssh` sustituyen la orden completa por una versión
sin argumentos sensibles cuando exponen un `domain.ExitError`, de modo que
la contraseña de Guest Additions o el script de `sudo` no aparecen en los
logs ni en el campo `tecnico` de la respuesta. La respuesta de error usa la
misma envoltura que la de éxito: `mensaje` siempre y `tecnico` cuando
existe, y `operacion`, `codigo`, `salida` y `stderr` solo cuando la orden
llegó a ejecutarse; el `X-Request-ID` viaja en la cabecera para correlacionar con
el servidor.

## Documentación y calidad del código

La documentación del código se escribe en los comentarios de paquete, tipos, funciones,
métodos y variables exportadas; cada comentario inicia con el nombre del elemento que
describe. Sobre esos comentarios se genera documentación Markdown con **gomarkdoc**:

```bash
task doc     # regenera docs/api/ a partir de los comentarios del código
```

| Herramienta | Papel | Tarea |
| --- | --- | --- |
| `gofumpt` | Formateador estricto (superset de `gofmt`) | `task fmt` |
| `golangci-lint` | Linter estricto con ~40 linters y reglas de estilo | `task lint` |
| `gomarkdoc` | Genera Markdown desde los comentarios de documentación | `task doc` |

Las tres están fijadas en la directiva `tool` de `go.mod` y se ejecutan con
`go tool`, igual que `npx` en Node. `task setup` descarga las herramientas y
las dependencias de `web/`; `task doctor` comprueba que todo esté disponible.
No es necesario instalar los binarios en el `PATH`.
| Redocly CLI | Valida el contrato OpenAPI (`docs/openapi.yaml`) | `task openapi` |

La configuración vive en `.golangci.yml` (linters, límites de complejidad y longitud) y
`.gomarkdoc.yml` (formato y carpeta de salida). `task check` encadena formateo, lint,
pruebas y verificación de que la documentación generada está actualizada; es el comando
que debe pasar antes de cada commit.

Instalación de las herramientas:

```bash
go install mvdan.cc/gofumpt@latest
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
go install github.com/princjef/gomarkdoc/cmd/gomarkdoc@latest
```

## Multiplataforma (Windows y GNU/Linux)

- En GNU/Linux el binario de VirtualBox se llama `VBoxManage`; en Windows `VBoxManage.exe`. `exec.LookPath` resuelve cualquiera de los dos, por lo que el adaptador usa el nombre `VBoxManage` sin extensión.
- El cliente OpenSSH (`ssh` y `scp`) está incluido en Windows 10+ y en cualquier GNU/Linux; también se resuelve con `LookPath`.
- Las rutas de llaves privadas se manejan con `filepath`/`os`, sin asumir separadores de ruta.

## Decisiones tomadas

- **Binarios del sistema, no librerías nativas.** El SDK de VirtualBox exige COM en Windows y bindings específicos; en su lugar se usa `VBoxManage`, que es multiplataforma y oficial. Para SSH se usa el cliente del sistema en lugar de `golang.org/x/crypto/ssh`, para no cargar la gestión de llaves y configuración del cliente.
- **Errores tipados.** Las órdenes que terminan con código distinto de cero producen `domain.ExitError`, que el núcleo entiende sin importar qué adaptador la originó.
- **Validación de direcciones.** Las IPs se modelan con `net/netip.Addr`, que valida el formato y evita cadenas arbitrarias.
- **Comillas solo donde hay shell.** `guestcontrol` recibe argumentos como lista (sin shell intermedio); `ssh` sí requiere comillado del comando remoto, que realiza el adaptador antes de construir la línea.

## Próximos pasos

- Servicio de descubrimiento y selección de sesión (intentar `ssh`; si no hay llave aún, usar `guest-additions` para instalar SSH).
- Módulos funcionales sobre `Session`: APT, usuarios, `/etc/issue`, hostname, systemd y HAProxy.
