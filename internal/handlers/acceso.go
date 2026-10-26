// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"consola-vm/internal/services"
)

// AccessService registra una máquina en la consola.
type AccessService interface {
	// Register guarda las credenciales y prepara SSH.
	Register(ctx context.Context, user, password string) (services.Outcome, error)
	// Forget elimina el registro y las credenciales.
	Forget() error
}

// Access expone POST y DELETE /api/session/register: registra la máquina
// vinculada o elimina su registro.
type Access struct {
	service AccessService
}

// NewAccess construye el handler de credenciales.
func NewAccess(service AccessService) *Access {
	return &Access{service: service}
}

// accessRequest es el cuerpo de POST /api/session/login.
type accessRequest struct {
	User     string `json:"usuario"`
	Password string `json:"password"`
}

// ServeHTTP implementa http.Handler.
func (h *Access) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost, http.MethodDelete) {
		return
	}

	if r.Method == http.MethodDelete {
		h.forget(w)

		return
	}

	h.register(w, r)
}

// register valida y registra la máquina.
func (h *Access) register(w http.ResponseWriter, r *http.Request) {
	var request accessRequest
	if err := decodeJSON(w, r, &request); err != nil {
		respondValidacion(w, err)

		return
	}

	request.User = strings.TrimSpace(request.User)

	if err := validateUser(userField, request.User); err != nil {
		respondValidacion(w, err)

		return
	}

	result, err := h.service.Register(r.Context(), request.User, request.Password)
	switch {
	case errors.Is(err, services.ErrNoMachine):
		writeJSON(w, http.StatusConflict,
			responseFromError(result, "Primero elige una máquina virtual.", err))
	case errors.Is(err, services.ErrInvalidCredentials):
		writeJSON(w, http.StatusUnauthorized, responseFromError(result,
			"Guest Additions rechazó las credenciales o no pudo ejecutar la orden.", err))
	case errors.Is(err, services.ErrSessionUnavailable):
		writeJSON(w, http.StatusServiceUnavailable, responseFromError(result,
			"La máquina no está encendida o no publica Guest Additions.", err))
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, responseFromError(result,
			"No fue posible guardar o verificar las credenciales.", err))
	default:
		writeResult(w, http.StatusCreated, result)
	}
}

// forget elimina las credenciales de la máquina activa.
func (h *Access) forget(w http.ResponseWriter) {
	if err := h.service.Forget(); err != nil {
		code := http.StatusInternalServerError
		if errors.Is(err, services.ErrNoMachine) {
			code = http.StatusConflict
		}

		writeJSON(w, code, errorResponse{
			Message:   "No fue posible olvidar las credenciales.",
			Technical: err.Error(),
		})

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// userField es el nombre del campo de usuario del cuerpo.
const userField = "usuario"

// passwordField es el nombre del campo de contraseña del cuerpo.
const passwordField = "password"
