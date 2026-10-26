// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"net/http"
	"net/netip"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Reglas de validación del contrato (docs/openapi.yaml y
// docs/diseno-solucion.md §5).
var (
	// rePaquete acepta nombres de paquete APT.
	rePaquete = regexp.MustCompile(`^[a-z0-9][a-z0-9+.-]*$`)
	// reUsuario acepta nombres de usuario POSIX.
	reUsuario = regexp.MustCompile(`^[a-z_][a-z0-9_-]{0,31}$`)
	// reHostname acepta nombres de host RFC 1123.
	reHostname = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
	// reUnidad acepta unidades systemd con extensión .service.
	reUnidad = regexp.MustCompile(`^[A-Za-z0-9_.@-]+\.service$`)
	// reModo acepta permisos en octal.
	reModo = regexp.MustCompile(`^[0-7]{3,4}$`)
)

// Nombres de los campos del contrato, usados en los errores de
// validación. Los parámetros de ruta usan su nombre en inglés y los
// cuerpos JSON conservan los nombres del frontend.
const (
	campoActivo     = "activo"
	campoArchivos   = "archivos"
	campoDestino    = "destino"
	campoDireccion  = "direccion"
	campoFiles      = "files"
	campoHabilitado = "habilitado"
	campoIP         = "ip"
	campoMensaje    = "mensaje"
	campoModo       = "modo"
	campoName       = "name"
	campoNombre     = "nombre"
	campoOrigen     = "origen"
	campoPuerto     = "puerto"
	campoServer     = "server"
	campoUnit       = "unit"
)

// maxMensajeIssue es la longitud máxima del mensaje de /etc/issue.
const maxMensajeIssue = 512

// validarPaquete comprueba que el nombre sea un paquete APT válido.
func validarPaquete(campo, paquete string) error {
	if !rePaquete.MatchString(paquete) {
		return campoInvalido(campo, "debe ser un nombre de paquete APT válido (letras, dígitos, +, . y -)")
	}

	return nil
}

// validarUsuario comprueba que el nombre sea un usuario POSIX válido.
func validarUsuario(campo, nombre string) error {
	if !reUsuario.MatchString(nombre) {
		return campoInvalido(campo, "debe iniciar con minúscula o guion bajo y usar solo minúsculas, dígitos, - y _")
	}

	return nil
}

// validarHostname comprueba que el nombre cumpla RFC 1123.
func validarHostname(nombre string) error {
	if !reHostname.MatchString(nombre) {
		return campoInvalido(campoNombre, "debe ser un nombre de host RFC 1123 (minúsculas, dígitos y guiones, sin empezar ni terminar en guion)")
	}

	return nil
}

// validarUnidad comprueba que la unidad systemd tenga extensión .service.
func validarUnidad(unidad string) error {
	if !reUnidad.MatchString(unidad) {
		return campoInvalido(campoUnit, "debe ser una unidad systemd terminada en .service")
	}

	return nil
}

// validarServidor comprueba un identificador «direccion:puerto» con IPv4
// y puerto en el rango 1–65535, y devuelve el destino ya interpretado.
func validarServidor(campo, servidor string) (netip.AddrPort, error) {
	direccion, err := netip.ParseAddrPort(servidor)
	if err != nil || !direccion.Addr().Is4() || direccion.Port() == 0 {
		return netip.AddrPort{},
			campoInvalido(campo, "debe tener el formato direccion:puerto con IPv4 y puerto entre 1 y 65535")
	}

	return direccion, nil
}

// validarRuta comprueba que la ruta sea absoluta, sin traversal ni
// comodines de shell.
func validarRuta(campo, ruta string) error {
	switch {
	case ruta == "":
		return campoInvalido(campo, "no puede estar vacío")
	case !strings.HasPrefix(ruta, "/"):
		return campoInvalido(campo, "debe ser una ruta absoluta")
	case strings.Contains(ruta, ".."):
		return campoInvalido(campo, "no puede contener «..»")
	case strings.ContainsAny(ruta, "*?[]{}"):
		return campoInvalido(campo, "no puede contener comodines de shell")
	case strings.ContainsRune(ruta, 0):
		return campoInvalido(campo, "no puede contener el carácter NUL")
	default:
		return nil
	}
}

// validarMensajeIssue comprueba el límite de longitud y la ausencia de
// NUL en el mensaje de /etc/issue.
func validarMensajeIssue(mensaje string) error {
	switch {
	case utf8.RuneCountInString(mensaje) > maxMensajeIssue:
		return campoInvalido(campoMensaje, "no puede superar los 512 caracteres")
	case strings.ContainsRune(mensaje, 0):
		return campoInvalido(campoMensaje, "no puede contener el carácter NUL")
	case strings.ContainsAny(mensaje, "\r\n"):
		return campoInvalido(campoMensaje, "no puede contener saltos de línea")
	default:
		return nil
	}
}

// validarIPv4 comprueba que el valor sea una dirección IPv4 válida.
func validarIPv4(campo, valor string) error {
	direccion, err := netip.ParseAddr(valor)
	if err != nil || !direccion.Is4() {
		return campoInvalido(campo, "debe ser una dirección IPv4 válida")
	}

	return nil
}

// validarPuerto comprueba que el puerto esté en el rango 1–65535.
func validarPuerto(puerto int) error {
	if puerto < 1 || puerto > 65535 {
		return campoInvalido(campoPuerto, "debe estar entre 1 y 65535")
	}

	return nil
}

// validarModo comprueba que los permisos estén en octal.
func validarModo(campo, modo string) error {
	if !reModo.MatchString(modo) {
		return campoInvalido(campo, "debe ser un modo octal de 3 o 4 dígitos")
	}

	return nil
}

// booleanoQuery lee un parámetro booleano de la query; vacío equivale a
// false.
func booleanoQuery(r *http.Request, nombre string) (bool, error) {
	switch valor := r.URL.Query().Get(nombre); valor {
	case "", "false":
		return false, nil
	case "true":
		return true, nil
	default:
		return false, campoInvalido(nombre, "debe ser true o false")
	}
}
