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
| Servicio de descubrimiento/selección de sesión | [ ] No iniciado |
| Casos de uso (`internal/services`) | [ ] Stub (solo comentario de paquete) |
| Endpoints HTTP (`internal/handlers`) | [ ] Stub (solo comentario de paquete) |
| Ensamblaje (`main.go`: rutas + inyección de dependencias) | [ ] Stub (solo comentario de paquete) |
| Interfaz web (`web/modules`, `web/static/main.js`) | [ ] Placeholders |
| Pruebas de `services` y `handlers` | [ ] Sin cobertura (ambos paquetes: *no test files*) |
| Escenario: diagrama (numeral 5) | [ ] No iniciado |
| Pruebas manuales en video (numeral 4) | [ ] No iniciado |
| Informe (`docs/informe`) | [~] Estructura modular lista; cuerpo con TODOs |
| Resumen / index terms | [ ] Pendiente de redactar al final |

---

## 2. Precondiciones y alcance (numeral 2)

Verificar que el entorno de pruebas exista antes de ejecutar las pruebas registradas en video.

- [ ] **a)** VM Debian 13.7.0 actualizada (`apt update && apt upgrade`), SSH instalado y en ejecución, adaptador de red en modo puente — *pruebas generales*.
- [ ] **b)** VM Debian con HAPROXY instalado y configuración predeterminada — *pruebas de balanceo*.
- [ ] **c)** Disco virtual multiconexión con todo lo necesario para crear máquinas listas para el balanceo — *pruebas de despliegue*.
- [ ] **d)** Ejecutable de la aplicación de catálogo de libros + archivos requeridos — *pruebas de systemd*.
- [ ] **e)** Script que agrega fecha y hora a un archivo en cada ejecución — *pruebas de systemd*.
- [ ] Registrar IPs, nombres de VM y rutas de discos (insumo para el diagrama del escenario).

---

## 3. Infraestructura de la aplicación (bloque previo a los módulos)

Orden de trabajo obligatorio: **docs → tests → implementación**.

- [ ] **Docs:** especificar en `docs/` el descubrimiento de sesión y el comportamiento de cada módulo.
  - [ ] Servicio de descubrimiento/selección de sesión: intentar `ssh`; si aún no hay llave, usar `guest-additions` para instalar SSH (ver `docs/arquitectura.md`, «Próximos pasos»).
- [ ] **Tests:** pruebas unitarias de `internal/services` con un `Session` falso y de `internal/handlers` con `httptest`.
- [ ] **Implementación:** solo hasta que las pruebas escritas pasen.
- [ ] **`main.go`:** rutas de cada módulo, inyección de adaptadores y servido de `web/`.
- [ ] **`task fmt` + `task lint` + `task test` en verde antes de cada commit.**

---

## 4. Módulos funcionales (numeral 3)

### 4.1 Preparación del sistema

| # | Requisito | Docs | Tests | Código | UI | Video |
| --- | --- | --- | --- | --- | --- | --- |
| 3.1.1 | Instalación y eliminación de aplicaciones (`apt install` / `apt remove`) con formulario y notificación de resultado | [ ] | [ ] | [ ] | [ ] | [ ] |
| 3.1.2 | Llave SSH para `root`: generar par, instalar en `authorized_keys`, deshabilitar acceso por login/contraseña, **descarga de la llave privada** y mensaje de recomendación de uso | [ ] | [ ] | [ ] | [ ] | [ ] |
| 3.1.3 | Personalizar `/etc/issue` con nombre de host, fecha, hora, dirección IP y mensaje personalizado | [ ] | [ ] | [ ] | [ ] | [ ] |
| 3.1.4 | Asignar nombre de host y actualizar `/etc/hosts` | [ ] | [ ] | [ ] | [ ] | [ ] |

### 4.2 Gestión de usuarios

| # | Requisito | Docs | Tests | Código | UI | Video |
| --- | --- | --- | --- | --- | --- | --- |
| 3.2.1 | Crear usuario con casilla *sudo*; casilla para generar par de llaves, instalar la pública y **descargar la privada**; deshabilitar login/contraseña al habilitar SSH | [ ] | [ ] | [ ] | [ ] | [ ] |
| 3.2.2 | Eliminar usuario con casilla para eliminar también todos sus archivos | [ ] | [ ] | [ ] | [ ] | [ ] |

### 4.3 Gestión de demonios con systemd

| # | Requisito | Docs | Tests | Código | UI | Video |
| --- | --- | --- | --- | --- | --- | --- |
| 3.3 | Transferir archivos de la aplicación del catálogo de libros a la VM y gestionar permisos | [ ] | [ ] | [ ] | [ ] | [ ] |
| 3.3 | Botones `start`, `stop`, `restart`, `status`, `enable`, `disable` sobre las unidades administradas | [ ] | [ ] | [ ] | [ ] | [ ] |
| 3.3 | Prueba complementaria con el script de la precondición e) | [ ] | [ ] | [ ] | [ ] | [ ] |

### 4.4 Balanceo de carga con HAProxy

| # | Requisito | Docs | Tests | Código | UI | Video |
| --- | --- | --- | --- | --- | --- | --- |
| 3.4 | Agregar un servidor al back-end | [ ] | [ ] | [ ] | [ ] | [ ] |
| 3.4 | Eliminar un servidor del back-end | [ ] | [ ] | [ ] | [ ] | [ ] |

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

- [ ] Elaborar el diagrama con mínimo: equipo host · software de virtualización · máquinas virtuales · SO de cada VM · interfaces de red · direcciones IP · mecanismo de comunicación host ↔ VM · disco de almacenamiento · flujo general de la automatización.
- [ ] Guardar el diagrama en `docs/` y referenciarlo desde `docs/informe/body/03-diseno-solucion.tex` (descomentar `\includegraphics`).
- [ ] Redactar la descripción breve del escenario.
- [ ] Redactar el mecanismo de comunicación y el flujo general de la automatización.

---

## 7. Informe (`docs/informe`) (numeral 6)

Estructura modular lista: `main.tex` (resumen, index terms, introducción, bibliografía) + `body/` (una sección por archivo).

- [ ] `body/02-marco-teorico.tex` — completar secciones con TODO: Debian, APT, systemd, HAProxy.
- [ ] `body/03-diseno-solucion.tex` — diagrama + descripción del escenario; visión general, estructura del proyecto y decisiones tomadas.
- [ ] `body/04-implementacion.tex` — redactar 4.1 (apt, SSH root, `/etc/issue`, hostname), 4.2 (usuarios), 4.3 (systemd), 4.4 (HAProxy).
- [ ] `body/05-pruebas.tex` — diseño de las pruebas y resultados con evidencias (videos).
- [ ] `body/06-conclusiones.tex` — problemas y soluciones, herramientas de IA utilizadas, conclusiones del trabajo.
- [ ] **Resumen (abstract) y index terms** — redactar al final, 150–250 palabras.
- [ ] Revisar referencias bibliográficas en `ref.bib` y citas \parencite.
- [ ] Verificar compilación limpia con `latexmk` (LuaLaTeX): sin citas ni referencias indefinidas.
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

## Notas de seguimiento

- **24 oct 2026** — Informe reorganizado en formato modular (`main.tex` + `body/`); secciones fusionadas en *Diseño de la Solución*, *Implementación* y *Conclusiones*. Compila con `latexmk`.
- **24 oct 2026** — Se revisaron los criterios del parcial y se construyó este documento de seguimiento.
