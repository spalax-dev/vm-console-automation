// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

// Package handlers expone los endpoints HTTP de la consola de
// administración: preparación del sistema, gestión de usuarios,
// gestión de servicios systemd y balanceo de carga con HAProxy. Los
// handlers solo enrutan, validan y traducen a JSON; los casos de uso
// viven en internal/services.
package handlers

import "net/http"

// Sistema agrupa los recursos de preparación del sistema (numeral 3.1
// del enunciado).
type Sistema struct{}

// NuevoSistema construye el handler de preparación del sistema.
func NuevoSistema() *Sistema {
	return &Sistema{}
}

// solicitudPaquete es el cuerpo de POST /api/packages.
type solicitudPaquete struct {
	Nombre string `json:"nombre"`
}

// solicitudIssue es el cuerpo de PUT /api/issue.
type solicitudIssue struct {
	Mensaje    string `json:"mensaje"`
	NombreHost bool   `json:"nombre_host"`
	Fecha      bool   `json:"fecha"`
	Hora       bool   `json:"hora"`
	IP         bool   `json:"ip"`
}

// solicitudHostname es el cuerpo de PUT /api/hostname.
type solicitudHostname struct {
	Nombre          string `json:"nombre"`
	IP              string `json:"ip"`
	ActualizarHosts bool   `json:"actualizar_hosts"`
}

// Paquetes atiende GET y POST /api/packages: lista los paquetes
// instalados o instala uno nuevo.
func (h *Sistema) Paquetes(w http.ResponseWriter, r *http.Request) {
	if !exigirMetodo(w, r, http.MethodGet, http.MethodPost) {
		return
	}

	if r.Method == http.MethodGet {
		noImplementado(w, "listar los paquetes instalados")

		return
	}

	var solicitud solicitudPaquete
	if err := decodificarJSON(w, r, &solicitud); err != nil {
		responderValidacion(w, err)

		return
	}

	if err := validarPaquete(campoNombre, solicitud.Nombre); err != nil {
		responderValidacion(w, err)

		return
	}

	noImplementado(w, "instalar el paquete "+solicitud.Nombre)
}

// Paquete atiende DELETE /api/packages/{name}: elimina el paquete
// indicado en la ruta.
func (h *Sistema) Paquete(w http.ResponseWriter, r *http.Request) {
	if !exigirMetodo(w, r, http.MethodDelete) {
		return
	}

	nombre := r.PathValue(campoName)
	if err := validarPaquete(campoName, nombre); err != nil {
		responderValidacion(w, err)

		return
	}

	noImplementado(w, "eliminar el paquete "+nombre)
}

// LlaveRoot atiende GET y POST /api/keys/root: descarga la llave
// privada de root o genera un par nuevo.
func (h *Sistema) LlaveRoot(w http.ResponseWriter, r *http.Request) {
	if !exigirMetodo(w, r, http.MethodGet, http.MethodPost) {
		return
	}

	if r.Method == http.MethodGet {
		noImplementado(w, "descargar la llave privada de root")

		return
	}

	noImplementado(w, "generar la llave SSH de root")
}

// Issue atiende GET y PUT /api/issue: consulta o reemplaza el banner
// previo al inicio de sesión.
func (h *Sistema) Issue(w http.ResponseWriter, r *http.Request) {
	if !exigirMetodo(w, r, http.MethodGet, http.MethodPut) {
		return
	}

	if r.Method == http.MethodGet {
		noImplementado(w, "consultar /etc/issue")

		return
	}

	var solicitud solicitudIssue
	if err := decodificarJSON(w, r, &solicitud); err != nil {
		responderValidacion(w, err)

		return
	}

	if err := validarMensajeIssue(solicitud.Mensaje); err != nil {
		responderValidacion(w, err)

		return
	}

	noImplementado(w, "personalizar /etc/issue")
}

// Hostname atiende GET y PUT /api/hostname: consulta o reemplaza el
// nombre de host y su entrada en /etc/hosts.
func (h *Sistema) Hostname(w http.ResponseWriter, r *http.Request) {
	if !exigirMetodo(w, r, http.MethodGet, http.MethodPut) {
		return
	}

	if r.Method == http.MethodGet {
		noImplementado(w, "consultar el nombre de host")

		return
	}

	var solicitud solicitudHostname
	if err := decodificarJSON(w, r, &solicitud); err != nil {
		responderValidacion(w, err)

		return
	}

	if err := validarHostname(solicitud.Nombre); err != nil {
		responderValidacion(w, err)

		return
	}

	if err := validarIPv4(campoIP, solicitud.IP); err != nil {
		responderValidacion(w, err)

		return
	}

	noImplementado(w, "asignar el nombre de host")
}
