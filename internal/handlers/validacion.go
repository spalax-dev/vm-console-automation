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
	rePkg = regexp.MustCompile(`^[a-z0-9][a-z0-9+.-]*$`)
	// reUsuario acepta nombres de usuario POSIX.
	reUser = regexp.MustCompile(`^[a-z_][a-z0-9_-]{0,31}$`)
	// reHostname acepta nombres de host RFC 1123.
	reHostname = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
	// reUnidad acepta unidades systemd con extensión .service.
	reUnit = regexp.MustCompile(`^[A-Za-z0-9_.@-]+\.service$`)
	// reModo acepta permisos en octal.
	reMode = regexp.MustCompile(`^[0-7]{3,4}$`)
)

// Nombres de los campos del contrato, usados en los errores de
// validación. Los parámetros de ruta usan su nombre en inglés y los
// cuerpos JSON conservan los nombres del frontend.
const (
	activeField      = "activo"
	filesField       = "archivos"
	destinationField = "destino"
	addressField     = "direccion"
	filesParam       = "files"
	enabledField     = "habilitado"
	ipField          = "ip"
	messageField     = "mensaje"
	modeField        = "modo"
	nameParam        = "name"
	nameField        = "nombre"
	sourceField      = "origen"
	portField        = "puerto"
	serverParam      = "server"
	unitParam        = "unit"
)

// maxMessageIssue es la longitud máxima del mensaje de /etc/issue.
const maxMessageIssue = 512

// validatePkg comprueba que el nombre sea un paquete APT válido.
func validatePkg(field, pkg string) error {
	if !rePkg.MatchString(pkg) {
		return invalidField(field, "debe ser un nombre de paquete APT válido (letras, dígitos, +, . y -)")
	}

	return nil
}

// validateUser comprueba que el nombre sea un usuario POSIX válido.
func validateUser(field, name string) error {
	if !reUser.MatchString(name) {
		return invalidField(field, "debe iniciar con minúscula o guion bajo y usar solo minúsculas, dígitos, - y _")
	}

	return nil
}

// validateHostname comprueba que el nombre cumpla RFC 1123.
func validateHostname(name string) error {
	if !reHostname.MatchString(name) {
		return invalidField(nameField, "debe ser un nombre de host RFC 1123 (minúsculas, dígitos y guiones, sin empezar ni terminar en guion)")
	}

	return nil
}

// validateUnit comprueba que la unidad systemd tenga extensión .service.
func validateUnit(unit string) error {
	if !reUnit.MatchString(unit) {
		return invalidField(unitParam, "debe ser una unidad systemd terminada en .service")
	}

	return nil
}

// validateServer comprueba un identificador «direccion:puerto» con IPv4
// y puerto en el rango 1–65535, y devuelve el destino ya interpretado.
func validateServer(field, server string) (netip.AddrPort, error) {
	address, err := netip.ParseAddrPort(server)
	if err != nil || !address.Addr().Is4() || address.Port() == 0 {
		return netip.AddrPort{},
			invalidField(field, "debe tener el formato direccion:puerto con IPv4 y puerto entre 1 y 65535")
	}

	return address, nil
}

// validatePath comprueba que la ruta sea absoluta, sin traversal ni
// comodines de shell.
func validatePath(field, path string) error {
	switch {
	case path == "":
		return invalidField(field, "no puede estar vacío")
	case !strings.HasPrefix(path, "/"):
		return invalidField(field, "debe ser una ruta absoluta")
	case strings.Contains(path, ".."):
		return invalidField(field, "no puede contener «..»")
	case strings.ContainsAny(path, "*?[]{}"):
		return invalidField(field, "no puede contener comodines de shell")
	case strings.ContainsRune(path, 0):
		return invalidField(field, "no puede contener el carácter NUL")
	default:
		return nil
	}
}

// validateMessageIssue comprueba el límite de longitud y la ausencia de
// NUL en el mensaje de /etc/issue.
func validateMessageIssue(message string) error {
	switch {
	case utf8.RuneCountInString(message) > maxMessageIssue:
		return invalidField(messageField, "no puede superar los 512 caracteres")
	case strings.ContainsRune(message, 0):
		return invalidField(messageField, "no puede contener el carácter NUL")
	case strings.ContainsAny(message, "\r\n"):
		return invalidField(messageField, "no puede contener saltos de línea")
	default:
		return nil
	}
}

// validateIPv4 comprueba que el valor sea una dirección IPv4 válida.
func validateIPv4(field, value string) error {
	address, err := netip.ParseAddr(value)
	if err != nil || !address.Is4() {
		return invalidField(field, "debe ser una dirección IPv4 válida")
	}

	return nil
}

// validatePort comprueba que el puerto esté en el rango 1–65535.
func validatePort(port int) error {
	if port < 1 || port > 65535 {
		return invalidField(portField, "debe estar entre 1 y 65535")
	}

	return nil
}

// validateMode comprueba que los permisos estén en octal.
func validateMode(field, mode string) error {
	if !reMode.MatchString(mode) {
		return invalidField(field, "debe ser un modo octal de 3 o 4 dígitos")
	}

	return nil
}

// booleanoQuery lee un parámetro booleano de la query; vacío equivale a
// false.
func booleanoQuery(r *http.Request, name string) (bool, error) {
	switch value := r.URL.Query().Get(name); value {
	case "", "false":
		return false, nil
	case "true":
		return true, nil
	default:
		return false, invalidField(name, "debe ser true o false")
	}
}
