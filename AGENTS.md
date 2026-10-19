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
Mantener la estructura de directorios: internal/handlers, internal/services, internal/utils y web/.
Los handlers solo enrutan peticiones HTTP y validan entradas; la lógica del sistema operativo va en internal/services.
La ejecución de comandos del sistema operativo se centraliza en internal/utils.
Los comentarios y la documentación del proyecto se escriben en español.
Los archivos del informe LaTeX viven en docs/informe y se compilan con latexmk, que usa LuaLaTeX por defecto.
El diagrama del escenario y las evidencias de las pruebas se guardan en docs/.
