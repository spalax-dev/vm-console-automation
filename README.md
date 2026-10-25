# vmcons

Consola web de administración de máquinas virtuales GNU/Linux desarrollada en Go para el curso de Computación en la Nube. Permite ejecutar, de manera no interactiva desde una interfaz web, tareas de administración sobre una máquina virtual Debian: instalación y eliminación de paquetes, configuración de llaves SSH, personalización de la entrada del sistema, gestión de usuarios y servicios con systemd, y alta y baja de servidores en un balanceador de carga HAProxy.

## Requisitos

- Go 1.26 o superior.
- VirtualBox 7 con `VBoxManage` en el `PATH`.
- Cliente OpenSSH (`ssh` y `scp`) en el `PATH`.
- `sshpass` para el respaldo de registro por SSH con contraseña.
- Herramientas de desarrollo:

```bash
go install mvdan.cc/gofumpt@latest
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
go install github.com/princjef/gomarkdoc/cmd/gomarkdoc@latest
```

## Configuración

La conexión con la máquina virtual se configura por variables de entorno:

| Variable | Obligatoria | Descripción |
| --- | --- | --- |
| `VMCONS_VM` | Sí | Nombre o UUID de la VM en VirtualBox. Sin ella, `GET /api/session` responde `503`. |
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
