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
| Adaptadores (`localproc`, `vbox`, `ssh`) | [x] Completos — cobertura 100 % / 69.1 % / 75.0 % |
| Calidad de código (`task fmt`, `task lint`, `task test`) | [x] 0 issues, todas las pruebas en verde |
| `docs/arquitectura.md` | [x] Escrito, con persistencia, observabilidad y herramientas |
| Contrato OpenAPI (`docs/openapi.yaml`) | [x] 19 rutas, validado con Redocly, sin respuestas 501 |
| Controladores HTTP de los módulos | [x] Todos los módulos conectados a los servicios |
| Servicio de descubrimiento/selección de sesión | [x] Completo — cobertura 100 % |
| Casos de uso (`internal/services`) | [x] Preparación, usuarios, systemd y HAProxy completos |
| Endpoints HTTP (`internal/handlers`) | [x] Todos los recursos implementados, con validación y errores comunes |
| Ensamblaje (`main.go`: rutas + inyección de dependencias) | [x] Servidor, `web/` embebido, rutas, historial y observabilidad |
| Interfaz web (`web/src`, Vite) | [x] Vistas por módulo, historial, registro y diagnóstico |
| Observabilidad y logs | [x] `X-Request-ID`, logs estructurados y errores sanitizados |
| Pruebas de `services` y `handlers` | [~] `services` 89.0 %; `handlers` 92.4 %; 317 casos Go y 67 del cliente |
| Escenario: diagrama (numeral 5) | [x] Hecho en PlantUML (`docs/diagramas/`), en §3.1 del informe |
| Pruebas manuales en video (numeral 4) | [ ] Diseño listo (§5.1); falta ejecutar y grabar V01–V18 |
| Informe (`docs/informe`) | [~] §3 a §6 y resumen escritos; §5.2 pendiente de las evidencias |
| Resumen / index terms | [x] Redactado (225 palabras) |

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
| 3.1.2 | Llave SSH para `root`: generar par, instalar en `authorized_keys`, deshabilitar acceso por login/contraseña, **descarga de la llave privada** y mensaje de recomendación de uso | [x] | [x] | [x] | [x] | [ ] |
| 3.1.3 | Personalizar `/etc/issue` con nombre de host, fecha, hora, dirección IP y mensaje personalizado | [x] | [x] | [x] | [ ] | [ ] |
| 3.1.4 | Asignar nombre de host y actualizar `/etc/hosts` | [x] | [x] | [x] | [ ] | [ ] |

### 4.2 Gestión de usuarios

| # | Requisito | Docs | Tests | Código | UI | Video |
| --- | --- | --- | --- | --- | --- | --- |
| 3.2.1 | Crear usuario con casilla *sudo*; casilla para generar par de llaves, instalar la pública y **descargar la privada**; deshabilitar login/contraseña al habilitar SSH | [x] | [x] | [x] | [x] | [ ] |
| 3.2.2 | Eliminar usuario con casilla para eliminar también todos sus archivos | [x] | [x] | [x] | [ ] | [ ] |

### 4.3 Gestión de demonios con systemd

| # | Requisito | Docs | Tests | Código | UI | Video |
| --- | --- | --- | --- | --- | --- | --- |
| 3.3 | Transferir archivos de la aplicación del catálogo de libros a la VM y gestionar permisos | [x] | [x] | [x] | [ ] | [ ] |
| 3.3 | Botones `start`, `stop`, `restart`, `status`, `enable`, `disable` sobre las unidades administradas | [x] | [x] | [x] | [ ] | [ ] |
| 3.3 | Prueba complementaria con el script de la precondición e) | [x] | [x] | [~] | [ ] | [ ] |

### 4.4 Balanceo de carga con HAProxy

| # | Requisito | Docs | Tests | Código | UI | Video |
| --- | --- | --- | --- | --- | --- | --- |
| 3.4 | Agregar un servidor al back-end | [x] | [x] | [x] | [ ] | [ ] |
| 3.4 | Eliminar un servidor del back-end | [x] | [x] | [x] | [ ] | [ ] |

### 4.5 Requisitos transversales de la aplicación web

- [ ] Organización de la interfaz por módulos con formularios adecuados.
- [ ] Validación de datos **antes** de ejecutar cada operación (cliente y servidor).
- [ ] Notificación de finalización correcta de cada operación.
- [ ] En caso de error: mensaje comprensible + conservación de la información técnica útil para diagnóstico.
- [ ] Comunicación inicial con la VM por Guest Additions y, tras instalar SSH, **exclusivamente por SSH con llaves**.

---

## 5. Pruebas (numeral 4)

- [x] Diseñar una prueba por cada elemento de la solución (estado anterior → operación → estado posterior): P01–P18 en `docs/informe/body/05-pruebas.tex` §5.1.
- [ ] Ejecutar cada prueba y **grabarla en video**, dejando claro el estado anterior y el posterior (V01–V18).
- [ ] Organizar los videos (nombre, carpeta, índice) para referenciarlos desde el informe.
- [ ] Registrar capturas de pantalla de apoyo para el informe.
- [ ] Sustituir las marcas `\pendiente{por grabar}` del cuadro de evidencias (§5.2) por el resultado de cada prueba.

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
- [x] `body/04-implementacion.tex` — capa HTTP, contrato REST, paquetes, llaves SSH, `/etc/issue`, hostname, usuarios, systemd, HAProxy, historial, observabilidad e interfaz web, con las cifras de prueba y cobertura medidas el 25 oct 2026.
- [~] `body/05-pruebas.tex` — §5.1 con las 18 pruebas diseñadas (estado anterior → operación → estado posterior) y §5.2 con los resultados automáticos; falta grabar V01–V18 y sustituir las marcas `\pendiente{por grabar}` del cuadro de evidencias.
- [x] `body/06-conclusiones.tex` — problemas y soluciones, herramientas de IA y conclusiones.
- [x] **Resumen (abstract) y index terms** — 225 palabras, con las palabras clave del trabajo.
- [x] Revisar referencias bibliográficas en `ref.bib` y citas \parencite.
- [x] Verificar compilación limpia con `latexmk` (LuaLaTeX): 13 páginas, sin citas ni referencias indefinidas y sin *overfull* (25 oct 2026).
- [x] Detallar problemas encontrados durante el desarrollo y cómo se solucionaron (§6.1).
- [x] Detallar las herramientas de IA empleadas y **para qué** se usaron (§6.2).

---

## 8. Criterios de calidad y cierre

- [x] `task fmt` → sin archivos pendientes.
- [x] `task lint` → 0 issues (25 oct 2026).
- [x] `task test` → verde: 317 casos Go; cobertura `services` 89.0 %, `handlers` 92.4 %.
- [x] `pnpm test` en `web/` → 67 casos en verde.
- [x] `latexmk` → PDF de 13 páginas sin errores, sin referencias indefinidas ni *overfull*.
- [ ] Commits incrementales en español, uno por bloque funcional.
- [ ] Confirmar registro del equipo en Classroom (respuesta al mensaje del docente).
- [ ] Preparar guion/orden de la sustentación (27 oct, 6:00 pm).
- [ ] Opcional: subir la cobertura de `adapters/vbox` (69.1 %) y `adapters/ssh` (75.0 %), que bajó al añadir el encendido, el inventario y el respaldo por `sshpass`.

---

## 9. Pendientes diferidos (interfaz, historial y persistencia)

Decisiones tomadas en la fase de diseño que se implementan **después**
de los módulos del numeral 3.

- [x] **Selector global de VM** en la barra superior (diseño §10.3): la lista
  se descubre en tiempo de ejecución con `VBoxManage list vms`
  (`GET /api/vms?q=`) y la máquina se vincula con `PUT /api/session`.
- [x] **Persistir la vinculación** en SQLite (`configuracion.vm_activa`);
  al arrancar sin `VMCONS_VM` se restaura la última máquina vinculada.
- [x] **Pestaña Historial** (diseño §10.4): tabla de operaciones con
  descripción legible y filtros por búsqueda de texto, fecha, hora, VM y
  módulo → `GET /api/historial`, con exportación CSV en la interfaz.
- [x] **Persistencia en SQLite** en `data/vmcons.db`: tabla `credenciales`
  por VM. La dependencia `modernc.org/sqlite` está justificada en
  `docs/persistencia.md` (puro Go, sin CGO: no rompe el binario único).
  - [x] Tabla `operaciones` para el historial, registrada automáticamente
    por el middleware HTTP.
- [x] **Autenticación** de la consola con usuario y contraseña guardados
  en SQLite (`usuarios_consola`, hash bcrypt) y cookie firmada de doce
  horas. El enunciado no la pide: se añadió al cerrar los módulos.
- [x] **Coherencia con el informe:** la estructura, las decisiones, la
  tabla de capacidades y las rutas se actualizaron con persistencia,
  registro, diagnóstico e historial. La frase «la consola no crea ni
  registra máquinas virtuales» sigue siendo cierta: no se crea nada en
  el hipervisor.

---

## Notas de seguimiento

- **25 oct 2026** — Informe al día con el código: se corrigieron las cifras
  desactualizadas de §4 (diecinueve rutas, 317 casos Go, 67 del cliente,
  cobertura real por paquete), se quitó la afirmación de que el historial
  estaba deshabilitado y se ajustó §3 para explicar el `501` en pasado.
  Se escribieron §5.1 (diseño de 18 pruebas con estado anterior, operación
  y estado posterior), §5.2 (resultado de las pruebas automáticas y cuadro
  de evidencias), §6 (problemas y soluciones, herramientas de IA y
  conclusiones) y el resumen con las palabras clave. El PDF compila en 14
  páginas sin referencias indefinidas ni *overfull*.

- **25 oct 2026** — Armazón con altura de ventana: la barra superior y las
  pestañas quedan fijas y solo el contenido del módulo se desplaza, de
  modo que Usuarios no arrastra la página entera. El panel de resultado se
  mantiene visible con `sticky` mientras se recorre el formulario. Se
  agregó el icono `Loader2`, que faltaba en el registro y dejaba un aviso
  en la consola del navegador.

- **25 oct 2026** — Panel de resultado: las fichas se mantienen como
  tarjetas claras con mensaje, hora y detalle técnico plegable, y el área
  que las contiene pasa a gris oscuro (`#2B2B2B`) para que se lea como
  una consola. Se descartó la ficha tipo terminal completo: añadía ruido
  sin mejorar la lectura.

- **25 oct 2026** — Historial de resultados en el panel lateral: cada
  operación añade una ficha y las anteriores se conservan, en orden
  cronológico y con desplazamiento automático a la última. El contenido
  se guarda en `localStorage` (`vmcons:resultados`, máximo 50 fichas)
  para que sobreviva al cambio de módulo y a la recarga, con un botón
  **Limpiar** que vacía el historial. Si el navegador bloquea el
  almacenamiento, se usa un respaldo en memoria.
- **25 oct 2026** — Panel de resultado: la ficha más reciente se abre con
  su detalle técnico y la vista queda siempre desplazada a ella, tanto al
  añadir una operación como al montar el panel tras cambiar de módulo
  (se reintenta en el cuadro siguiente porque al montarlo todavía no hay
  medidas). Los mensajes genéricos («La operación terminó correctamente»)
  se sustituyen por la orden ejecutada cuando el servidor no devuelve
  texto, que era lo que ocurría en los borrados con `204`.

- **25 oct 2026** — Dependencias de desarrollo: `gofumpt`,
  `golangci-lint` y `gomarkdoc` quedaron fijados en la directiva `tool`
  de `go.mod` y se ejecutan con `go tool`; `task setup` descarga las
  herramientas y las dependencias de `web/`, y `task doctor` verifica el
  entorno. Ya no hace falta instalar binarios en el `PATH`.

- **25 oct 2026** — Estabilidad de VirtualBox: `VBoxManage` se ejecuta
  en serie dentro del adaptador y las consultas de solo lectura se
  reintentan una vez si el programa está ocupado. Durante el arranque,
  `GET /api/session` responde `503` (advertencia, no error) y el
  selector reintenta hasta 90 s. Los logs usan un formato compacto con
  color por nivel (`INFO`/`WARN`/`ERROR`) y `503` como advertencia.

- **25 oct 2026** — Corregido `guestcontrol run`: el adaptador pasaba la
  ruta del ejecutable dos veces (`--exe /usr/bin/id -- /usr/bin/id -u`),
  por lo que `id` interpretaba la ruta como nombre de usuario. Ahora los
  argumentos van solo después de `--`. Esto explica los fallos de
  Guest Additions con credenciales correctas.
- **25 oct 2026** — Arranque de VM: mientras Guest Additions no publica
  la IP, `GET /api/session` responde `503` (no `500`) y el selector
  reintenta cada 2 s hasta 90 s, mostrando «esperando a que arranque»,
  «esperando Guest Additions/SSH» y el resultado final. Ya no hace falta
  recargar la página.

- **25 oct 2026** — Corregido el refresco al cambiar de máquina. La caché
  no debía servir datos de la máquina anterior por dos motivos: una
  lectura que ya estaba en vuelo podía guardarse *después* de la
  invalidación, y el navegador podía reutilizar respuestas de la API. Lo
  primero se resuelve con una generación que avanza en cada
  invalidación; lo segundo con `Cache-Control: no-store`. En el cliente,
  el aviso de cambio de máquina se emite en cuanto el servidor acepta el
  vínculo, aunque el registro de credenciales quede pendiente.

- **25 oct 2026** — Observabilidad y verificación: cada petición de la
  API recibe `X-Request-ID` y se registra con método, ruta, descripción,
  estado, duración, tamaño, VM y, en los errores, el motivo;
  `VMCONS_LOG=json` y `VMCONS_LOG_LEVEL` controlan
  el formato y el nivel. Los adaptadores sanitizan las órdenes para no
  exponer contraseñas en logs ni en `tecnico`. Se añadió
  `docs/verificacion.md` con la matriz capacidad → API → interfaz.

- **25 oct 2026** — Diagnóstico de registro: `GET /api/session/diagnostico`
  informa estado, red, IP, versión/nivel de Guest Additions, usuario con
  sesión iniciada y si SSH responde. El modal lo muestra al abrirse y
  tras un fallo, con problemas y sugerencias; sugiere el usuario `jorge`
  cuando hay una sesión iniciada.

- **25 oct 2026** — Corregido el registro: el modal deshabilitaba los
  campos antes de leer `FormData`, por lo que el navegador enviaba
  `usuario` y `password` vacíos y el servidor respondía «campo usuario».
  Ahora se leen los datos antes de bloquear el formulario y el servidor
  recorta espacios del usuario.

- **25 oct 2026** — Arranque sin configuración: `task run` escucha en
  `127.0.0.1:8080`, reinicia una instancia anterior, recompila la
  interfaz si hay `pnpm` y no exige variables de entorno. `task stop`
  detiene la consola. `VMCONS_ADDR` queda como override opcional.

- **25 oct 2026** — Respaldo de registro por SSH: si Guest Additions
  rechaza las credenciales (por ejemplo, por una versión antigua de
  Guest Additions), el registro reintenta con `sshpass` sobre la misma
  dirección descubierta, instala la llave y verifica SSH. El modal
  muestra el detalle técnico del fallo para diagnosticar usuario,
  contraseña o versión de Guest Additions.
- **25 oct 2026** — Diagnóstico del entorno: VirtualBox 7.2.18 con
  Guest Additions 6.0.0 en las VM; las credenciales de `guestcontrol`
  pueden fallar aunque la contraseña sea correcta. Actualizar Guest
  Additions o usar el respaldo SSH.

- **25 oct 2026** — Vinculación persistente: la tabla `configuracion`
  guarda `vm_activa` al vincular una máquina y, si `VMCONS_VM` no está
  definida, la consola restaura esa máquina al arrancar.

- **25 oct 2026** — Caché de lecturas: paquete `internal/httpcache` que
  guarda las respuestas `200` de `GET /api/…` con vigencia por recurso
  (3 s sesión, 5 s inventario y unidades, 15 s listados, 30 s
  configuración) y vacía todo ante cualquier escritura. No se guardan
  errores; `Cache-Control: no-cache` fuerza una lectura fresca y
  `X-Cache` anuncia `HIT`/`MISS`. La memoria está acotada a 256 entradas
  y 1 MiB por respuesta. En el cliente, el armazón se monta una sola vez
  para no repetir la consulta de sesión al cambiar de pestaña.

- **25 oct 2026** — Historial de operaciones: el middleware HTTP
  registra `POST`/`PUT`/`PATCH`/`DELETE` de los módulos en la tabla
  `operaciones` de SQLite (VM activa, módulo, método+ruta, descripción,
  resultado y estado HTTP). La descripción resume la acción con el nombre
  del paquete, usuario, unidad o servidor, sin guardar contraseñas ni el
  cuerpo completo. Las bases anteriores reciben la columna `descripcion`
  automáticamente. `GET /api/historial` filtra por texto —incluida la
  descripción—, VM, módulo, rango de fechas y rango horario, con límite de
  200 por omisión. La pestaña Historial lo consulta y exporta a CSV.

- **25 oct 2026** — Credenciales en SQLite: se reemplazó
  `credenciales.json` por `data/vmcons.db` (tabla `credenciales`) y la
  contraseña se cifra con AES-256-GCM usando `data/credenciales.key`
  (`0600`). Se justificó `modernc.org/sqlite` en
  `docs/persistencia.md` por ser puro Go y no romper el binario único.
  La base y la clave están excluidas de Git. Las máquinas registradas
  con la versión JSON deben registrarse de nuevo.

- **25 oct 2026** — Registro de máquinas: una VM con Guest Additions y
  red puente/host-only aparece como «sin registrar» y, al elegirla, la
  web abre un modal flotante de usuario y contraseña. `POST
  /api/session/register` guarda las credenciales (0600), verifica `id
  -u`, prepara `openssh-server`, la llave de root y sshd, comprueba SSH
  y marca la máquina como registrada; si falla, restaura el estado. Las
  registradas no vuelven a pedir login. `GET /api/session` y
  `GET /api/vms` exponen `registrada`. El selector exige red puente u
  host-only incluso para máquinas apagadas.

- **25 oct 2026** — Inventario administrable: el selector solo lista las
  máquinas encendidas que publican Guest Additions (canal para instalar
  SSH y red puente u host-only) y las apagadas, para poder encenderlas. La web muestra el aviso y
  el `footer` del selector indica que solo aparecen máquinas con Guest
  Additions. Al generar la llave de root, si `openssh-server` no está
  instalado se instala con APT desde la sesión de Guest Additions, se
  arranca con `systemctl enable --now ssh` y se recarga sshd para aplicar
  `PasswordAuthentication no` de inmediato.

- **25 oct 2026** — Instalación y arranque: `VMCONS_WEB` permite servir
  la interfaz desde un directorio en lugar de la copia embebida; se
  valida al arrancar (`index.html` presente) y el error es explícito.
  `task install` deja el binario en `~/.local/bin/vmcons` y la interfaz
  en `~/.local/share/vmcons/web`; `task uninstall` los retira. El
  arranque imprime el nombre en arte ASCII, la URL, el origen de la
  interfaz y la máquina vinculada, con color solo si la salida es una
  terminal y respetando `NO_COLOR`.

- **25 oct 2026** — Back-end de HAProxy (3.4): el servicio `Haproxy`
  interpreta el primer bloque `backend` de `/etc/haproxy/haproxy.cfg`,
  agrega o quita líneas `server` y valida la configuración propuesta con
  `haproxy -c` antes de moverla e recargar el servicio. `POST` responde
  `201` (`409` si el servidor ya existe), `DELETE` `204` (`404` si no
  está). Cobertura: `services` 97.1 %, `handlers` 95.6 %.

- **25 oct 2026** — Unidades de systemd (3.3): el servicio `Systemd`
  combina `list-units` y `list-unit-files` para el listado, usa
  `systemctl show` para el estado (`LoadState`, `ActiveState`,
  `UnitFileState`, `SubState`) y despliega con `Session.Put` + `chmod` +
  `chown` + `daemon-reload`. `PATCH` ejecuta `start`/`stop` o
  `enable`/`disable`, y el reinicio es `POST .../restarts`.
  `GET`, `PUT` y `PATCH` responden `200`; `POST .../restarts`, `202`;
  unidad inexistente, `404`. Cobertura: `services` 97.9 %,
  `handlers` 95.8 %.

- **25 oct 2026** — Encendido de la máquina virtual: nuevo puerto
  `Power`, implementado con `VBoxManage startvm --type headless`.
  `POST /api/vms/{vm}/poweron` arranca la máquina y devuelve su estado
  ya en `running`; responde `409` si ya estaba encendida y `404` si no
  está en el inventario. La interfaz muestra el botón de encendido en el
  selector solo cuando la máquina vinculada está apagada. Apagar queda
  fuera de alcance a propósito: la consola no interrumpe servicios.

- **25 oct 2026** — Vinculación de la máquina virtual: nuevo puerto
  `Inventory`, implementado por el adaptador de VirtualBox con
  `VBoxManage list vms` y `list runningvms`; `GET /api/vms?q=` busca por
  nombre o UUID y `PUT /api/session` vincula la consola en caliente. El
  objetivo del descubridor pasa a ser mutable con un cerrojo de lectura y
  escritura, y la consola arranca sin máquina vinculada cuando falta
  `VMCONS_VM` en lugar de rechazar toda operación. En la interfaz, el
  selector de la barra superior busca mientras se escribe y muestra el
  estado de cada máquina. Cobertura de `services` 100 % y `handlers`
  97.8 %; cero incidencias de `golangci-lint`.

- **25 oct 2026** — Gestión de usuarios (3.2): el servicio `Usuarios`
  lista cuentas regulares (`UID ≥ 1000`), crea con `useradd -m -s
  /bin/bash`, agrega a `sudo` con `usermod -aG`, genera el par en el
  anfitrión e instala la pública en `authorized_keys` con `700`/`600` y
  propietario del usuario, y bloquea la contraseña con `passwd -l`.
  Elimina con `userdel` o `userdel -r` y borra las llaves del
  anfitrión. `POST` responde `201` (`409` si existe), `DELETE` `204`
  (`404` si no existe) y `GET .../key` entrega la privada. Cobertura:
  `services` 98.0 %, `handlers` 96.4 %.

- **25 oct 2026** — La SPA se sirve desde el binario: `main.go` embebe
  `web/dist` y lo publica en la raíz (`/assets/` para los archivos con
  huella y `/` para `index.html`), mientras la interfaz anterior sigue
  accesible en `/modules/`. `web/dist` pasó a versionarse para que
  `//go:embed` compile en un clon limpio y la tarea `build` depende de
  `web-build`. Las tipografías (IBM Plex Sans y JetBrains Mono, subset
  latino) se empaquetan con `@fontsource`, de modo que la consola no
  depende de una CDN. Verificado con el binario compilado: `/`,
  `/assets/*.js`, `/assets/*.woff2` y `/modules/` responden 200.

- **25 oct 2026** — Nombre de host y `/etc/hosts` (3.1.4): el servicio
  `Hostname` lee `/etc/hostname`, la primera IPv4 de `hostname -I` y la
  presencia del nombre en `/etc/hosts`. Al escribir ejecuta
  `hostnamectl set-hostname` y, si `actualizar_hosts` es verdadero,
  reescribe `/etc/hosts` sin las líneas que mapean el nombre y con
  `ip nombre` al final. `GET` y `PUT /api/hostname` responden `200`;
  nombre RFC 1123 e IPv4 se validan antes de ejecutar. Cobertura:
  `services` 100 %, `handlers` 97.8 %.

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
  Pruebas del cliente con Vitest: 42 casos (\texttt{api},
  \texttt{validacion}, \texttt{ui} y montaje de vistas en DOM).
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
