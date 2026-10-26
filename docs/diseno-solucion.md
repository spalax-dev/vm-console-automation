# Diseño de la solución — consola `vmcons`

Este documento define el diseño funcional de la aplicación web: qué operaciones ofrece,
cómo se validan, qué flujos de proceso ejecuta y cómo se comunica con la máquina virtual.
Es el insumo de la sección *Diseño de la Solución* del informe
(`docs/informe/body/03-diseno-solucion.tex`) y el contrato que debe cumplir la
implementación de `internal/services` e `internal/handlers`.

Relación con otros documentos:

- `docs/arquitectura.md` — capas, puertos y adaptadores (cómo se ejecuta una orden).
- `docs/TODO.md` — seguimiento del avance frente a los criterios del parcial.
- `docs/diagramas/` — fuentes PlantUML y renderizados de los diagramas.

---

## 1. Vistas del diseño

| Vista | Contenido | Diagrama |
| --- | --- | --- |
| Despliegue | Escenario físico y lógico: host, hipervisor, VMs, red, IPs, disco y comunicación | `docs/diagramas/escenario.puml` (obligatorio, numeral 5) |
| Proceso | Flujo de una operación solicitada desde la interfaz, con validación y manejo de error | `docs/diagramas/flujo-procesos.puml` |
| Módulos | Contratos de los endpoints HTTP por módulo | §4 (tabla) |
| Reglas | Validación de entradas, modelo de error y seguridad | §5–§7 (tablas) |

---

## 2. Descubrimiento y selección de sesión

La consola nunca asume cómo llegó a la máquina: elige un **conector** (`ports.Connector`)
según el estado real de la VM.

| Estado de la máquina | Conector usado | Acción |
| --- | --- | --- |
| Encendida, sin llave SSH instalada | `guest-additions` | Obtener IP con `VBoxManage guestproperty get …/Net/0/V4/IP`, generar el par de llaves en el anfitrión, instalar `openssh-server`, copiar la llave pública a `authorized_keys`, detener el acceso por contraseña |
| Encendida, con llave SSH instalada | `ssh` | Sesión `ssh`/`scp` con la llave privada del anfitrión |
| Apagada o inaccesible | — | Error tipificado: el usuario debe encender la VM o revisar la red |

**Regla transversal del parcial:** la comunicación inicial es por Guest Additions y,
a partir de la instalación de SSH, **exclusivamente por SSH con llaves de autenticación**.
El conector activo se expone en `GET /api/session` para que la interfaz lo muestre.

### 2.1 Algoritmo de selección

El servicio `Discoverer` (`internal/services/sesion.go`) resuelve el conector sin que
el núcleo conozca los adaptadores:

1. Consulta el estado con `ports.Machine.State`. Si la máquina no está en ejecución,
   termina con `ErrSessionUnavailable` (HTTP 503).
2. Si el `domain.Target` no trae dirección, la descubre con `ports.Machine.Address`.
3. Recorre los conectores **en orden de preferencia** (`ssh` primero,
   `guest-additions` después) y se queda con el primero cuyo `Probe` termine sin error.
   Si la máquina ya está registrada, solo prueba SSH: Guest Additions queda
   reservado para la preparación inicial y no se usa como respaldo.
4. Si ninguno supera el sondeo, termina con `ErrSessionUnavailable`.
5. Devuelve `SessionInfo{Connector, IP, State}`. No deja sesiones abiertas: cada
   operación abre la suya a partir del conector elegido.

El orden de preferencia lo fija `main.go` al inyectar la lista, no el servicio: así se
puede probar con conectores falsos y cambiar la política sin tocar el caso de uso.

El **inventario del selector** filtra por red: solo incluye las máquinas con
la primera interfaz en modo puente u host-only. Cada entrada informa
`administrable` y, si no lo es, una `advertencia`. Las apagadas se listan para
poder encenderlas; las registradas usan SSH; y las encendidas sin registro
necesitan Guest Additions. Una encendida sin Guest Additions se muestra con el
aviso «Sin Guest Additions» y, al encenderla, la interfaz pregunta si ocultarla
o mantenerla en la lista. La preferencia se guarda en el navegador y se puede
revertir con «Mostrar». En NAT, red interna o sin adaptador el anfitrión no
alcanza la dirección y la máquina no se incluye.

La instalación de SSH cuando aún no hay llave es responsabilidad del módulo 3.1.2
(`GET/POST /api/keys/root`); el descubridor solo informa qué conector está disponible.
Mientras no exista la llave, la interfaz muestra `guest-additions` y ofrece el módulo de
llaves.

### 2.2 Servicio de paquetes APT

El servicio `Packages` (`internal/services/paquetes.go`) implementa el recurso
`/api/packages` sobre una sesión abierta por el descubridor:

| Operación | Orden remota | Respuesta |
| --- | --- | --- |
| `GET /api/packages` | `dpkg-query -W -f='${Package}\t${Version}\t${Status}\n'` | lista de paquetes con estado `installed` |
| `POST /api/packages` | `apt-get install -y <paquete>` | `201` con el resultado de la orden |
| `DELETE /api/packages/{name}` | `apt-get remove -y <paquete>` | `204` sin cuerpo |

Las órdenes se ejecutan con `DEBIAN_FRONTEND=noninteractive` para que APT no
solicite confirmaciones. El servicio abre una sesión por operación, la cierra
siempre al terminar y traduce el resultado del dominio:

- código de salida `0` → `Outcome` con `salida` y mensaje de éxito;
- código distinto de `0` → se conserva `domain.ExitError`, el `stderr` y la orden
  ejecutada para diagnóstico;
- fallo de conexión → se propaga el error de sesión (`503` en el *handler*).

El listado se filtra por el estado de `dpkg-query` (`install ok installed`);
las líneas de paquetes configurados a medias o desinstalados no se devuelven.

### 2.3 Llaves SSH de root

El recurso `/api/keys/root` gestiona el par de llaves que habilita el acceso
por SSH con autenticación de llaves:

| Operación | Acción | Respuesta |
| --- | --- | --- |
| `POST /api/keys/root` | genera el par en el anfitrión e instala la pública | `201`; `409` si la llave ya existe |
| `GET /api/keys/root` | descarga la llave privada | `200` como archivo adjunto; `404` si no existe |

El servicio `Keys` (`internal/services/llaves.go`) ejecuta esta secuencia:

1. Comprueba que la llave privada no exista en el anfitrión; si existe,
   responde `409` para no sobrescribirla en silencio.
2. Abre una sesión **antes** de generar la llave: como todavía no hay llave,
   el descubrimiento elige `guest-additions`.
3. Genera el par con `ssh-keygen -t ed25519 -N "" -f <ruta> -C vmcons-root`.
4. Si `/usr/sbin/sshd` no existe, instala `openssh-server` con APT y
   `DEBIAN_FRONTEND=noninteractive`; el canal de Guest Additions permite
   hacerlo antes de que SSH exista.
5. Instala la llave pública en `/root/.ssh/authorized_keys` con permisos
   `700` para el directorio y `600` para el archivo.
6. Escribe `/etc/ssh/sshd_config.d/99-vmcons.conf` con
   `PasswordAuthentication no` y `PermitRootLogin prohibit-password`,
   habilita y arranca el servicio con `systemctl enable --now ssh` y
   recarga la configuración con `systemctl reload ssh` (con `restart` como
   respaldo) y verifica con `sshd -T` que la autenticación por contraseña
   quedó deshabilitada.
7. Cierra la sesión. A partir de este momento el conector `ssh` queda
   disponible y pasa a ser el preferido.

La ruta de la llave se configura con `VMCONS_SSH_KEY`; si no se define, se usa
`<configuración del usuario>/vmcons/id_ed25519`. La llave privada nunca se
escribe en los logs ni en el cuerpo JSON: solo se entrega como descarga con
`Content-Disposition: attachment`. La interfaz muestra un aviso recomendando
su resguardo y el uso exclusivo por parte del administrador.

El módulo **Llaves SSH** permite elegir entre la llave de `root` —la de
conexión de la consola— y cualquier usuario regular de la máquina. Para un
usuario, `POST /api/users/{name}/key` genera el par en el anfitrión, instala
la pública en `/home/<usuario>/.ssh/authorized_keys`, deshabilita la
contraseña y devuelve `201`; `GET` descarga la privada. Una llave ya existente
responde `409` para no sobrescribirla.

### 2.4 Entrada del sistema (`/etc/issue`)

El recurso `/api/issue` lee y reemplaza el banner que `agetty` muestra antes
del inicio de sesión:

| Operación | Acción | Respuesta |
| --- | --- | --- |
| `GET /api/issue` | lee `/etc/issue` y deduce las opciones activas | `200` con `{mensaje, nombre_host, fecha, hora, ip, contenido}` |
| `PUT /api/issue` | reemplaza `/etc/issue` | `200` con el resultado de la orden |

El servicio `Issue` (`internal/services/issue.go`) escribe un formato de una
línea por elemento, usando las secuencias de `agetty`:

```
<mensaje personalizado>
Hostname: \n
Fecha: \d
Hora: \t
IP: \4
```

Solo se incluyen las líneas de las opciones activas. Al leer, el servicio
reconoce esas líneas para reconstruir las casillas y trata el resto como
mensaje personalizado; si el archivo no tiene el formato de la consola (por
ejemplo, el `/etc/issue` original de una instalación recién creada), todo el contenido se devuelve
como mensaje y las opciones quedan desactivadas.

La escritura se hace con `printf` en una sola orden POSIX y deja permisos
`644`. El mensaje se valida en el servidor: máximo 512 caracteres, sin NUL y
con saltos de línea permitidos; cada línea del mensaje se escribe tal cual y
las opciones se agregan después. La vista previa de la interfaz sustituye las
secuencias de `agetty` por el nombre, la fecha, la hora y la IP actuales, en
lugar de mostrar los códigos.

### 2.5 Nombre de host y `/etc/hosts`

El recurso `/api/hostname` administra el nombre de la máquina y su entrada en
`/etc/hosts`:

| Operación | Acción | Respuesta |
| --- | --- | --- |
| `GET /api/hostname` | lee `/etc/hostname`, la IP y `/etc/hosts` | `200` con `{nombre, ip, actualizar_hosts}` |
| `PUT /api/hostname` | cambia el nombre y actualiza `/etc/hosts` | `200` con el resultado de la orden |

El servicio `Hostname` (`internal/services/hostname.go`) obtiene el nombre de
`/etc/hostname`, la primera dirección IPv4 de `hostname -I` y comprueba si el
nombre ya aparece como destino en `/etc/hosts`. Al escribir:

1. ejecuta `hostnamectl set-hostname <nombre>`;
2. si `actualizar_hosts` es verdadero, lee `/etc/hosts`, elimina las líneas
   que mapean el nombre y agrega `<ip> <nombre>`;
3. escribe el archivo resultante con una sola orden `printf`.

El nombre se valida como RFC 1123 y la IP con `netip.Addr` antes de ejecutar
nada. Si `hostname -I` no devuelve una IPv4, la lectura termina con un error
comprensible en lugar de inventar una dirección.

### 2.6 Gestión de usuarios

El recurso `/api/users` administra las cuentas locales de la máquina:

| Operación | Acción | Respuesta |
| --- | --- | --- |
| `GET /api/users` | lista las cuentas regulares (UID ≥ 1000) | `200` con `{usuarios: [...]}` |
| `POST /api/users` | crea la cuenta, con `sudo` y llaves opcionales | `201`; `409` si ya existe |
| `GET /api/users/{name}` | consulta una cuenta | `200`; `404` si no existe |
| `DELETE /api/users/{name}` | elimina la cuenta; `files=true` borra sus archivos | `204`; `404` si no existe |
| `GET /api/users/{name}/key` | descarga la llave privada | `200` como adjunto; `404` si no existe |

El servicio `Users` (`internal/services/usuarios.go`) abre una sesión por
operación y traduce el resultado del dominio. Para crear:

1. comprueba que la cuenta no exista (`getent passwd`);
2. ejecuta `useradd -m -s /bin/bash`;
3. si `sudo` es verdadero, `usermod -aG sudo`;
4. si `llaves` es verdadero, genera el par en el anfitrión
   (`ssh-keygen`), instala la pública en
   `/home/<usuario>/.ssh/authorized_keys` con `700`/`600` y propietario del
   usuario, y bloquea el acceso por contraseña con `passwd -l`.

Las llaves de usuario se guardan en `<config>/vmcons/users/<usuario>_ed25519`;
la privada solo se entrega por `GET .../key`. La eliminación usa
`userdel -r` cuando se piden los archivos y `userdel` en caso contrario, y
borra del anfitrión las llaves generadas. El listado toma las cuentas de
`getent passwd`, la pertenencia a `sudo` de `getent group sudo` y la
existencia de la llave en el anfitrión.

### 2.7 Unidades de systemd

El recurso `/api/units` administra los servicios de la máquina:

| Operación | Acción | Respuesta |
| --- | --- | --- |
| `GET /api/units` | lista las unidades de servicio y su estado | `200` con `{unidades: [...]}` |
| `GET /api/units/{unit}` | consulta una unidad | `200`; `404` si no existe |
| `GET /api/units/{unit}/status` | devuelve la salida de `systemctl status` | `200` con el resultado de la orden |
| `PUT /api/units/{unit}` | transfiere la unidad y sus archivos | `200` con el resultado |
| `PATCH /api/units/{unit}` | cambia `activo` o `habilitado` | `200` con el estado resultante |
| `POST /api/units/{unit}/restarts` | reinicia la unidad | `202` con el resultado |

El servicio `Systemd` (`internal/services/systemd.go`) usa `systemctl` de forma
no interactiva:

- **Listado:** combina `systemctl list-units --type=service --all` (estado en
  ejecución) con `systemctl list-unit-files --type=service` (habilitación).
- **Consulta:** `systemctl show` con `LoadState`, `ActiveState`,
  `UnitFileState` y `SubState`; si `LoadState` es `not-found`, responde `404`.
- **Estado detallado:** `systemctl status --no-pager --full` devuelve la
  salida completa de la unidad para mostrarla en el panel de resultados.
  Un código 3 (unidad inactiva o fallida) se considera una consulta válida;
  solo el código 4 se traduce en `404`. La interfaz resalta las etiquetas
  (`Loaded`, `Active`, `Condition`, etc.) y los estados (`active`, `inactive`,
  `failed`, `skipped`, etc.) sin dejar de escapar el contenido.
- **Despliegue:** transfiere cada archivo con `Session.Put`, aplica el modo
  solicitado con `chmod`, ajusta propietario con `chown` si se indica y
  termina con `systemctl daemon-reload`.
- **Estado:** `PATCH` con `activo` ejecuta `start`/`stop`; con `habilitado`
  ejecuta `enable`/`disable`; el reinicio es el subrecurso `restarts`
  (`systemctl restart`).

El listado devuelve `activo`, `habilitado`, `estado` (`active`, `inactive`,
`failed`, `unknown`) y un `detalle` con el subestado (`running`, `dead`,
`exited`, etc.) para diagnóstico. Los archivos y el propietario se validan en
el *handler* antes de llegar a la máquina.

### 2.8 Registro de máquinas

Una máquina administrable tiene dos estados en `vmcons`:

| Estado | Criterio | Comportamiento |
| --- | --- | --- |
| **Sin registrar** | tiene Guest Additions y red puente u host-only, pero no hay registro | la vista de inicio muestra el botón **Registrar esta máquina**; al pulsarlo se abre el modal flotante de usuario y contraseña |
| **Registrada** | credenciales guardadas y SSH verificado | no pide login; usa la llave de root y las credenciales almacenadas |

El registro se expone en `POST /api/session/register` y sigue esta secuencia:

1. guarda el usuario y la contraseña en SQLite (`data/vmcons.db`); la
   contraseña se cifra con AES-256-GCM antes de insertarse;
2. abre Guest Additions y verifica las credenciales ejecutando `id -u`;
   si Guest Additions rechaza el acceso, comprueba primero si la llave del
   anfitrión ya abre una sesión SSH: en ese caso la máquina ya estaba
   preparada y solo falta guardar el registro. Si la llave tampoco funciona,
   reintenta por SSH con la misma contraseña (`sshpass`) antes de darse por
   vencido;
3. genera la llave de root en el anfitrión si todavía no existe;
4. instala `openssh-server` si falta, la llave pública en
   `/root/.ssh/authorized_keys`, el archivo
   `/etc/ssh/sshd_config.d/99-vmcons.conf` con
   `PasswordAuthentication no` y `PermitRootLogin prohibit-password`,
   ejecuta `systemctl enable --now ssh` y recarga la configuración con
   `systemctl reload ssh` (con `restart` como respaldo);
5. comprueba la conexión SSH real con la llave;
6. marca la máquina como registrada y responde `201`.

Si cualquier paso falla, restaura las credenciales anteriores y la máquina
queda **sin registrar**. `DELETE /api/session/register` elimina el registro de
la máquina vinculada: borra su fila de la tabla `credenciales` y la deja de
nuevo en estado **sin registrar**. En la interfaz, esa operación se ofrece
como **Olvidar registro** en el selector de máquina, detrás de una
confirmación; la llave de `root` del anfitrión se conserva porque es común a
todas las máquinas y su pérdida obligaría a regenerarla en cada registro.
`GET /api/session/diagnostico` sigue disponible para consultar el estado, el
modo de red, la versión y el nivel de Guest Additions, los usuarios con
sesión iniciada y si el puerto 22 responde, junto con problemas y
sugerencias; el modal de registro ya no lo consulta automáticamente.
`GET /api/session` y `GET /api/vms` informan el campo `registrada` para que
la interfaz decida si muestra el botón de registro. La contraseña se
conserva porque las órdenes se ejecutan de forma no interactiva; la base y
la clave quedan fuera del repositorio, con permisos `0600`. El detalle de la
persistencia y de la dependencia `modernc.org/sqlite` está en
`docs/persistencia.md`.

### 2.9 Servidores del back-end de HAProxy

El recurso `/api/haproxy/backend/servers` administra los servidores que
HAProxy usa como back-end:

| Operación | Acción | Respuesta |
| --- | --- | --- |
| `GET /api/haproxy/backend/servers` | lista los servidores del primer `backend` | `200` con `{servidores: [...]}` |
| `POST /api/haproxy/backend/servers` | agrega `direccion:puerto` | `201`; `409` si ya existe |
| `DELETE /api/haproxy/backend/servers/{server}` | elimina `direccion:puerto` | `204`; `404` si no existe |

El servicio `HAProxy` (`internal/services/haproxy.go`) lee y reescribe
`/etc/haproxy/haproxy.cfg`. Toma el **primer** bloque `backend` de la
configuración y opera sobre sus líneas `server`:

- **Alta:** agrega `server srv_<ip>_<puerto> <ip>:<puerto> check` al final del
  bloque, salvo que ya exista ese `direccion:puerto` (`409`).
- **Baja:** elimina la línea `server` cuyo destino coincide (`404` si no está).
- **Validación:** codifica la configuración propuesta en Base64 para evitar
  conflictos de entrecomillado y saltos de línea en shells multiplataforma,
  la decodifica en un archivo temporal (`/tmp/haproxy.cfg.vmcons`), ejecuta
  `haproxy -c -f` y, solo si es válida, la mueve a `/etc/haproxy/haproxy.cfg`;
  después recarga el servicio con `systemctl reload haproxy`. Un error de sintaxis
  deja intacta la configuración anterior.
- **Salto de línea final:** antes de codificar, el servicio garantiza que la
  configuración termine en `\n`. Cuando el `backend` es la última sección del
  archivo, `cat` puede devolver la última línea sin salto y la concatenación
  posterior lo perdería; HAProxy rechaza esa configuración con «Missing LF on
  last line» aunque el resto sea válido.

El identificador del servidor en la ruta es `direccion:puerto`
(por ejemplo, `192.168.1.61:8080`). El estado de cada servidor se toma del
socket de estadísticas de HAProxy: el listado ejecuta `show stat` sobre
`/run/haproxy/admin.sock` con `socat` y traduce el campo `status` de cada
fila (`UP` → `up`, `DOWN` y `MAINT` → `down`, el resto → `unknown`). Si el
socket no responde o `socat` no está instalado, la consola no falla: devuelve
los servidores con estado `unknown`, porque el listado debe seguir siendo
útil aunque la consulta de estado no esté disponible. El socket se configura
en el bloque `global` de `haproxy.cfg`; la configuración por omisión de
Debian y sus derivados suelen definir `stats socket /run/haproxy/admin.sock mode 660 level admin`.

### 2.7 Vinculación de la máquina virtual

Hasta ahora la máquina administrada se fijaba al arrancar con `VMCONS_VM`.
El recurso `/api/vms` permite buscarla y `PUT /api/session` la vincula en
caliente, sin reiniciar la consola:

| Operación | Acción | Respuesta |
| --- | --- | --- |
| `GET /api/vms?q=` | enumera las máquinas del hipervisor y marca la vinculada | `200` con `{maquinas: [...]}` |
| `PUT /api/session` | vincula la consola con `{"vm": "nombre-o-uuid"}` | `200` con la máquina vinculada; `404` si no existe |
| `POST /api/vms/{vm}/poweron` | enciende la máquina en modo sin interfaz | `200` con la máquina en `running`; `409` si ya está encendida |

El inventario se obtiene con `VBoxManage list vms` (nombre y UUID) y
`VBoxManage list runningvms` (solo los nombres en ejecución). Una máquina
que no aparece en la segunda lista se informa como `poweroff`: es una
aproximación deliberada, porque consultar el estado exacto de cada máquina
con `showvminfo` costaría una ejecución por máquina en cada búsqueda. La
pantalla de sesión sigue usando `showvminfo` para la máquina vinculada,
que es la única cuyo estado exacto se necesita.

El encendido invoca `VBoxManage startvm <nombre> --type headless`, que es
el modo que corresponde a una consola de administración: la máquina arranca
sin ventana en el anfitrión. La orden es síncrona, así que la respuesta ya
incluye el estado `running`; encender una máquina que ya corre responde
`409` en lugar de tratarse como un éxito silencioso. La consola no apaga
máquinas: un apagado desde la interfaz podría interrumpir servicios que el
usuario está administrando, de modo que esa operación se deja al
hipervisor.

La vinculación se guarda en memoria, en el mismo objeto que resuelve la
sesión, protegida con un cerrojo de lectura y escritura porque los
manejadores HTTP la consultan y la modifican en paralelo. Al reiniciar el
proceso se vuelve al valor de `VMCONS_VM`; la persistencia queda para la
tabla de configuración de SQLite que está diferida en `docs/TODO.md` §9.
Vincular una máquina apagada es válido: la selección se aplica y solo
falla la apertura de sesión con `503` hasta que se encienda. Si no hay
ninguna máquina vinculada, `GET /api/session` responde `503` con un
mensaje que invita a elegir una del inventario.

### 2.8 Caché de lecturas

Cada listado abre una sesión con la máquina y ejecuta órdenes dentro de ella:
`dpkg-query` para los paquetes, `getent` para los usuarios, `systemctl` para las
unidades. Repetir esa consulta cada vez que se abre una pestaña es caro y, en la
mayoría de los casos, innecesario: esos datos cambian solo cuando el usuario
ejecuta una operación desde la misma consola.

El servidor guarda las respuestas correctas de `GET` sobre `/api/` durante una
vigencia corta, distinta por recurso:

| Recurso | Vigencia | Motivo |
| --- | --- | --- |
| `/api/session` | 3 s | el estado de encendido puede cambiar desde el hipervisor |
| `/api/vms` | 5 s | el inventario solo cambia al crear o borrar máquinas |
| `/api/units` | 5 s | el estado de una unidad cambia al reiniciarla |
| `/api/packages`, `/api/users`, `/api/haproxy` | 15 s | cambian solo por una operación de la consola |
| `/api/issue`, `/api/hostname` | 30 s | configuración estable |

Las reglas que hacen segura la caché:

- **Solo se recuerdan respuestas `200`.** Un `503` por falta de sesión o
  cualquier error se vuelven a pedir siempre, para que el usuario vea el
  cambio en cuanto arranque la máquina o se restablezca el servicio.
- **Cualquier escritura vacía la caché completa.** Instalar un paquete, crear un
  usuario o cambiar el hostname descartan todas las entradas; así no queda una
  lista vieja después de una operación, que es el riesgo clásico de cachear.
- **Una lectura en vuelo no contamina.** Cada invalidación avanza una
  generación; la respuesta que empezó a calcularse antes del cambio de máquina
  no se guarda al terminar, porque describe el estado anterior. Sin esta regla,
  una lista que tardaba en llegar podía quedar cacheada como si fuera de la
  máquina nueva.
- **El navegador no cachea la API.** Las respuestas llevan
  `Cache-Control: no-store`: la única caché es la del servidor, que sabe cuándo
  invalidarla.
- **Solo se cachea `/api/`.** Los archivos estáticos ya llevan huella en el
  nombre y se sirven aparte.
- **Se puede forzar una lectura fresca** con `Cache-Control: no-cache`; la
  respuesta se guarda igual, de modo que la siguiente lectura la reutiliza.
- **La memoria está acotada**: como máximo 256 entradas y 1 MiB por respuesta; al
  superar el límite se descarta todo, porque las entradas duran segundos.
- Cada respuesta anuncia `X-Cache: HIT` o `X-Cache: MISS`, un encabezado de
  diagnóstico que no forma parte del contrato.

En el cliente, además, el armazón se monta una sola vez: cambiar de pestaña
sustituye solo la barra de pestañas y el contenido, sin reconstruir la barra
superior ni repetir la consulta de sesión. Al vincular otra máquina ---o
encenderla--- la vista activa se vuelve a montar, de modo que las peticiones
se repiten contra la máquina nueva y nunca se muestran datos de la anterior.

---

## 3. Flujo de proceso de una operación

Diagrama: `docs/diagramas/flujo-procesos.puml`.

1. **Entrada** — el usuario completa el formulario de un módulo y envía la petición HTTP.
2. **Validación en el cliente** — comprobaciones inmediatas (campos vacíos, formato) para
   respuesta ágil; nunca es la única validación.
3. **Validación en el servidor** — el *handler* aplica las reglas de §5 sobre los parámetros
   crudos. Si alguna falla, responde `400` con el mensaje del campo y **no ejecuta nada**.
4. **Selección de sesión** — el servicio resuelve el conector (§2). Si no hay sesión
   posible, responde `503` con la causa.
5. **Ejecución no interactiva** — el servicio construye el `domain.Command` y lo envía por
   `ports.Session.Run`; el adaptador traduce la orden al conector elegido.
6. **Clasificación del resultado**
   - código de salida `0` → operación exitosa: se construye el mensaje de éxito con la
     salida relevante;
   - código distinto de `0` → `domain.ExitError` con código, `stdout` y `stderr`;
   - fallo de conexión o tiempo de espera agotado → error de transporte.
7. **Notificación** — la respuesta indica si terminó correctamente, un mensaje comprensible
   y, cuando falla, la información técnica (orden ejecutada, código de salida, `stderr`)
   para diagnóstico. La operación queda registrada en el log del servidor con
   su descripción y, si falla, con el motivo comprensible.
8. **Cierre** — la sesión se cierra o devuelve al pool; la interfaz conserva el formulario
   con los datos ingresos.

**Responsabilidades por capa:** el *handler* solo enruta y valida; el servicio contiene el
caso de uso; el adaptador traduce al conector; el núcleo no conoce VirtualBox ni OpenSSH.

---

## 4. Contratos de la interfaz HTTP

El contrato formal y validable vive en **`docs/openapi.yaml`** (OpenAPI 3.1):
es la fuente de verdad para el frontend y para las pruebas de los *handlers*.
Esta sección resume las rutas; el documento OpenAPI detalla esquemas, ejemplos
y códigos de respuesta.

**Estado de implementación:** todas las rutas del contrato están registradas,
validan sus entradas y tienen su caso de uso conectado; los errores usan el
modelo común descrito en §6. La API incluye la sesión y el registro de la
máquina, los cuatro módulos del enunciado, el historial y la cuenta local de
la consola.

Todas las respuestas son JSON (`Content-Type: application/json`). Las peticiones
con cuerpo envían **JSON** (`Content-Type: application/json`), incluidos
`DELETE /api/usuarios` y `DELETE /api/haproxy/backend`. Las únicas respuestas que
no son JSON son las descargas de llaves, que se entregan como archivo adjunto
(`application/octet-stream` con `Content-Disposition`). Las rutas son relativas
al servidor de la consola.

### 4.1 Sesión

| Método y ruta | Parámetros | Respuesta |
| --- | --- | --- |
| `GET /api/session` | — | `{ conector, ip, estado }` |

### 4.2 Preparación del sistema

| Método y ruta | Parámetros | Validación |
| --- | --- | --- |
| `GET /api/packages` | — | lista de paquetes instalados |
| `POST /api/packages` | `nombre` (cuerpo) | nombre de paquete |
| `DELETE /api/packages/{name}` | — | nombre de paquete |
| `GET /api/keys/root` | — | descarga con `Content-Disposition` |
| `POST /api/keys/root` | — | — |
| `GET /api/issue` | — | configuración actual |
| `PUT /api/issue` | `mensaje`, `nombre_host`, `fecha`, `hora`, `ip` | sin NUL, longitud ≤ 512 |
| `GET /api/hostname` | — | configuración actual |
| `PUT /api/hostname` | `nombre`, `ip`, `actualizar_hosts` | RFC 1123, `netip.Addr` |

### 4.3 Gestión de usuarios

| Método y ruta | Parámetros | Validación |
| --- | --- | --- |
| `GET /api/users` | — | lista de usuarios |
| `POST /api/users` | `nombre`, `sudo`, `llaves` | nombre de usuario, no existente |
| `GET /api/users/{name}` | — | nombre de usuario |
| `DELETE /api/users/{name}` | `files` (query) | nombre de usuario, existente |
| `GET /api/users/{name}/key` | — | descarga de la llave privada |

### 4.4 Servicios con systemd

| Método y ruta | Parámetros | Validación |
| --- | --- | --- |
| `GET /api/units` | — | lista de unidades |
| `GET /api/units/{unit}` | — | nombre de unidad |
| `PUT /api/units/{unit}` | `archivos[]`, `usuario`, `grupo` | rutas absolutas, sin `..` |
| `PATCH /api/units/{unit}` | `activo` o `habilitado` (excluyentes) | nombre de unidad |
| `POST /api/units/{unit}/restarts` | — | nombre de unidad |

### 4.5 Balanceo con HAProxy

| Método y ruta | Parámetros | Validación |
| --- | --- | --- |
| `GET /api/haproxy/backend/servers` | — | lista actual de servidores |
| `POST /api/haproxy/backend/servers` | `direccion`, `puerto` | `netip.Addr`, puerto 1–65535 |
| `DELETE /api/haproxy/backend/servers/{server}` | — | `direccion:puerto` existente |

Los identificadores (`name`, `unit`, `server`) viajan en la ruta, no en el
cuerpo. El servidor del back-end se identifica como `direccion:puerto`
(por ejemplo, `192.168.1.61:8080`). Las acciones de systemd se expresan
como cambios de estado (`PATCH`) salvo el reinicio, que es un subrecurso
(`restarts`) por no ser idempotente.

### 4.6 Autenticación de la consola

| Método y ruta | Parámetros | Validación | Respuesta |
| --- | --- | --- | --- |
| `GET /api/auth/estado` | — | — | `200` con `{configurada, autenticado, usuario}` |
| `POST /api/auth/setup` | `usuario`, `password` | usuario y contraseña de la consola | `201` con cookie; `409` si ya hay usuarios |
| `POST /api/auth/login` | `usuario`, `password` | credenciales | `200` con cookie; `401` si no coinciden |
| `POST /api/auth/logout` | — | cookie | `204` y cookie vencida |

`GET /api/auth/estado` y las dos operaciones de acceso no exigen cookie:
son las que permiten entrar. El resto de `/api/...` responde `401` con el
mensaje «La sesión de la consola no está iniciada» cuando la cookie falta,
está vencida o la firma no coincide. La interfaz no muestra el armazón
hasta que `estado` informa `autenticado: true`.

La contraseña nunca se guarda en claro: se deriva con `bcrypt` (costo por
omisión) y solo se conserva el hash. La sesión es una cookie `HttpOnly`,
`SameSite=Lax` y `Path=/` con vigencia de doce horas; su valor es
`usuario|expira|firma`, donde la firma es un HMAC-SHA256 con un secreto
aleatorio de 32 bytes guardado en la tabla `configuracion`. Al ser sin
estado, reiniciar la consola no invalida las sesiones vigentes; cerrar
sesión vence la cookie en el navegador.

## 5. Validación de entradas

Se valida **siempre** en el servidor, con mensajes que nombran el campo y el motivo.

| Campo | Regla | Ejemplo rechazado |
| --- | --- | --- |
| Usuario | `^[a-z_][a-z0-9_-]{0,31}$` y existencia según la operación | `José`, `root` al crear |
| Nombre de host | RFC 1123: `^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$` | `mi equipo`, `-inicio` |
| Paquete APT | `^[a-z0-9][a-z0-9+.-]*$` | `nano; rm -rf /` |
| Unidad systemd | `^[A-Za-z0-9_.@-]+\.service$` | `../etc/passwd` |
| Dirección IP | `net/netip.Addr` y que sea IPv4 del back-end | `999.1.1.1`, `localhost` |
| Puerto | entero 1–65535 | `0`, `70000` |
| Ruta de archivo | absoluta, sin `..` ni comodines de shell | `../../etc/shadow` |
| Texto libre (`/etc/issue`) | longitud ≤ 512, sin NUL | cadena de 4000 caracteres |
| Usuario de la consola | `^[a-z_][a-z0-9_-]{2,31}$` | `José`, `a` |
| Contraseña de la consola | 8–128 caracteres, sin NUL | `1234` |

---

## 6. Modelo de respuesta y de error

```json
{ "ok": true,  "mensaje": "Paquete «nano» instalado.",
  "operacion": "apt-get install -y nano", "codigo": 0,
  "salida": "…", "stderr": "" }
{ "ok": false, "mensaje": "No fue posible instalar «nano».",
  "operacion": "apt-get install -y nano", "codigo": 100,
  "stderr": "E: Unable to locate package nano",
  "tecnico": "exit=100 · stderr=E: Unable to locate package nano" }
{ "ok": false, "mensaje": "El campo «nombre» no es válido.",
  "campo": "nombre", "tecnico": "no cumple ^[a-z0-9][a-z0-9+.-]*$" }
```

Todas las respuestas comparten una sola envoltura, `ResultadoOperacion`, tanto
las de éxito como las de error; `ok` y `mensaje` nunca faltan. En un error,
`campo` aparece únicamente cuando falla la validación de un dato de entrada, y
`operacion`, `codigo`, `salida` y `stderr` solo se incluyen si la orden llegó a
ejecutarse; por eso `codigo` está ausente cuando no hay código de salida, como
en la validación `400`, la falta de autenticación `401` o los errores `404`,
`409` y `503` que no ejecutaron nada. `tecnico` es opcional y resume el fallo
para el diagnóstico. El identificador `X-Request-ID` viaja en la cabecera, no
en el cuerpo, para correlacionar con el servidor.

| Situación | HTTP | Origen |
| --- | --- | --- |
| Entrada inválida | 400 | validación del *handler* |
| Recurso inexistente (usuario/unidad) | 404 | caso de uso del servicio |
| Recurso creado | 201 | `POST` con `Location` |
| Recurso reemplazado o modificado | 200 | `PUT` y `PATCH` |
| Recurso eliminado | 204 | `DELETE` |
| Sin sesión posible (VM apagada) | 503 | selección de conector |
| Orden ejecutada con código ≠ 0 | 500 | `domain.ExitError` |

El campo `mensaje` es el texto que ve el usuario; `campo` indica el dato
rechazado; `operacion`, `codigo`, `salida`, `stderr` y `tecnico` son la
información conservada para el diagnóstico, sin exponer llaves privadas.

---

## 7. Seguridad y manejo de llaves

- Los pares de llaves se generan en el anfitrión; la **llave privada nunca se escribe en
  los logs** ni viaja en el cuerpo de las respuestas JSON, solo como descarga.
- Al instalar una llave se deshabilita el acceso por contraseña
  (`PasswordAuthentication no`) y se fijan permisos `700` para `~/.ssh` y `600` para
  `authorized_keys`.
- La interfaz muestra siempre un mensaje recomendando el uso correcto de la llave privada.
- La consola no eleva privilegios por sí misma: las órdenes que requieren `root` se
  ejecutan como `root` dentro de la VM, no en el equipo anfitrión.

---

## 8. Decisiones de diseño

1. **Validar antes de ejecutar.** Ninguna operación llega a la VM con parámetros crudos:
   primero el *handler*, luego el caso de uso.
2. **Un solo flujo de proceso para los siete módulos.** Mismo ciclo
   entrada → validación → sesión → ejecución → clasificación → notificación; solo cambia
   la orden concreta, lo que uniformiza mensajes de error y pruebas.
3. **El conector es una decisión en tiempo de ejecución.** La migración Guest Additions →
   SSH no está incrustada en la lógica de negocio: se resuelve en la selección de sesión.
4. **Errores tipados, no cadenas.** `domain.ExitError` conserva código y salida para
   armar la respuesta de diagnóstico en cualquier adaptador.
5. **Interfaces descargables.** Las llaves privadas se entregan como archivo con
   `Content-Disposition`, no como texto pegado en la página.
6. **La consola administra VM, no las crea.** El enunciado no pide
   aprovisionar máquinas: los módulos son preparación del sistema,
   gestión de usuarios, systemd y HAProxy (§3.1 a §3.4). El disco
   multiconexión de la precondición (c) es *entorno de pruebas*
   preparado con VirtualBox fuera de la consola, y en el diagrama del
   numeral 5 aparece únicamente como «disco utilizado para
   almacenamiento», sin enlaces de creación.

---

## 9. Diagramas

Fuentes en `docs/diagramas/`, renderizadas con PlantUML y copiadas a
`docs/informe/images/` para su inclusión en el informe.

```bash
task diagramas
```

| Archivo | Tipo | Uso |
| --- | --- | --- |
| `escenario.puml` | despliegue | Informe §3.1 (numeral 5, obligatorio) |
| `flujo-procesos.puml` | procesos (actividades) | Informe §3.2.1 |

Criterios de estilo acordados para ambos diagramas:

- Sin sombras, sin esquinas redondeadas y sin recuadros de leyenda:
  `skinparam shadowing false` y `skinparam roundCorner 0`. Lo que antes
  decía la leyenda pasa al texto y al pie de la figura del informe.
- En el escenario, `vmcons` dibuja flechas de uso hacia VirtualBox 7 y
  OpenSSH, y el disco multiconexión se declara dentro del equipo host
  porque vive en el anfitrión: es almacenamiento, sin flechas de
  creación de VM.
- El contenedor de VM lleva un título corto («Máquinas virtuales -
  Debian 13»); los roles (pruebas, back-end) y la posibilidad de
  añadir otras quedan en §3.1.1 del informe.
- El flujo general de la automatización se dibuja como bucle 1 → 4 en
  el navegador: «1 formulario / 4 resultado»; el paso intermedio
  (validación, sesión y ejecución) ocurre entre `vmcons`, los
  adaptadores y la VM.
- El balanceador de HAProxy va en nodo propio (módulo 3.4 y punto de
  partida del reenvío); la VM de pruebas y las back-end se resumen en un
  único contenedor marcado como «o cualesquiera».
- Legibilidad: el texto debe imprimirse a ≥ 7 pt, lo que obliga a
  mantener cortas las etiquetas y el aspecto de la figura cercano a 1.
- Direcciones IP: no se fijan en el diagrama. El anfitrión y cada VM
  reciben la suya por DHCP en cada arranque, y la consola la descubre
  en tiempo de ejecución con `VBoxManage guestproperty get`; por eso
  las interfaces se rotulan «IP por DHCP».
- Un único enlace discontinuo desde `vmcons` hacia las VM, rotulado
  «gestión de VM»: indica que la consola maneja las máquinas. Los
  mecanismos concretos (Guest Additions y SSH con llaves) se explican
  en §3.1.2 del informe.

---

## 10. Interfaz de usuario (diseño en Stitch)

Proyecto Stitch: **`vmcons — consola de automatización`**. Una sola
página con pestañas por módulo; el prototipo se genera antes de escribir
`web/`.

### 10.1 Armazón común

- **Barra superior:** `vmcons` + **selector de la VM activa** (§10.3) +
  IP, conector actual (Guest Additions / SSH) y estado `● Conectado`
  con botón `⟳`.
- **Fila de pestañas** con icono y etiqueta: Inicio · Paquetes · Llaves
  SSH · `/etc/issue` · Hostname · Usuarios · systemd · HAProxy ·
  Historial. La activa lleva subrayado rojo.
- **Contenido a dos columnas:** formulario a la izquierda y, a la
  derecha, el panel de resultado: banda `✔/✖` con mensaje comprensible
  y bloque «detalle técnico» plegable con la orden ejecutada, el código
  de salida y `stderr`.
- **Estilo tipo Qt en escala de grises con acento rojo:** fondo
  `#F0F0F0`, superficies `#FFFFFF`, texto `#1F1F1F`, bordes de 1 px
  `#B0B0B0` con radio de 3–4 px, acento `#C0392B`. Botones compactos
  (≈26 px de alto) sin sombras ni degradados.
- **Iconos:** Lucide, línea de 16 px y trazo 1.5 en pestañas y
  botones; color solo en el punto de estado.
- **Animaciones sutiles permitidas:** subrayado de pestaña 120 ms,
  entrada de la banda de resultado (*fade* + 4 px) 150 ms, spinner
  durante la ejecución, borde rojo en `hover` 100 ms y `fade` en la
  vista previa de `/etc/issue`.

### 10.2 Pantallas

| # | Pestaña | Endpoint principal | Contenido |
| --- | --- | --- | --- |
| 01 | Inicio | `GET /api/session` | tarjeta de sesión (estado, conector, IP, última consulta) y cuadrícula de atajos a los módulos |
| 02 | Paquetes | `GET/POST /api/packages` | radios *instalar/eliminar* y campo de paquete con validación en línea |
| 03 | Llaves SSH | `GET/POST /api/keys/root` | estado del par, generar, descargar y aviso de uso seguro de la llave |
| 04 | `/etc/issue` | `GET/PUT /api/issue` | mensaje, casillas host/fecha/hora/IP y vista previa en vivo |
| 05 | Hostname | `GET/PUT /api/hostname` | nombre (RFC 1123) + IP y casilla de `/etc/hosts` |
| 06 | Usuarios | `GET/POST /api/users` · `GET/DELETE /api/users/{name}` | crear (sudo, llaves), eliminar (con archivos) y descarga de llaves |
| 07 | systemd | `/api/units/…` | despliegue de la unidad + 6 botones de control + insignia de estado |
| 08 | HAProxy | `/api/haproxy/backend/servers` | tabla del back-end, formulario de alta y eliminación con confirmación |
| 09 | Historial | `GET /api/historial` (diferido) | tabla de operaciones con filtros (§10.4) |

**Estados comunes:** VM apagada o inaccesible (503) como estado vacío en
la tarjeta de sesión; «ejecutando…» con botones bloqueados y spinner;
diálogo de confirmación plano para las acciones destructivas; error de
validación en línea bajo el campo.

### 10.3 Contexto de VM

- El selector de la barra superior marca **sobre qué VM se trabaja**:
  todas las pestañas operan sobre esa VM y el panel de resultado repite
  su nombre, para evitar operaciones sobre la máquina equivocada.
- La lista **se descubre en tiempo de ejecución** con
  `VBoxManage list vms`: la consola no crea ni registra máquinas
  virtuales (coherente con §8 y con §3.1.1 del informe), solo administra
  las que ya existen en el hipervisor.
- Cada operación registrada en el historial lleva el nombre de la VM a
  la que se le aplicó.
- Si la VM está registrada, el selector ofrece **Olvidar registro**:
  borra las credenciales guardadas y la marca como *sin registrar*, de modo
  que la próxima vez pida usuario y contraseña. Pide confirmación porque
  obliga a repetir el registro.

### 10.4 Historial

Tabla con `fecha y hora` · `VM` · `módulo` · `descripción` ·
`operación` · `resultado`, con filtros combinables: **búsqueda de texto,
rango de fechas, rango horario, VM y módulo**. La consola registra
automáticamente las operaciones mutantes (`POST`, `PUT`, `PATCH`,
`DELETE`) desde la capa HTTP y las guarda en la tabla `operaciones` de
SQLite. La descripción es un resumen legible construido a partir del
método, la ruta y los campos descriptivos del cuerpo (por ejemplo,
«Instalar el paquete nano»); nunca guarda contraseñas ni el cuerpo
completo. `GET /api/historial` las consulta con esos filtros y la
interfaz exporta el resultado a CSV.
