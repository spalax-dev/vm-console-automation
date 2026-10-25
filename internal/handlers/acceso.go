// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"context"
	"errors"
	"net/http"

	"consola-vm/internal/services"
)

// ServicioAcceso valida y guarda las credenciales de la máquina activa.
type ServicioAcceso interface {
	// Iniciar guarda y verifica las credenciales.
	Iniciar(ctx context.Context, usuario, password string, puerto uint16) (services.Resultado, error)
	// Olvidar elimina las credenciales guardadas.
	Olvidar() error
}

// Acceso expone POST y DELETE /api/session/login: guarda o elimina las
// credenciales de Guest Additions de la máquina vinculada.
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
	Puerto   int    `json:"puerto,omitempty"`
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

	h.iniciar(w, r)
}

// iniciar valida y guarda las credenciales.
func (h *Acceso) iniciar(w http.ResponseWriter, r *http.Request) {
	var solicitud solicitudAcceso
	if err := decodificarJSON(w, r, &solicitud); err != nil {
		responderValidacion(w, err)

		return
	}

	if err := validarUsuario(campoUsuario, solicitud.Usuario); err != nil {
		responderValidacion(w, err)

		return
	}

	if solicitud.Puerto != 0 && (solicitud.Puerto < 1 || solicitud.Puerto > 65535) {
		responderValidacion(w, campoInvalido(campoPuerto, "debe estar entre 1 y 65535"))

		return
	}

	// #nosec G115 -- el puerto se validó entre 0 y 65535.
	resultado, err := h.servicio.Iniciar(
		r.Context(), solicitud.Usuario, solicitud.Password, uint16(solicitud.Puerto),
	)
	switch {
	case errors.Is(err, services.ErrSinMaquina):
		escribirJSON(w, http.StatusConflict, respuestaOperacion{
			Mensaje: "Primero elige una máquina virtual.",
			Tecnico: err.Error(),
		})
	case errors.Is(err, services.ErrCredencialesInvalidas):
		escribirJSON(w, http.StatusUnauthorized, respuestaOperacion{
			Mensaje: "No fue posible iniciar sesión con esas credenciales.",
			Tecnico: err.Error(),
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
		escribirResultado(w, http.StatusOK, resultado)
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
