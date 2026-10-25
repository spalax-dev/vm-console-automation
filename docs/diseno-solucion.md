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
- Un único enlace de comunicación con las VM, rotulado **A / B**; la
  definición de A (Guest Additions) y B (SSH con llaves) vive en el pie
  de la Figura 1 y en §3.1.2.
- En el escenario, `vmcons` dibuja flechas de uso hacia VirtualBox 7 y
  OpenSSH, y el disco multiconexión se declara dentro del equipo host
  porque vive en el anfitrión.
- El balanceador de HAProxy va en nodo propio (módulo 3.4 y punto de
  partida del reenvío); la VM de pruebas y las back-end se resumen en un
  único contenedor marcado como «o cualesquiera».
- Legibilidad: el texto debe imprimirse a ≥ 7 pt, lo que obliga a
  mantener cortas las etiquetas y el aspecto de la figura cercano a 1.
