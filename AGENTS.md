## Nombres y documentación

- Los identificadores de Go —paquetes, tipos, funciones, métodos, variables, campos y constantes— se escriben en inglés.
- Los comentarios, la documentación y los mensajes al usuario se escriben en español.
- Los nombres del contrato JSON y de las rutas de la API se mantienen en español; los tipos de Go usan nombres en inglés con la etiqueta `json` correspondiente.
- Las pruebas usan nombres y datos auxiliares en inglés. No se usa el nombre de una distribución como sinónimo de máquina virtual: se dice «máquina virtual» o «distribución basada en Debian».

Trabajar siempre en este orden: docs, luego tests y por último implementación.
Antes de escribir código, escribir o actualizar primero la documentación correspondiente en docs/.
Antes de implementar una funcionalidad, escribir primero sus pruebas.
La implementación debe limitarse a hacer pasar las pruebas ya escritas.
Cada archivo .go debe iniciar con el encabezado de la licencia Apache 2.0.
El encabezado de licencia incluye la línea SPDX-License-Identifier: Apache-2.0 y el copyright del proyecto.
El encabezado de licencia se coloca como comentarios al inicio del archivo, separado por una línea en blanco del comentario del paquete.
Cada función, método, tipo y variable exportada debe tener comentario de documentación.
El comentario de documentación debe iniciar con el nombre de la función, método, tipo o variable que describe.
Describir en el comentario qué hace la función, sus parámetros y su valor de retorno cuando no sean obvios.
Formatear el código con gofumpt mediante task fmt antes de dar por terminada cualquier tarea.
Verificar la calidad del código con golangci-lint mediante task lint antes de dar por terminada cualquier tarea.
Ejecutar task test antes de considerar terminada cualquier tarea.
No commitear cambios sin haber ejecutado task lint y task test con éxito.
Usar los comandos del Taskfile.yml en lugar de invocar go, gofumpt o golangci-lint directamente.
No agregar dependencias nuevas sin justificarla en la documentación en docs/.
Mantener la estructura de directorios: internal/core, internal/adapters, internal/handlers, internal/services y web/.
internal/core/domain contiene los tipos del dominio (Command, Result, Target, MachineState, ExitError).
internal/core/ports contiene las interfaces puertos (Executor, Machine, Connector, Session); no importa de internal/adapters ni de librerías externas.
internal/adapters contiene las implementaciones concretas (localproc, vbox, ssh) de los puertos; traducen su vocabulario al dominio.
El núcleo (internal/core) nunca importa adaptadores concretos como vbox o ssh, ni conoce Guest Additions ni VirtualBox.
La inyección de dependencias de los adaptadores se realiza únicamente en main.go.
Los handlers solo enrutan peticiones HTTP y validan entradas; la lógica del sistema operativo va en internal/services y internal/adapters.
La ejecución de procesos del sistema operativo anfitrión se centraliza en el puerto Executor, implementado por internal/adapters/localproc.
Los comentarios y la documentación del proyecto se escriben en español.
Los archivos del informe LaTeX viven en docs/informe y se compilan con latexmk, que usa LuaLaTeX por defecto.
El diagrama del escenario y las evidencias de las pruebas se guardan en docs/.
