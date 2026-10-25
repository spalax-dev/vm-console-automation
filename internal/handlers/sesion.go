// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"consola-vm/internal/services"
)

// DescubridorSesion es el caso de uso que resuelve el conector
// disponible con la máquina virtual.
type DescubridorSesion interface {
	Descubrir(ctx context.Context) (services.InfoSesion, error)
}

// Sesion expone GET /api/session: informa el conector activo, la
// dirección de la máquina y su estado de encendido.
type Sesion struct {
	descubridor DescubridorSesion
}

// NuevoSesion construye el handler del recurso de sesión.
func NuevoSesion(descubridor DescubridorSesion) *Sesion {
	return &Sesion{descubridor: descubridor}
}

// respuestaSesion es el cuerpo JSON de una sesión resuelta.
type respuestaSesion struct {
	Conector string `json:"conector"`
	IP       string `json:"ip"`
	Estado   string `json:"estado"`
}

// ServeHTTP implementa http.Handler: solo admite GET y traduce el
// resultado del descubrimiento al contrato JSON de la consola.
func (h *Sesion) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !exigirMetodo(w, r, http.MethodGet) {
		return
	}

	info, err := h.descubridor.Descubrir(r.Context())
	if err != nil {
		h.responderError(w, err)

		return
	}

	escribirJSON(w, http.StatusOK, respuestaSesion{
		Conector: info.Conector,
		IP:       info.IP.String(),
		Estado:   string(info.Estado),
	})
}

// responderError clasifica el fallo del descubrimiento y escribe la
// respuesta HTTP correspondiente, conservando el detalle técnico.
func (h *Sesion) responderError(w http.ResponseWriter, err error) {
	if errors.Is(err, services.ErrSesionNoDisponible) {
		escribirJSON(w, http.StatusServiceUnavailable, respuestaError{
			Mensaje: "No hay una sesión disponible con la máquina: enciéndela o revisa la red.",
			Tecnico: err.Error(),
		})

		return
	}

	slog.Error("descubrir la sesión", "error", err)
	escribirJSON(w, http.StatusInternalServerError, respuestaError{
		Mensaje: "No fue posible determinar el estado de la máquina.",
		Tecnico: err.Error(),
	})
}
