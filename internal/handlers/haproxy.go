// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import "net/http"

// Haproxy agrupa los recursos de balanceo de carga con HAProxy
// (numeral 3.4).
type Haproxy struct{}

// NuevoHaproxy construye el handler de balanceo con HAProxy.
func NuevoHaproxy() *Haproxy {
	return &Haproxy{}
}

// servidorBackend es el cuerpo de POST /api/haproxy/backend/servers.
type servidorBackend struct {
	Direccion string `json:"direccion"`
	Puerto    int    `json:"puerto"`
}

// Servidores atiende GET y POST /api/haproxy/backend/servers: lista o
// agrega servidores al back-end.
func (h *Haproxy) Servidores(w http.ResponseWriter, r *http.Request) {
	if !exigirMetodo(w, r, http.MethodGet, http.MethodPost) {
		return
	}

	if r.Method == http.MethodGet {
		noImplementado(w, "listar los servidores del back-end")

		return
	}

	var solicitud servidorBackend
	if err := decodificarJSON(w, r, &solicitud); err != nil {
		responderValidacion(w, err)

		return
	}

	if err := validarIPv4(campoDireccion, solicitud.Direccion); err != nil {
		responderValidacion(w, err)

		return
	}

	if err := validarPuerto(solicitud.Puerto); err != nil {
		responderValidacion(w, err)

		return
	}

	noImplementado(w, "agregar un servidor al back-end")
}

// Servidor atiende DELETE /api/haproxy/backend/servers/{server}: elimina
// el servidor identificado como «direccion:puerto».
func (h *Haproxy) Servidor(w http.ResponseWriter, r *http.Request) {
	if !exigirMetodo(w, r, http.MethodDelete) {
		return
	}

	servidor := r.PathValue(campoServer)
	if err := validarServidor(campoServer, servidor); err != nil {
		responderValidacion(w, err)

		return
	}

	noImplementado(w, "eliminar el servidor "+servidor)
}
