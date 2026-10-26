# vmcons

Consola web de administración de máquinas virtuales GNU/Linux desarrollada en Go para el curso de Computación en la Nube. Permite ejecutar, de manera no interactiva desde una interfaz web, tareas de administración sobre una máquina virtual Debian: instalación y eliminación de paquetes, configuración de llaves SSH, personalización de la entrada del sistema, gestión de usuarios y servicios con systemd, y alta y baja de servidores en un balanceador de carga HAProxy.

## Requisitos

- Go 1.26 o superior.
- VirtualBox 7 con `VBoxManage` en el `PATH`.
- Cliente OpenSSH (`ssh` y `scp`) en el `PATH`.
- `sshpass` para el respaldo de registro por SSH con contraseña.
- `socat` en la VM del balanceador para leer el estado real de los
  servidores del back-end de HAProxy (sin él se informa `unknown`).
### Dependencias de desarrollo

```bash
# 1. Instalar Task (una sola vez)
go install github.com/go-task/task/v3/cmd/task@latest

# 2. Instalar herramientas de Go y dependencias del frontend
task setup

# 3. Comprobar que todo está disponible
task doctor
```

`task setup` ejecuta `go mod download` y `pnpm install --frozen-lockfile`.
Las herramientas `gofumpt`, `golangci-lint` y `gomarkdoc` están fijadas en
`go.mod` con directivas `tool` y se ejecutan con `go tool`, igual que
`npx` en Node; no hace falta añadir nada al `PATH`.

Para `pnpm`, si no está instalado:

```bash
corepack enable
corepack prepare pnpm@latest --activate
```

Dependencias del sistema (Debian/Ubuntu):

```bash
sudo apt install virtualbox openssh-client sshpass
```

Opcionales para diagramas e informe: `plantuml`, `latexmk`, `texlive` y
`biber`. `npx` (Node) se usa para validar OpenAPI con Redocly.

## Configuración

La conexión con la máquina virtual se configura por variables de entorno:

| Variable | Obligatoria | Descripción |
| --- | --- | --- |
| `VMCONS_VM` | No | Nombre o UUID de la VM en VirtualBox. Si falta, se restaura la última vinculada desde SQLite. |
| `VMCONS_SSH_KEY` | No | Ruta de la llave privada del anfitrión para el conector SSH. |
| `VMCONS_GUEST_USER` | No | Usuario de Guest Additions (por omisión, `root`). |
| `VMCONS_GUEST_PASS` | No | Contraseña de Guest Additions. |
| `VMCONS_ADDR` | No | Dirección de escucha; por omisión `127.0.0.1:8080`. No hace falta definirla para `task run`. |
| `VMCONS_DB` | No | Ruta de la base SQLite (por omisión, `data/vmcons.db`). |
| `VMCONS_LOG` | No | `text` (por omisión) o `json` para los logs. |
| `VMCONS_LOG_LEVEL` | No | `debug`, `info`, `warn` o `error`. |
| `VMCONS_WEB` | No | Directorio con la interfaz web (`index.html` y `assets/`). Si falta, se usa la copia embebida en el binario. |

```bash
task run
```

Las variables son opcionales: la consola arranca en `http://localhost:8080`, restaura la última VM vinculada desde SQLite y la eliges en el selector.

En el primer arranque la consola pide crear la cuenta local con la que se
administrará (usuario de 3 a 32 caracteres y contraseña de 8 a 128). La
contraseña se guarda como hash bcrypt y la sesión es una cookie firmada de
doce horas; a partir de ahí, cada visita pide iniciar sesión.

## Instalación en el entorno del usuario

```bash
task install
```

Deja el binario en `~/.local/bin/vmcons` y la interfaz en
`~/.local/share/vmcons/web`. Después, para que el binario use esa copia de la
interfaz en lugar de la embebida:

```bash
export VMCONS_WEB="$HOME/.local/share/vmcons/web"
```

Conviene añadir esa línea al perfil (`~/.zshrc` o `~/.bashrc`). Para
desinstalar: `task uninstall`.

## Comandos

| Comando | Efecto |
| --- | --- |
| `task setup` | Instala las dependencias de desarrollo de Go y del frontend. |
| `task doctor` | Comprueba herramientas y dependencias del sistema. |
| `task run` | Ejecuta la consola sin variables de entorno; reinicia una instancia anterior y, si hay `pnpm`, recompila la interfaz. |
| `task stop` | Detiene la consola iniciada con `task run`. |
| `task test` | Formatea y ejecuta las pruebas con `-race` y cobertura. |
| `task fmt` | Aplica `gofumpt` de forma estricta. |
| `task lint` | Ejecuta `golangci-lint` con la configuración estricta de `.golangci.yml`. |
| `task build` | Compila el binario `vmcons` con la interfaz embebida. |
| `task install` | Instala el binario y la interfaz en `~/.local`. |
| `task uninstall` | Elimina la instalación de `~/.local`. |
| `task doc` | Regenera `docs/api/` desde los comentarios de documentación. |
| `task doc-check` | Verifica que `docs/api/` esté actualizada. |
| `task check` | Ciclo completo: `fmt` + `lint` + `test` + `doc-check`. |

## Documentación

- `docs/arquitectura.md` — capas, puertos y adaptadores.
- `docs/diseno-solucion.md` — contratos HTTP, validaciones y diseño de la interfaz.
- `docs/TODO.md` — seguimiento del avance del parcial.
- `docs/persistencia.md` — base SQLite, cifrado de credenciales y justificación de la dependencia.
- `docs/api/` — documentación del código generada con `gomarkdoc` (no editar a mano).
- `docs/informe/` — informe LaTeX.
