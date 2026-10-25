# TO-DO — Parcial 2: Consola de automatización de máquina virtual (`vmcons`)

Documento de seguimiento del proyecto. Se actualiza conforme se avanza.

**Entrega:** 26 de octubre de 2026, 11:59 pm · **Sustentación:** a partir del 27 de octubre de 2026, 6:00 pm
**Equipo:** Jorge Acosta · Andrey Tombe

**Leyenda de estados:** `[ ]` pendiente · `[~]` en progreso · `[x]` completado · `[!]` bloqueado

---

## 1. Estado general

| Área | Estado |
| --- | --- |
| Núcleo (`internal/core/domain` + `ports`) | [x] Completo — cobertura 100 % |
| Adaptadores (`localproc`, `vbox`, `ssh`) | [x] Completo — cobertura 100 % / 89.2 % / 83.6 % |
| Calidad de código (`task fmt`, `task lint`, `task test`) | [x] 0 issues, todas las pruebas en verde |
| `docs/arquitectura.md` | [x] Escrito |
| Contrato OpenAPI (`docs/openapi.yaml`) | [x] 12 rutas, validado con Redocly |
| Controladores HTTP de los módulos | [~] Registrados y validando; responden 501 hasta conectar servicios |
| Servicio de descubrimiento/selección de sesión | [x] Completo — cobertura 100 % |
| Casos de uso (`internal/services`) | [~] Descubrimiento, paquetes APT, llaves SSH y `/etc/issue` hechos; módulos restantes pendientes |
| Endpoints HTTP (`internal/handlers`) | [~] `GET /api/session` hecho; módulos pendientes |
| Ensamblaje (`main.go`: rutas + inyección de dependencias) | [~] Servidor, `web/` embebido y todas las rutas registradas; faltan los servicios de módulos |
| Interfaz web (`web/modules`, `web/static/main.js`) | [ ] Placeholders |
| Pruebas de `services` y `handlers` | [~] `services` 100 %; `handlers` 98.4 % |
| Escenario: diagrama (numeral 5) | [x] Hecho en PlantUML (`docs/diagramas/`), en §3.1 del informe |
| Pruebas manuales en video (numeral 4) | [ ] No iniciado |
| Informe (`docs/informe`) | [~] §3 (diseño y escenario) redactado con los 2 diagramas; resto con TODOs |
| Resumen / index terms | [ ] Pendiente de redactar al final |

---

## 2. Precondiciones y alcance (numeral 2)

Verificar que el entorno de pruebas exista antes de ejecutar las pruebas registradas en video.

- [ ] **a)** VM Debian 13.7.0 actualizada (`apt update && apt upgrade`), SSH instalado y en ejecución, adaptador de red en modo puente — *pruebas generales*.
- [ ] **b)** VM Debian con HAPROXY instalado y configuración predeterminada — *pruebas de balanceo*.
- [ ] **c)** Disco virtual multiconexión con todo lo necesario para crear máquinas listas para el balanceo — *pruebas de despliegue*.
- [ ] **d)** Ejecutable de la aplicación de catálogo de libros + archivos requeridos — *pruebas de systemd*.
- [ ] **e)** Script que agrega fecha y hora a un archivo en cada ejecución — *pruebas de systemd*.
- [ ] Registrar el esquema de red (subred local y DHCP), los nombres de las VM y las rutas de los discos (insumo para el diagrama del escenario). Las direcciones IP se asignan por DHCP, así que no se fijan en el diagrama.

---

## 3. Infraestructura de la aplicación (bloque previo a los módulos)

Orden de trabajo obligatorio: **docs → tests → implementación**.

- [x] **Docs:** descubrimiento de sesión especificado en `docs/diseno-solucion.md` §2.1 y herramientas en `docs/arquitectura.md`.
  - [x] Servicio de descubrimiento/selección de sesión: sondea `ssh` y recurre a `guest-additions` cuando no hay llave.
- [x] **Tests:** pruebas unitarias de `internal/services` con `Machine` y conectores falsos y de `internal/handlers` con `httptest`.
- [x] **Implementación:** `internal/services/sesion.go` e `internal/handlers/sesion.go`.
- [~] **`main.go`:** servidor, `web/` embebido, inyección de adaptadores y `/api/session`; faltan las rutas de los módulos.
- [ ] **`task fmt` + `task lint` + `task test` en verde antes de cada commit.**

---

## 4. Módulos funcionales (numeral 3)

### 4.1 Preparación del sistema

| # | Requisito | Docs | Tests | Código | UI | Video |
| --- | --- | --- | --- | --- | --- | --- |
| 3.1.1 | Instalación y eliminación de aplicaciones (`apt install` / `apt remove`) con formulario y notificación de resultado | [x] | [x] | [x] | [ ] | [ ] |
| 3.1.2 | Llave SSH para `root`: generar par, instalar en `authorized_keys`, deshabilitar acceso por login/contraseña, **descarga de la llave privada** y mensaje de recomendación de uso | [x] | [x] | [x] | [ ] | [ ] |
| 3.1.3 | Personalizar `/etc/issue` con nombre de host, fecha, hora, dirección IP y mensaje personalizado | [x] | [x] | [x] | [ ] | [ ] |
| 3.1.4 | Asignar nombre de host y actualizar `/etc/hosts` | [x] | [x] | [~] | [ ] | [ ] |

### 4.2 Gestión de usuarios

| # | Requisito | Docs | Tests | Código | UI | Video |
| --- | --- | --- | --- | --- | --- | --- |
| 3.2.1 | Crear usuario con casilla *sudo*; casilla para generar par de llaves, instalar la pública y **descargar la privada**; deshabilitar login/contraseña al habilitar SSH | [x] | [x] | [~] | [ ] | [ ] |
| 3.2.2 | Eliminar usuario con casilla para eliminar también todos sus archivos | [x] | [x] | [~] | [ ] | [ ] |

### 4.3 Gestión de demonios con systemd

| # | Requisito | Docs | Tests | Código | UI | Video |
| --- | --- | --- | --- | --- | --- | --- |
| 3.3 | Transferir archivos de la aplicación del catálogo de libros a la VM y gestionar permisos | [x] | [x] | [~] | [ ] | [ ] |
| 3.3 | Botones `start`, `stop`, `restart`, `status`, `enable`, `disable` sobre las unidades administradas | [x] | [x] | [~] | [ ] | [ ] |
| 3.3 | Prueba complementaria con el script de la precondición e) | [x] | [x] | [~] | [ ] | [ ] |

### 4.4 Balanceo de carga con HAProxy

| # | Requisito | Docs | Tests | Código | UI | Video |
| --- | --- | --- | --- | --- | --- | --- |
| 3.4 | Agregar un servidor al back-end | [x] | [x] | [~] | [ ] | [ ] |
| 3.4 | Eliminar un servidor del back-end | [x] | [x] | [~] | [ ] | [ ] |

### 4.5 Requisitos transversales de la aplicación web

- [ ] Organización de la interfaz por módulos con formularios adecuados.
- [ ] Validación de datos **antes** de ejecutar cada operación (cliente y servidor).
- [ ] Notificación de finalización correcta de cada operación.
- [ ] En caso de error: mensaje comprensible + conservación de la información técnica útil para diagnóstico.
- [ ] Comunicación inicial con la VM por Guest Additions y, tras instalar SSH, **exclusivamente por SSH con llaves**.

---

## 5. Pruebas (numeral 4)

- [ ] Diseñar una prueba por cada elemento de la solución (estado anterior → operación → estado posterior).
- [ ] Ejecutar cada prueba y **grabarla en video**, dejando claro el estado anterior y el posterior.
- [ ] Organizar los videos (nombre, carpeta, índice) para referenciarlos desde el informe.
- [ ] Registrar capturas de pantalla de apoyo para el informe.

---

## 6. Escenario de aplicación (numeral 5)

- [x] Elaborar el diagrama con mínimo: equipo host · software de virtualización · máquinas virtuales · SO de cada VM · interfaces de red · direcciones IP · mecanismo de comunicación host ↔ VM · disco de almacenamiento · flujo general de la automatización.
- [x] Guardar el diagrama en `docs/diagramas/escenario.puml` y referenciarlo desde `docs/informe/body/03-diseno-solucion.tex` (regenerar con `task diagramas`).
- [x] Redactar la descripción breve del escenario.
- [x] Redactar el mecanismo de comunicación y el flujo general de la automatización.

---

## 7. Informe (`docs/informe`) (numeral 6)

Estructura modular lista: `main.tex` (resumen, index terms, introducción, bibliografía) + `body/` (una sección por archivo).

- [x] `body/02-marco-teorico.tex` — completar secciones con TODO: Debian, APT, systemd, HAProxy.
- [x] `body/03-diseno-solucion.tex` — diagrama + descripción del escenario; visión general, estructura del proyecto y decisiones tomadas.
- [~] `body/04-implementacion.tex` — capa HTTP, contrato REST, paquetes, llaves SSH y `/etc/issue` documentados; 4.1.4 a 4.4 siguen pendientes de conectar los casos de uso.
- [ ] `body/05-pruebas.tex` — diseño de las pruebas y resultados con evidencias (videos).
- [ ] `body/06-conclusiones.tex` — problemas y soluciones, herramientas de IA utilizadas, conclusiones del trabajo.
- [ ] **Resumen (abstract) y index terms** — redactar al final, 150–250 palabras.
- [x] Revisar referencias bibliográficas en `ref.bib` y citas \parencite.
- [x] Verificar compilación limpia con `latexmk` (LuaLaTeX): sin citas ni referencias indefinidas (24 oct 2026).
- [ ] Detallar problemas encontrados durante el desarrollo y cómo se solucionaron.
- [ ] Detallar las herramientas de IA empleadas y **para qué** se usaron.

---

## 8. Criterios de calidad y cierre

- [ ] `task fmt` → sin archivos pendientes.
- [ ] `task lint` → 0 issues.
- [ ] `task test` → verde, con cobertura de `services` y `handlers`.
- [ ] `latexmk` → PDF sin errores ni warnings de referencias.
- [ ] Commits incrementales en español, uno por bloque funcional.
- [ ] Confirmar registro del equipo en Classroom (respuesta al mensaje del docente).
- [ ] Preparar guion/orden de la sustentación (27 oct, 6:00 pm).

---

## 9. Pendientes diferidos (interfaz, historial y persistencia)

Decisiones tomadas en la fase de diseño que se implementan **después**
de los módulos del numeral 3.

- [ ] **Selector global de VM** en la barra superior (diseño §10.3): lista
  descubierta en tiempo de ejecución con `VBoxManage list vms`; las
  pestañas y los logs quedan siempre vinculados a la VM seleccionada.
  Falta el recurso `GET /api/vms` en el contrato; mientras tanto la barra
  muestra `VITE_VM_NOMBRE` y el estado real de `GET /api/session`.
- [ ] **Pestaña Historial** (diseño §10.4): tabla de operaciones con
  filtros por búsqueda de texto, fecha, hora, VM y módulo →
  `GET /api/historial`.
- [ ] **Persistencia en SQLite** en `data/vmcons.db`: tabla `operaciones`
  (historial por VM) y tabla de usuarios de la consola.
  - [ ] Justificar la dependencia en `docs/` antes de agregarla, p. ej.
    `modernc.org/sqlite` (puro Go, sin CGO: no rompe el binario único
    sin dependencias en tiempo de ejecución).
- [ ] **Autenticación** de la consola con usuario y contraseña guardados
  en SQLite. El enunciado no la pide: opcional, después de los módulos.
- [ ] **Coherencia con el informe:** si se adopta la persistencia,
  actualizar la estructura y las decisiones de §3.2 y revisar la frase
  «la consola no crea ni registra máquinas virtuales» de §3.1.1 (sigue
  siendo cierta: no se crea ni se registra nada en el hipervisor).

---

## Notas de seguimiento

- **24 oct 2026** — Entrada del sistema (3.1.3): el servicio `Issue`
  escribe `/etc/issue` con una línea por elemento y las secuencias de
  agetty (`\n`, `\d`, `\t`, `\4`) precedidas por etiquetas legibles;
  al leer reconoce las etiquetas y reconstruye las casillas. Un banner
  ajeno se devuelve completo como mensaje. `GET /api/issue` responde la
  configuración y el contenido; `PUT` valida el mensaje (≤ 512, sin NUL
  ni saltos de línea) y exige al menos un elemento. Cobertura:
  `services` 100 %, `handlers` 98.4 %.

- **24 oct 2026** — Cliente web conectado al contrato OpenAPI: se
  reescribió `web/src/api.js` contra `docs/openapi.yaml` (rutas en
  inglés, identificadores en la ruta, `204` sin cuerpo, descargas como
  `Blob`) con `ApiError` normalizado y detección de `501`. Se agregó
  `web/src/validacion.js` con las expresiones del contrato y
  `web/src/ui.js` con el escapado, la traducción de estados y el flujo
  de ejecución. Las ocho vistas consultan su recurso al montarse y
  muestran carga, vacío y error; las operaciones de escritura alimentan
  el panel de resultado. Se documentó en `docs/web-cliente.md`.
  Pruebas del cliente con Vitest: 36 casos (`api`, `validacion`, `ui`).
  Presupuesto: 64 kB de JavaScript (antes 528 kB) al importar solo los
  iconos utilizados.

- **24 oct 2026** — Llaves SSH de root (3.1.2): el servicio `Llaves`
  abre la sesión por Guest Additions antes de generar la llave, ejecuta
  `ssh-keygen -t ed25519` en el anfitrión, instala la pública en
  `authorized_keys` (700/600) y escribe
  `/etc/ssh/sshd_config.d/99-vmcons.conf` con
  `PasswordAuthentication no` y `PermitRootLogin prohibit-password`.
  `POST /api/keys/root` responde `201` (`409` si la llave existe) y
  `GET` entrega la privada como adjunto (`404` si no existe). La ruta
  se configura con `VMCONS_SSH_KEY` y, por omisión, es
  `<config>/vmcons/id_ed25519`. Cobertura: `services` 100 %,
  `handlers` 99.0 %.

- **24 oct 2026** — Primer módulo conectado: paquetes APT. El servicio
  `Paquetes` abre una sesión por operación, ejecuta
  `apt-get install/remove -y` con `DEBIAN_FRONTEND=noninteractive`,
  lista con `dpkg-query` y traduce `domain.ExitError` a la respuesta
  `500` con código y `stderr`. `POST /api/packages` responde `201` con
  `Location`, `DELETE` responde `204` y el listado devuelve solo
  paquetes instalados. El adaptador `vbox` ahora pasa `cmd.Env` con
  `--putenv`. Cobertura: `services` 100 %, `handlers` 96.0 %.

- **24 oct 2026** — Informe actualizado con la API REST: nueva
  subsección «Contrato REST de la API» en §3.2 con la tabla de recursos y
  los códigos de estado, y párrafo de estado en §4 (manejadores,
  validación y `501` mientras se conectan los servicios). La tabla usa
  `tabularx` y se eliminó el único *overfull*; el PDF compila en 6
  páginas sin referencias indefinidas.

- **24 oct 2026** — API replanteada como REST: recursos en inglés
  (`/api/packages`, `/api/keys/root`, `/api/users`, `/api/units`,
  `/api/haproxy/backend/servers`), identificadores en la ruta, verbos
  estándar (`GET`, `POST`, `PUT`, `PATCH`, `DELETE`) y códigos
  `201/204`. Desaparecen los endpoints con `{accion}` en el cuerpo.
  Contrato actualizado en `docs/openapi.yaml` (14 rutas) y validado con
  Redocly; handlers y pruebas ajustados.

- **24 oct 2026** — Contrato y controladores HTTP: se revisó el diseño
  en Stitch (`docs/stitch/DESIGN.md`), la SPA de `web/` y
  `docs/diseno-solucion.md`. Se escribió `docs/openapi.yaml` (OpenAPI
  3.1, 12 rutas, 23 esquemas) que unifica la envoltura
  `{ok, mensaje, operacion, codigo, salida, stderr, tecnico, campo}` y
  valida con Redocly (`task openapi`). Los controladores de sistema,
  usuarios, systemd y HAProxy quedaron registrados en `main.go`, con
  validación completa según §5 y respuesta `501` mientras no se
  conecten los casos de uso. Cobertura de `handlers`: 96.8 %.

- **24 oct 2026** — Descubrimiento de sesión implementado con el ciclo
  docs → tests → implementación. `Descubridor` verifica el estado de la
  VM, descubre la dirección si falta y sondea los conectores en orden
  (`ssh`, `guest-additions`); expone `GET /api/session` con
  `{conector, ip, estado}` y responde 503 con `ErrSesionNoDisponible`.
  `main.go` arma el servidor, embebe `web/` y lee la configuración de
  variables de entorno. Calidad: `services` 100 % y `handlers` 94.7 % de
  cobertura, `golangci-lint` con 0 issues.
- **24 oct 2026** — Herramientas de documentación y calidad: se agregó
  `gomarkdoc` (`.gomarkdoc.yml`, `task doc` y `task doc-check`) para
  generar `docs/api/` desde los comentarios del código, y `task check`
  encadena `fmt` + `lint` + `test` + `doc-check` antes de cada commit.

- **24 oct 2026** — Diseño de la interfaz (diseño §10): una sola página
  con 9 pestañas (las 8 vistas por módulo + Historial), armazón Qt en
  escala de grises con acento rojo, iconos Lucide y animaciones sutiles;
  selector de VM en la barra superior con lista descubierta por
  `VBoxManage list vms`. Historial con filtros por texto, fecha y hora y
  persistencia en SQLite (operaciones + usuarios de la consola)
  registrados como pendientes diferidos en §9.

- **24 oct 2026** — Marco teórico (§2) redactado con criterio de profundidad: se desarrollan los puntos que el lector necesita entender (descubrimiento de la IP con Guest Additions antes de SSH, autenticación por llaves no interactiva, unidades y ciclo de vida de systemd, front-end/back-end de HAProxy) y se dejan breves los que son de dominio común; Debian se fusionó con APT en una única subsección y se retiraron las secciones de Go y de arquitectura de puertos y adaptadores por no ser lo esencial (el patrón se cita ahora en 3.2.2, donde se aplica). Revisión de fuentes: `ref.bib` con 9 entradas, todas citadas y citadas todas, sin la plantilla de ejemplo; las 8 URL devuelven 200; `plantuml -checkonly` OK en los 2 diagramas; informe de 6 páginas con 0 *overfull*, 0 referencias indefinidas y las dos figuras dentro de §3.
- **24 oct 2026** — Escenario (numeral 5) y §3 del informe: diagramas PlantUML (`docs/diagramas/escenario.puml` y `flujo-procesos.puml`, tarea `task diagramas`), redacción completa de 3.1 y 3.2, y verificación de que las dos figuras caen dentro de la sección III (figuras en las páginas 3 y 4; §III ocupa las páginas 2 a 5). Legibilidad: ~7 pt impreso.
- **24 oct 2026** — Alcance confirmado contra el enunciado: **la consola no crea máquinas virtuales** (el enunciado no lo pide; el disco multiconexión es precondición (c), entorno de pruebas). El diagrama conserva el disco como «almacenamiento · imagen base» sin la flecha «5 despliegue», el contenedor pasó a llamarse «Máquinas virtuales - Debian 13» y el flujo se dibuja como bucle 1 → 4 (formulario/resultado) en el navegador.
- **24 oct 2026** — Estilo de los diagramas: sin sombras ni esquinas redondeadas, sin recuadros de leyenda (su contenido pasó al texto y al pie de la figura). El balanceador de HAProxy quedó en nodo propio; `vmcons` dibuja flechas de uso hacia VirtualBox 7 y OpenSSH y un único enlace discontinuo hacia las VM rotulado «gestión de VM» (sin letras A/B). Las direcciones IP ya no se fijan en el diagrama ni en el texto: las interfaces se rotulan «IP por DHCP» y la consola descubre la dirección en tiempo de ejecución. Aprovechado para corregir las etiquetas de `cleveref`, que salían en inglés («fig. 1», «section»).
- **24 oct 2026** — Informe reorganizado en formato modular (`main.tex` + `body/`); secciones fusionadas en *Diseño de la Solución*, *Implementación* y *Conclusiones*. Compila con `latexmk`.
- **24 oct 2026** — Se revisaron los criterios del parcial y se construyó este documento de seguimiento.
