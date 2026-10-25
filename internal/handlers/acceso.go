// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"context"
	"errors"
	"net/http"

	"consola-vm/internal/services"
)

// ServicioAcceso registra una máquina en la consola.
type ServicioAcceso interface {
	// Registrar guarda las credenciales y prepara SSH.
	Registrar(ctx context.Context, usuario, password string) (services.Resultado, error)
	// Olvidar elimina el registro y las credenciales.
	Olvidar() error
}

// Acceso expone POST y DELETE /api/session/register: registra la máquina
// vinculada o elimina su registro.
type Acceso struct {
	servicio ServicioAcceso
}

// NuevoAcceso construye el handler de credenciales.
func NuevoAcceso(servicio ServicioAcceso) *Acceso {
	return &Acceso{servicio: servicio}
}

// solicitudAcceso es el cuerpo de POST /api/session/login.
type solicitudAcceso struct {
	Usuario  string `json:"usuario"`
	Password string `json:"password"`
}

// ServeHTTP implementa http.Handler.
func (h *Acceso) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !exigirMetodo(w, r, http.MethodPost, http.MethodDelete) {
		return
	}

	if r.Method == http.MethodDelete {
		h.olvidar(w)

		return
	}

	h.registrar(w, r)
}

// registrar valida y registra la máquina.
func (h *Acceso) registrar(w http.ResponseWriter, r *http.Request) {
	var solicitud solicitudAcceso
	if err := decodificarJSON(w, r, &solicitud); err != nil {
		responderValidacion(w, err)

		return
	}

	if err := validarUsuario(campoUsuario, solicitud.Usuario); err != nil {
		responderValidacion(w, err)

		return
	}

	resultado, err := h.servicio.Registrar(r.Context(), solicitud.Usuario, solicitud.Password)
	switch {
	case errors.Is(err, services.ErrSinMaquina):
		escribirJSON(w, http.StatusConflict, respuestaOperacion{
			Mensaje: "Primero elige una máquina virtual.",
			Tecnico: err.Error(),
		})
	case errors.Is(err, services.ErrCredencialesInvalidas):
		escribirJSON(w, http.StatusUnauthorized, respuestaOperacion{
			Mensaje:   "Guest Additions rechazó las credenciales o no pudo ejecutar la orden.",
			Operacion: resultado.Operacion,
			Codigo:    resultado.Codigo,
			Stderr:    resultado.Stderr,
			Tecnico:   err.Error(),
		})
	case errors.Is(err, services.ErrSesionNoDisponible):
		escribirJSON(w, http.StatusServiceUnavailable, respuestaOperacion{
			Mensaje: "La máquina no está encendida o no publica Guest Additions.",
			Tecnico: err.Error(),
		})
	case err != nil:
		escribirJSON(w, http.StatusInternalServerError, respuestaOperacion{
			Mensaje: "No fue posible guardar o verificar las credenciales.",
			Tecnico: err.Error(),
		})
	default:
		escribirResultado(w, http.StatusCreated, resultado)
	}
}

// olvidar elimina las credenciales de la máquina activa.
func (h *Acceso) olvidar(w http.ResponseWriter) {
	if err := h.servicio.Olvidar(); err != nil {
		codigo := http.StatusInternalServerError
		if errors.Is(err, services.ErrSinMaquina) {
			codigo = http.StatusConflict
		}

		escribirJSON(w, codigo, respuestaOperacion{
			Mensaje: "No fue posible olvidar las credenciales.",
			Tecnico: err.Error(),
		})

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// campoUsuario es el nombre del campo de usuario del cuerpo.
const campoUsuario = "usuario"
