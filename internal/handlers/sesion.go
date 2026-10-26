// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"consola-vm/internal/services"
)

// SessionDiscoverer es el caso de uso que resuelve el conector
// disponible con la máquina virtual.
type SessionDiscoverer interface {
	Discover(ctx context.Context) (services.SessionInfo, error)
}

// SessionBinder cambia la máquina con la que trabaja la consola.
type SessionBinder interface {
	// Bind establece la máquina indicada por su nombre o UUID.
	Bind(ctx context.Context, identifier string) (services.VirtualMachine, error)
	// Active devuelve el nombre de la máquina vinculada.
	Active() string
}

// SessionOption configura el handler del recurso de sesión.
type SessionOption func(*Session)

// WithBinder habilita PUT /api/session con el servicio dado.
func WithBinder(binder SessionBinder) SessionOption {
	return func(h *Session) {
		h.binder = binder
	}
}

// SessionRegistration informa si la máquina vinculada está registrada.
type SessionRegistration interface {
	IsRegistered(vm string) bool
}

// WithRegistration habilita el estado de registro en GET /api/session.
func WithRegistration(registration SessionRegistration) SessionOption {
	return func(h *Session) {
		h.registration = registration
	}
}

// Session expone GET /api/session, que informa el conector activo, la
// dirección de la máquina y su estado, y PUT /api/session, que vincula
// la consola con una máquina del inventario.
type Session struct {
	discoverer   SessionDiscoverer
	binder       SessionBinder
	registration SessionRegistration
}

// NewSession construye el handler del recurso de sesión.
func NewSession(discoverer SessionDiscoverer, options ...SessionOption) *Session {
	session := &Session{discoverer: discoverer}

	for _, option := range options {
		option(session)
	}

	return session
}

// sessionResponse es el cuerpo JSON de una sesión resuelta.
type sessionResponse struct {
	VM           string `json:"vm,omitempty"`
	Connector    string `json:"conector"`
	IP           string `json:"ip"`
	State        string `json:"estado"`
	IsRegistered bool   `json:"registrada"`
}

// bindRequest es el cuerpo de PUT /api/session.
type bindRequest struct {
	VM string `json:"vm"`
}

// ServeHTTP implementa http.Handler: traduce el descubrimiento al
// contrato JSON y acepta la vinculación de una máquina.
func (h *Session) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPut && h.binder == nil {
		w.Header().Set("Allow", http.MethodGet)
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{
			Message: "Método no permitido para este recurso.",
		})

		return
	}

	if !requireMethod(w, r, http.MethodGet, http.MethodPut) {
		return
	}

	if r.Method == http.MethodPut {
		h.bind(w, r)

		return
	}

	info, err := h.discoverer.Discover(r.Context())
	if err != nil {
		h.respondError(w, err)

		return
	}

	writeJSON(w, http.StatusOK, sessionResponse{
		VM:           h.binderActive(),
		Connector:    info.Connector,
		IP:           info.IP.String(),
		State:        string(info.State),
		IsRegistered: h.estaRegistered(),
	})
}

// bind atiende PUT /api/session: valida el cuerpo, aplica la
// vinculación y devuelve la máquina resultante.
func (h *Session) bind(w http.ResponseWriter, r *http.Request) {
	var request bindRequest

	if err := decodeJSON(w, r, &request); err != nil {
		respondValidacion(w, err)

		return
	}

	if strings.TrimSpace(request.VM) == "" {
		respondValidacion(w, invalidField("vm", "es obligatorio"))

		return
	}

	machine, err := h.binder.Bind(r.Context(), request.VM)
	if err != nil {
		if errors.Is(err, services.ErrMachineNotFound) {
			writeJSON(w, http.StatusNotFound, errorResponse{
				Message:   "La máquina indicada no está registrada en el hipervisor.",
				Technical: err.Error(),
			})

			return
		}

		slog.Error("vincular la máquina virtual", errorKey, err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Message:   "No fue posible vincular la máquina virtual.",
			Technical: err.Error(),
		})

		return
	}

	writeJSON(w, http.StatusOK, newMachineResponse(machine))
}

// binderActive es el nombre de la máquina vinculada, si el handler
// tiene vinculador.
func (h *Session) binderActive() string {
	if h.binder == nil {
		return ""
	}

	return h.binder.Active()
}

// estaRegistered informa si la máquina vinculada ya fue registrada.
func (h *Session) estaRegistered() bool {
	if h.registration == nil {
		return false
	}

	return h.registration.IsRegistered(h.binderActive())
}

// respondError clasifica el fallo del descubrimiento y escribe la
// respuesta HTTP correspondiente, conservando el detalle técnico.
func (h *Session) respondError(w http.ResponseWriter, err error) {
	if errors.Is(err, services.ErrNoMachine) {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{
			Message:   "La consola no está vinculada con ninguna máquina: elige una del inventario.",
			Technical: err.Error(),
		})

		return
	}

	if errors.Is(err, services.ErrSessionUnavailable) {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{
			Message:   "No hay una sesión disponible con la máquina: enciéndela o revisa la red.",
			Technical: err.Error(),
		})

		return
	}

	slog.Error("descubrir la sesión", errorKey, err)
	writeJSON(w, http.StatusInternalServerError, errorResponse{
		Message:   "No fue posible determinar el estado de la máquina.",
		Technical: err.Error(),
	})
}
