// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"consola-vm/internal/services"
)

// ServicioHaproxy ejecuta las operaciones sobre el back-end de HAProxy.
type ServicioHaproxy interface {
	// Listar devuelve los servidores del back-end.
	Listar(ctx context.Context) ([]services.Servidor, error)
	// Agregar agrega un servidor al back-end.
	Agregar(ctx context.Context, direccion string, puerto int) (services.Resultado, error)
	// Eliminar elimina un servidor del back-end.
	Eliminar(ctx context.Context, direccion string, puerto int) (services.Resultado, error)
}

// Haproxy agrupa los recursos de balanceo de carga con HAProxy
// (numeral 3.4).
type Haproxy struct {
	servicio ServicioHaproxy
}

// NuevoHaproxy construye el handler de balanceo con HAProxy.
func NuevoHaproxy(servicio ServicioHaproxy) *Haproxy {
	return &Haproxy{servicio: servicio}
}

// servidorBackend es el cuerpo de POST /api/haproxy/backend/servers.
type servidorBackend struct {
	Direccion string `json:"direccion"`
	Puerto    int    `json:"puerto"`
}

// servidorRespuesta es un servidor en la respuesta JSON.
type servidorRespuesta struct {
	Direccion string `json:"direccion"`
	Puerto    int    `json:"puerto"`
	Estado    string `json:"estado"`
}

// listaBackend es el cuerpo de GET /api/haproxy/backend/servers.
type listaBackend struct {
	Servidores []servidorRespuesta `json:"servidores"`
}

// Servidores atiende GET y POST /api/haproxy/backend/servers: lista o
// agrega servidores al back-end.
func (h *Haproxy) Servidores(w http.ResponseWriter, r *http.Request) {
	if !exigirMetodo(w, r, http.MethodGet, http.MethodPost) {
		return
	}

	if r.Method == http.MethodGet {
		h.listarBackend(w, r)

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

	resultado, err := h.servicio.Agregar(r.Context(), solicitud.Direccion, solicitud.Puerto)
	if errors.Is(err, services.ErrServidorExistente) {
		escribirJSON(w, http.StatusConflict, respuestaOperacion{
			Mensaje: "El servidor ya está en el back-end.",
			Tecnico: err.Error(),
		})

		return
	}

	if err != nil {
		responderErrorOperacion(w, resultado, err)

		return
	}

	w.Header().Set(cabeceraLocation, "/api/haproxy/backend/servers/"+solicitud.Direccion+":"+strconv.Itoa(solicitud.Puerto))
	escribirResultado(w, http.StatusCreated, resultado)
}

// listarBackend responde la colección de servidores.
func (h *Haproxy) listarBackend(w http.ResponseWriter, r *http.Request) {
	servidores, err := h.servicio.Listar(r.Context())
	if err != nil {
		responderErrorOperacion(w, services.Resultado{}, err)

		return
	}

	respuesta := listaBackend{Servidores: make([]servidorRespuesta, 0, len(servidores))}
	for _, servidor := range servidores {
		respuesta.Servidores = append(respuesta.Servidores, servidorRespuesta{
			Direccion: servidor.Direccion,
			Puerto:    servidor.Puerto,
			Estado:    servidor.Estado,
		})
	}

	escribirJSON(w, http.StatusOK, respuesta)
}

// Servidor atiende DELETE /api/haproxy/backend/servers/{server}: elimina
// el servidor identificado como «direccion:puerto».
func (h *Haproxy) Servidor(w http.ResponseWriter, r *http.Request) {
	if !exigirMetodo(w, r, http.MethodDelete) {
		return
	}

	servidor := r.PathValue(campoServer)

	destino, err := validarServidor(campoServer, servidor)
	if err != nil {
		responderValidacion(w, err)

		return
	}

	resultado, err := h.servicio.Eliminar(r.Context(), destino.Addr().String(), int(destino.Port()))
	if errors.Is(err, services.ErrServidorNoEncontrado) {
		escribirJSON(w, http.StatusNotFound, respuestaOperacion{
			Mensaje: "El servidor no está en el back-end.",
			Tecnico: err.Error(),
		})

		return
	}

	if err != nil {
		responderErrorOperacion(w, resultado, err)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}
