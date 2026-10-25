// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"context"
	"encoding/json"
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

// Sesion expone GET /api/sesion: informa el conector activo, la
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

// respuestaError es el cuerpo JSON común de los errores de la consola.
type respuestaError struct {
	OK      bool   `json:"ok"`
	Mensaje string `json:"mensaje"`
	Tecnico string `json:"tecnico,omitempty"`
}

// ServeHTTP implementa http.Handler: solo admite GET y traduce el
// resultado del descubrimiento al contrato JSON de la consola.
func (h *Sesion) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		escribirJSON(w, http.StatusMethodNotAllowed, respuestaError{
			Mensaje: "El recurso de sesión solo admite GET.",
		})

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

// escribirJSON serializa cuerpo como JSON con el código de estado dado.
// Si la serialización falla, responde 500 sin cuerpo adicional.
func escribirJSON(w http.ResponseWriter, codigo int, cuerpo any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(codigo)

	if err := json.NewEncoder(w).Encode(cuerpo); err != nil {
		slog.Error("escribir la respuesta JSON", "error", err)
	}
}
