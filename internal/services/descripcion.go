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
	claveNombre  = "nombre"
	claveVM      = "vm"
	rutaPaquetes = "/api/packages"
	rutaIssue    = "/api/issue"
	rutaUsuarios = "/api/users"
	rutaUnidades = "/api/units"
	rutaBackend  = "/api/haproxy/backend/servers"
	rutaSesion   = "/api/session"
	rutaVMs      = "/api/vms"
)

// DescribirOperacion construye un resumen legible de una operación para
// el historial. Nunca incluye contraseñas ni cuerpos completos.
func DescribirOperacion(metodo, patron, ruta string, datos map[string]any) string {
	partes := segmentos(ruta)

	switch {
	case strings.HasPrefix(ruta, rutaPaquetes):
		return describirPaquete(metodo, partes, datos)
	case strings.HasPrefix(ruta, "/api/keys/root"),
		strings.HasPrefix(ruta, rutaIssue),
		strings.HasPrefix(ruta, "/api/hostname"):
		return describirSistema(metodo, ruta, datos)
	case strings.HasPrefix(ruta, rutaUsuarios):
		return describirUsuario(metodo, partes, datos)
	case strings.HasPrefix(ruta, rutaUnidades):
		return describirUnidad(metodo, partes, datos)
	case strings.HasPrefix(ruta, rutaBackend):
		return describirHaproxy(metodo, partes, datos)
	case strings.HasPrefix(ruta, rutaSesion):
		return describirSesion(metodo, ruta, datos)
	case strings.HasPrefix(ruta, rutaVMs):
		return describirMaquina(metodo, partes)
	case strings.HasPrefix(ruta, "/api/historial"):
		if metodo == http.MethodGet {
			return "Consultar el historial de operaciones"
		}

		return sinDescripcion(metodo, patron, ruta)
	default:
		return sinDescripcion(metodo, patron, ruta)
	}
}

// describirPaquete resume las operaciones sobre paquetes.
func describirPaquete(metodo string, partes []string, datos map[string]any) string {
	switch metodo {
	case http.MethodPost:
		if nombre := texto(datos, claveNombre); nombre != "" {
			return fmt.Sprintf("Instalar el paquete «%s»", nombre)
		}

		return "Instalar un paquete"
	case http.MethodDelete:
		return fmt.Sprintf("Eliminar el paquete «%s»", ultimo(partes))
	case http.MethodGet:
		if len(partes) >= 3 {
			return fmt.Sprintf("Consultar el paquete «%s»", elemento(partes))
		}

		return "Listar los paquetes instalados"
	default:
		return "Gestionar paquetes"
	}
}

// describirSistema agrupa las lecturas y escrituras de preparación.
func describirSistema(metodo, ruta string, datos map[string]any) string {
	switch {
	case strings.HasPrefix(ruta, "/api/keys/root"):
		return describirLlave(metodo)
	case strings.HasPrefix(ruta, rutaIssue):
		return describirIssue(metodo)
	default:
		return describirHostname(metodo, datos)
	}
}

// describirLlave resume la generación de la llave de root.
func describirLlave(metodo string) string {
	if metodo == http.MethodPost {
		return "Generar la llave SSH de root"
	}

	return "Descargar la llave privada de root"
}

// describirIssue resume la personalización de /etc/issue.
func describirIssue(metodo string) string {
	if metodo == http.MethodPut {
		return "Personalizar /etc/issue"
	}

	return "Consultar /etc/issue"
}

// describirHostname resume el cambio de nombre de host.
func describirHostname(metodo string, datos map[string]any) string {
	if metodo != http.MethodPut {
		return "Consultar el nombre de host"
	}

	if nombre := texto(datos, claveNombre); nombre != "" {
		return fmt.Sprintf("Cambiar el nombre de host a «%s»", nombre)
	}

	return "Cambiar el nombre de host"
}

// describirUsuario resume las operaciones sobre cuentas de usuario.
func describirUsuario(metodo string, partes []string, datos map[string]any) string {
	usuario := elemento(partes)

	switch metodo {
	case http.MethodPost:
		if nombre := texto(datos, claveNombre); nombre != "" {
			return fmt.Sprintf("Crear el usuario «%s»", nombre)
		}

		return "Crear un usuario"
	case http.MethodDelete:
		if archivos, ok := datos["archivos"].(bool); ok && archivos {
			return fmt.Sprintf("Eliminar el usuario «%s» y sus archivos", usuario)
		}

		return fmt.Sprintf("Eliminar el usuario «%s»", usuario)
	case http.MethodGet:
		if len(partes) >= 4 && ultimo(partes) == "key" {
			return fmt.Sprintf("Descargar la llave privada de «%s»", usuario)
		}

		if len(partes) >= 3 {
			return fmt.Sprintf("Consultar el usuario «%s»", usuario)
		}

		return "Listar los usuarios"
	default:
		return fmt.Sprintf("Gestionar el usuario «%s»", usuario)
	}
}

// describirUnidad resume el despliegue y el control de una unidad.
func describirUnidad(metodo string, partes []string, datos map[string]any) string {
	unidad := elemento(partes)

	switch metodo {
	case http.MethodPut:
		return fmt.Sprintf("Desplegar la unidad «%s»", unidad)
	case http.MethodPatch:
		return describirEstadoUnidad(unidad, datos)
	case http.MethodPost:
		return fmt.Sprintf("Reiniciar la unidad «%s»", unidad)
	case http.MethodGet:
		if len(partes) >= 3 {
			return fmt.Sprintf("Consultar la unidad «%s»", unidad)
		}

		return "Listar las unidades de systemd"
	default:
		return fmt.Sprintf("Consultar la unidad «%s»", unidad)
	}
}

// describirHaproxy resume los cambios del back-end.
func describirHaproxy(metodo string, partes []string, datos map[string]any) string {
	switch metodo {
	case http.MethodPost:
		if destino := servidor(datos); destino != "" {
			return fmt.Sprintf("Agregar el servidor «%s» al back-end", destino)
		}

		return "Agregar un servidor al back-end"
	case http.MethodDelete:
		return fmt.Sprintf("Eliminar el servidor «%s» del back-end", ultimo(partes))
	case http.MethodGet:
		return "Listar los servidores del back-end"
	default:
		return "Consultar el back-end"
	}
}

// describirSesion resume el registro y la vinculación de máquinas.
func describirSesion(metodo, ruta string, datos map[string]any) string {
	switch metodo {
	case http.MethodPost:
		maquina := texto(datos, claveVM)
		if maquina == "" {
			return "Registrar una máquina"
		}

		if usuario := texto(datos, "usuario"); usuario != "" {
			return fmt.Sprintf("Registrar la máquina «%s» (usuario %s)", maquina, usuario)
		}

		return fmt.Sprintf("Registrar la máquina «%s»", maquina)
	case http.MethodDelete:
		return "Olvidar el registro de la máquina"
	case http.MethodPut:
		if maquina := texto(datos, claveVM); maquina != "" {
			return fmt.Sprintf("Vincular la máquina «%s»", maquina)
		}

		return "Vincular una máquina"
	case http.MethodGet:
		if strings.Contains(ruta, "/diagnostico") {
			return "Diagnosticar el registro de la máquina"
		}

		return "Consultar el estado de la sesión"
	default:
		return "Consultar la sesión"
	}
}

// describirMaquina resume el encendido de una máquina virtual.
func describirMaquina(metodo string, partes []string) string {
	if metodo == http.MethodPost {
		return fmt.Sprintf("Encender la máquina «%s»", elemento(partes))
	}

	return "Listar las máquinas virtuales"
}

// describirEstadoUnidad distingue activar, detener, habilitar y
// deshabilitar.
func describirEstadoUnidad(unidad string, datos map[string]any) string {
	if activo, ok := datos["activo"].(bool); ok {
		if activo {
			return fmt.Sprintf("Activar la unidad «%s»", unidad)
		}

		return fmt.Sprintf("Detener la unidad «%s»", unidad)
	}

	if habilitado, ok := datos["habilitado"].(bool); ok {
		if habilitado {
			return fmt.Sprintf("Habilitar el arranque de la unidad «%s»", unidad)
		}

		return fmt.Sprintf("Deshabilitar el arranque de la unidad «%s»", unidad)
	}

	return fmt.Sprintf("Cambiar el estado de la unidad «%s»", unidad)
}

// servidor arma «direccion:puerto» a partir del cuerpo.
func servidor(datos map[string]any) string {
	direccion := texto(datos, "direccion")
	if direccion == "" {
		return ""
	}

	puerto, ok := datos["puerto"].(float64)
	if !ok {
		return direccion
	}

	return direccion + ":" + strconv.Itoa(int(puerto))
}

// segmentos divide una ruta en sus partes no vacías y decodificadas.
func segmentos(ruta string) []string {
	partes := strings.Split(strings.Trim(ruta, "/"), "/")
	for i, parte := range partes {
		if decodificada, err := url.PathUnescape(parte); err == nil {
			partes[i] = decodificada
		}
	}

	return partes
}

// elemento devuelve el identificador del recurso en la tercera parte
// de la ruta: /api/{recurso}/{id}.
func elemento(partes []string) string {
	if len(partes) < 3 {
		return ""
	}

	return partes[2]
}

// ultimo devuelve la última parte de la ruta.
func ultimo(partes []string) string {
	if len(partes) == 0 {
		return ""
	}

	return partes[len(partes)-1]
}

// texto extrae una cadena del cuerpo y recorta los espacios.
func texto(datos map[string]any, clave string) string {
	valor, ok := datos[clave].(string)
	if !ok {
		return ""
	}

	return strings.TrimSpace(valor)
}

// sinDescripcion usa el patrón o la ruta cuando no hay un resumen
// específico.
func sinDescripcion(metodo, patron, ruta string) string {
	if patron != "" {
		return metodo + " " + patron
	}

	return metodo + " " + ruta
}
