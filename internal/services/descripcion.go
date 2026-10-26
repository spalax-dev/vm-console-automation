// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	keyName      = "nombre"
	keyVM        = "vm"
	pathPackages = "/api/packages"
	pathIssue    = "/api/issue"
	pathUsers    = "/api/users"
	pathUnits    = "/api/units"
	pathBackend  = "/api/haproxy/backend/servers"
	pathSession  = "/api/session"
	pathVMs      = "/api/vms"
)

// DescribeOperation construye un resumen legible de una operación para
// el historial. Nunca incluye contraseñas ni cuerpos completos.
func DescribeOperation(method, pattern, path string, data map[string]any) string {
	parts := segments(path)

	switch {
	case strings.HasPrefix(path, pathPackages):
		return describePkg(method, parts, data)
	case strings.HasPrefix(path, "/api/keys/root"),
		strings.HasPrefix(path, pathIssue),
		strings.HasPrefix(path, "/api/hostname"):
		return describeSystem(method, path, data)
	case strings.HasPrefix(path, pathUsers):
		return describeUser(method, parts, data)
	case strings.HasPrefix(path, pathUnits):
		return describeUnit(method, parts, data)
	case strings.HasPrefix(path, pathBackend):
		return describeHAProxy(method, parts, data)
	case strings.HasPrefix(path, pathSession):
		return describeSession(method, path, data)
	case strings.HasPrefix(path, pathVMs):
		return describeMachine(method, parts)
	case strings.HasPrefix(path, "/api/historial"):
		if method == http.MethodGet {
			return "Consultar el historial de operaciones"
		}

		return withoutDescription(method, pattern, path)
	default:
		return withoutDescription(method, pattern, path)
	}
}

// describePkg resume las operaciones sobre paquetes.
func describePkg(method string, parts []string, data map[string]any) string {
	switch method {
	case http.MethodPost:
		if name := text(data, keyName); name != "" {
			return fmt.Sprintf("Instalar el paquete «%s»", name)
		}

		return "Instalar un paquete"
	case http.MethodDelete:
		return fmt.Sprintf("Eliminar el paquete «%s»", last(parts))
	case http.MethodGet:
		if len(parts) >= 3 {
			return fmt.Sprintf("Consultar el paquete «%s»", element(parts))
		}

		return "Listar los paquetes instalados"
	default:
		return "Gestionar paquetes"
	}
}

// describeSystem agrupa las lecturas y escrituras de preparación.
func describeSystem(method, path string, data map[string]any) string {
	switch {
	case strings.HasPrefix(path, "/api/keys/root"):
		return describeSSHKey(method)
	case strings.HasPrefix(path, pathIssue):
		return describeIssue(method)
	default:
		return describeHostname(method, data)
	}
}

// describeSSHKey resume la generación de la llave de root.
func describeSSHKey(method string) string {
	if method == http.MethodPost {
		return "Generar la llave SSH de root"
	}

	return "Descargar la llave privada de root"
}

// describeIssue resume la personalización de /etc/issue.
func describeIssue(method string) string {
	if method == http.MethodPut {
		return "Personalizar /etc/issue"
	}

	return "Consultar /etc/issue"
}

// describeHostname resume el cambio de nombre de host.
func describeHostname(method string, data map[string]any) string {
	if method != http.MethodPut {
		return "Consultar el nombre de host"
	}

	if name := text(data, keyName); name != "" {
		return fmt.Sprintf("Cambiar el nombre de host a «%s»", name)
	}

	return "Cambiar el nombre de host"
}

// describeUser resume las operaciones sobre cuentas de usuario.
func describeUser(method string, parts []string, data map[string]any) string {
	user := element(parts)

	switch method {
	case http.MethodPost:
		if name := text(data, keyName); name != "" {
			return fmt.Sprintf("Crear el usuario «%s»", name)
		}

		return "Crear un usuario"
	case http.MethodDelete:
		if files, ok := data["archivos"].(bool); ok && files {
			return fmt.Sprintf("Eliminar el usuario «%s» y sus archivos", user)
		}

		return fmt.Sprintf("Eliminar el usuario «%s»", user)
	case http.MethodGet:
		if len(parts) >= 4 && last(parts) == "key" {
			return fmt.Sprintf("Descargar la llave privada de «%s»", user)
		}

		if len(parts) >= 3 {
			return fmt.Sprintf("Consultar el usuario «%s»", user)
		}

		return "Listar los usuarios"
	default:
		return fmt.Sprintf("Gestionar el usuario «%s»", user)
	}
}

// describeUnit resume el despliegue y el control de una unidad.
func describeUnit(method string, parts []string, data map[string]any) string {
	unit := element(parts)

	switch method {
	case http.MethodPut:
		return fmt.Sprintf("Desplegar la unidad «%s»", unit)
	case http.MethodPatch:
		return describeStateUnit(unit, data)
	case http.MethodPost:
		return fmt.Sprintf("Reiniciar la unidad «%s»", unit)
	case http.MethodGet:
		if len(parts) >= 3 {
			return fmt.Sprintf("Consultar la unidad «%s»", unit)
		}

		return "Listar las unidades de systemd"
	default:
		return fmt.Sprintf("Consultar la unidad «%s»", unit)
	}
}

// describeHAProxy resume los cambios del back-end.
func describeHAProxy(method string, parts []string, data map[string]any) string {
	switch method {
	case http.MethodPost:
		if destination := server(data); destination != "" {
			return fmt.Sprintf("Agregar el servidor «%s» al back-end", destination)
		}

		return "Agregar un servidor al back-end"
	case http.MethodDelete:
		return fmt.Sprintf("Eliminar el servidor «%s» del back-end", last(parts))
	case http.MethodGet:
		return "Listar los servidores del back-end"
	default:
		return "Consultar el back-end"
	}
}

// describeSession resume el registro y la vinculación de máquinas.
func describeSession(method, path string, data map[string]any) string {
	switch method {
	case http.MethodPost:
		machine := text(data, keyVM)
		if machine == "" {
			return "Registrar una máquina"
		}

		if user := text(data, "usuario"); user != "" {
			return fmt.Sprintf("Registrar la máquina «%s» (usuario %s)", machine, user)
		}

		return fmt.Sprintf("Registrar la máquina «%s»", machine)
	case http.MethodDelete:
		return "Olvidar el registro de la máquina"
	case http.MethodPut:
		if machine := text(data, keyVM); machine != "" {
			return fmt.Sprintf("Vincular la máquina «%s»", machine)
		}

		return "Vincular una máquina"
	case http.MethodGet:
		if strings.Contains(path, "/diagnostico") {
			return "Diagnosticar el registro de la máquina"
		}

		return "Consultar el estado de la sesión"
	default:
		return "Consultar la sesión"
	}
}

// describeMachine resume el encendido de una máquina virtual.
func describeMachine(method string, parts []string) string {
	if method == http.MethodPost {
		return fmt.Sprintf("Encender la máquina «%s»", element(parts))
	}

	return "Listar las máquinas virtuales"
}

// describeStateUnit distingue activar, detener, habilitar y
// deshabilitar.
func describeStateUnit(unit string, data map[string]any) string {
	if active, ok := data["activo"].(bool); ok {
		if active {
			return fmt.Sprintf("Activar la unidad «%s»", unit)
		}

		return fmt.Sprintf("Detener la unidad «%s»", unit)
	}

	if enabled, ok := data["habilitado"].(bool); ok {
		if enabled {
			return fmt.Sprintf("Habilitar el arranque de la unidad «%s»", unit)
		}

		return fmt.Sprintf("Deshabilitar el arranque de la unidad «%s»", unit)
	}

	return fmt.Sprintf("Cambiar el estado de la unidad «%s»", unit)
}

// server arma «direccion:puerto» a partir del cuerpo.
func server(data map[string]any) string {
	address := text(data, "direccion")
	if address == "" {
		return ""
	}

	port, ok := data["puerto"].(float64)
	if !ok {
		return address
	}

	return address + ":" + strconv.Itoa(int(port))
}

// segments divide una ruta en sus partes no vacías y decodificadas.
func segments(path string) []string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i, part := range parts {
		if decodificada, err := url.PathUnescape(part); err == nil {
			parts[i] = decodificada
		}
	}

	return parts
}

// element devuelve el identificador del recurso en la tercera parte
// de la ruta: /api/{recurso}/{id}.
func element(parts []string) string {
	if len(parts) < 3 {
		return ""
	}

	return parts[2]
}

// last devuelve la última parte de la ruta.
func last(parts []string) string {
	if len(parts) == 0 {
		return ""
	}

	return parts[len(parts)-1]
}

// text extrae una cadena del cuerpo y recorta los espacios.
func text(data map[string]any, key string) string {
	value, ok := data[key].(string)
	if !ok {
		return ""
	}

	return strings.TrimSpace(value)
}

// withoutDescription usa el patrón o la ruta cuando no hay un resumen
// específico.
func withoutDescription(method, pattern, path string) string {
	if pattern != "" {
		return method + " " + pattern
	}

	return method + " " + path
}
