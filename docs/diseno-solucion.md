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
El conector activo se expone en `GET /api/sesion` para que la interfaz lo muestre.

### 2.1 Algoritmo de selección

El servicio `Descubridor` (`internal/services/sesion.go`) resuelve el conector sin que
el núcleo conozca los adaptadores:

1. Consulta el estado con `ports.Machine.State`. Si la máquina no está en ejecución,
   termina con `ErrSesionNoDisponible` (HTTP 503).
2. Si el `domain.Target` no trae dirección, la descubre con `ports.Machine.Address`.
3. Recorre los conectores **en orden de preferencia** (`ssh` primero,
   `guest-additions` después) y se queda con el primero cuyo `Probe` termine sin error.
4. Si ninguno supera el sondeo, termina con `ErrSesionNoDisponible`.
5. Devuelve `InfoSesion{Conector, IP, Estado}`. No deja sesiones abiertas: cada
   operación abre la suya a partir del conector elegido.

El orden de preferencia lo fija `main.go` al inyectar la lista, no el servicio: así se
puede probar con conectores falsos y cambiar la política sin tocar el caso de uso.

La instalación de SSH cuando aún no hay llave es responsabilidad del módulo 3.1.2
(`POST /api/sistema/ssh-root`); el descubridor solo informa qué conector está disponible.
Mientras no exista la llave, la interfaz muestra `guest-additions` y ofrece el módulo de
llaves.

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
   para diagnóstico. La operación completa queda registrada en el log del servidor.
8. **Cierre** — la sesión se cierra o devuelve al pool; la interfaz conserva el formulario
   con los datos ingresos.

**Responsabilidades por capa:** el *handler* solo enruta y valida; el servicio contiene el
caso de uso; el adaptador traduce al conector; el núcleo no conoce VirtualBox ni OpenSSH.

---

## 4. Contratos de la interfaz HTTP

Todas las respuestas son JSON. Las rutas son relativas al servidor de la consola.

### 4.1 Sesión

| Método y ruta | Parámetros | Respuesta |
| --- | --- | --- |
| `GET /api/sesion` | — | `{ conector, ip, estado }` |

### 4.2 Preparación del sistema

| Método y ruta | Parámetros | Validación |
| --- | --- | --- |
| `POST /api/sistema/paquetes` | `accion` (`install`\|`remove`), `paquete` | nombre de paquete |
| `POST /api/sistema/ssh-root` | `accion` (`generar`) | — |
| `GET /api/sistema/ssh-root/llave` | — | descarga con `Content-Disposition` |
| `GET /api/sistema/issue` | — | plantilla y opciones actuales |
| `POST /api/sistema/issue` | `mensaje`, `nombre_host`, `fecha`, `hora`, `ip` (todas booleanas menos `mensaje`) | sin saltos de línea peligrosos, longitud ≤ 512 |
| `POST /api/sistema/hostname` | `nombre`, `ip` | nombre RFC 1123, `netip.Addr` |

### 4.3 Gestión de usuarios

| Método y ruta | Parámetros | Validación |
| --- | --- | --- |
| `POST /api/usuarios` | `nombre`, `sudo` (bool), `llaves` (bool) | nombre de usuario, no existente |
| `DELETE /api/usuarios` | `nombre`, `archivos` (bool) | nombre de usuario, existente |
| `GET /api/usuarios/llave` | `nombre` | descarga de la llave privada generada |

### 4.4 Servicios con systemd

| Método y ruta | Parámetros | Validación |
| --- | --- | --- |
| `POST /api/systemd/despliegue` | `unidad`, archivos y permisos | rutas absolutas, sin `..` |
| `POST /api/systemd/{unidad}/{accion}` | `accion` ∈ `start`,`stop`,`restart`,`enable`,`disable` | nombre de unidad |
| `GET /api/systemd/{unidad}` | — | devuelve `status` |

### 4.5 Balanceo con HAProxy

| Método y ruta | Parámetros | Validación |
| --- | --- | --- |
| `GET /api/haproxy/backend` | — | lista actual de servidores |
| `POST /api/haproxy/backend` | `direccion`, `puerto` | `netip.Addr`, puerto 1–65535 |
| `DELETE /api/haproxy/backend` | `direccion`, `puerto` | debe existir en el back-end |

---

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

---

## 6. Modelo de respuesta y de error

```json
{ "ok": true,  "mensaje": "Paquete «nano» instalado.", "salida": "…" }
{ "ok": false, "mensaje": "No fue posible instalar «nano».",
  "operacion": "apt-get install -y nano",
  "tecnico": "exit=100 · stderr=E: Unable to locate package nano" }
```

| Situación | HTTP | Origen |
| --- | --- | --- |
| Entrada inválida | 400 | validación del *handler* |
| Recurso inexistente (usuario/unidad) | 404 | caso de uso del servicio |
| Sin sesión posible (VM apagada) | 503 | selección de conector |
| Orden ejecutada con código ≠ 0 | 500 | `domain.ExitError` |
| Orden ejecutada correctamente | 200 | — |

El campo `mensaje` es el texto que ve el usuario; `operacion` y `tecnico` son la
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
| 01 | Inicio | `GET /api/sesion` | tarjeta de sesión (estado, conector, IP, última consulta) y cuadrícula de atajos a los módulos |
| 02 | Paquetes | `POST /api/sistema/paquetes` | radios *instalar/eliminar* y campo de paquete con validación en línea |
| 03 | Llaves SSH | `POST /api/sistema/ssh-root` | estado del par, generar, descargar y aviso de uso seguro de la llave |
| 04 | `/etc/issue` | `GET/POST /api/sistema/issue` | mensaje, casillas host/fecha/hora/IP y vista previa en vivo |
| 05 | Hostname | `POST /api/sistema/hostname` | nombre (RFC 1123) + IP y casilla de `/etc/hosts` |
| 06 | Usuarios | `POST/DELETE /api/usuarios` | crear (sudo, llaves), eliminar (con archivos) y descarga de llaves |
| 07 | systemd | `/api/systemd/…` | despliegue de la unidad + 6 botones de control + insignia de estado |
| 08 | HAProxy | `/api/haproxy/backend` | tabla del back-end, formulario de alta y eliminación con confirmación |
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

### 10.4 Historial (diferido: `docs/TODO.md` §9)

Tabla con `fecha y hora` · `VM` · `módulo` · `operación` · `resultado` ·
`usuario`, con filtros combinables: **búsqueda de texto, rango de
fechas, rango horario, VM y módulo**. Sus datos se guardan en SQLite,
junto con los usuarios de la consola.
