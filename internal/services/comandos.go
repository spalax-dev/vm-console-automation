// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

// Rutas absolutas de los programas del sistema que ejecutan los
// servicios dentro de la máquina.
const (
	binCat     = "/bin/cat"
	binSh      = "/bin/sh"
	binGetent  = "/usr/bin/getent"
	binUseradd = "/usr/sbin/useradd"
	binUsermod = "/usr/sbin/usermod"
	binUserdel = "/usr/sbin/userdel"
	// #nosec G101 -- es la ruta del programa passwd, no una credencial.
	binPasswd    = "/usr/bin/passwd"
	binSystemctl = "/usr/bin/systemctl"
	binChmod     = "/bin/chmod"
	binChown     = "/bin/chown"
	binHaproxy   = "/usr/sbin/haproxy"
)
