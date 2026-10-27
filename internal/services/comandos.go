// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

// logErrorKey es la clave de los mensajes de error en el log.
const logErrorKey = "error"

// msgCreateKeyDir es el contexto de error al crear el directorio de
// llaves.
const msgCreateKeyDir = "crear el directorio de llaves: %w"

// msgStateMachine es el contexto de error al consultar el estado.
const msgStateMachine = "consultar el estado de la máquina: %w"

// Rutas absolutas de los programas del sistema que ejecutan los servicios
// dentro de la máquina.
const (
	binCat       = "/bin/cat"
	binSh        = "/bin/sh"
	binGetent    = "/usr/bin/getent"
	binUseradd   = "/usr/sbin/useradd"
	binUsermod   = "/usr/sbin/usermod"
	binUserdel   = "/usr/sbin/userdel"
	binSystemctl = "/usr/bin/systemctl"
	binChmod     = "/bin/chmod"
	binChown     = "/bin/chown"
	binHaproxy   = "/usr/sbin/haproxy"
)
