# Arquitectura de la consola vmcons

Este documento define cómo funciona la comunicación entre la consola y la máquina virtual administrada, y la arquitectura de puertos y adaptadores que mantiene el núcleo de la aplicación libre de dependencias de VirtualBox, Guest Additions o SSH.

## Flujo general de conexión

1. La consola se ejecuta en el equipo anfitrión (Windows o GNU/Linux).
2. Para obtener la dirección IP de la máquina virtual se usa Guest Additions mediante `VBoxManage guestproperty get <vm> /VirtualBox/GuestInfo/Net/0/V4/IP`.
3. Con la IP descubierta se establece la primera conexión a la máquina virtual y se instala/configura el acceso por SSH con llaves.
4. A partir de ese momento, todas las operaciones sobre la máquina se realizan exclusivamente por SSH.

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
├── handlers/      # adaptadores primarios (HTTP): enrutan y validan
└── services/      # casos de uso; dependen solo de core/ports
```

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
