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

// DescubridorSesion es el caso de uso que resuelve el conector
// disponible con la máquina virtual.
type DescubridorSesion interface {
	Descubrir(ctx context.Context) (services.InfoSesion, error)
}

// VinculadorSesion cambia la máquina con la que trabaja la consola.
type VinculadorSesion interface {
	// Vincular establece la máquina indicada por su nombre o UUID.
	Vincular(ctx context.Context, identificador string) (services.MaquinaVirtual, error)
	// Activa devuelve el nombre de la máquina vinculada.
	Activa() string
}

// OpcionSesion configura el handler del recurso de sesión.
type OpcionSesion func(*Sesion)

// ConVinculador habilita PUT /api/session con el servicio dado.
func ConVinculador(vinculador VinculadorSesion) OpcionSesion {
	return func(h *Sesion) {
		h.vinculador = vinculador
	}
}

// RegistroSesion informa si la máquina vinculada está registrada.
type RegistroSesion interface {
	Registrada(vm string) bool
}

// ConRegistro habilita el estado de registro en GET /api/session.
func ConRegistro(registro RegistroSesion) OpcionSesion {
	return func(h *Sesion) {
		h.registro = registro
	}
}

// Sesion expone GET /api/session, que informa el conector activo, la
// dirección de la máquina y su estado, y PUT /api/session, que vincula
// la consola con una máquina del inventario.
type Sesion struct {
	descubridor DescubridorSesion
	vinculador  VinculadorSesion
	registro    RegistroSesion
}

// NuevoSesion construye el handler del recurso de sesión.
func NuevoSesion(descubridor DescubridorSesion, opciones ...OpcionSesion) *Sesion {
	sesion := &Sesion{descubridor: descubridor}

	for _, opcion := range opciones {
		opcion(sesion)
	}

	return sesion
}

// respuestaSesion es el cuerpo JSON de una sesión resuelta.
type respuestaSesion struct {
	VM         string `json:"vm,omitempty"`
	Conector   string `json:"conector"`
	IP         string `json:"ip"`
	Estado     string `json:"estado"`
	Registrada bool   `json:"registrada"`
}

// solicitudVinculacion es el cuerpo de PUT /api/session.
type solicitudVinculacion struct {
	VM string `json:"vm"`
}

// ServeHTTP implementa http.Handler: traduce el descubrimiento al
// contrato JSON y acepta la vinculación de una máquina.
func (h *Sesion) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPut && h.vinculador == nil {
		w.Header().Set("Allow", http.MethodGet)
		escribirJSON(w, http.StatusMethodNotAllowed, respuestaError{
			Mensaje: "Método no permitido para este recurso.",
		})

		return
	}

	if !exigirMetodo(w, r, http.MethodGet, http.MethodPut) {
		return
	}

	if r.Method == http.MethodPut {
		h.vincular(w, r)

		return
	}

	info, err := h.descubridor.Descubrir(r.Context())
	if err != nil {
		h.responderError(w, err)

		return
	}

	escribirJSON(w, http.StatusOK, respuestaSesion{
		VM:         h.vinculadorActivo(),
		Conector:   info.Conector,
		IP:         info.IP.String(),
		Estado:     string(info.Estado),
		Registrada: h.estaRegistrada(),
	})
}

// vincular atiende PUT /api/session: valida el cuerpo, aplica la
// vinculación y devuelve la máquina resultante.
func (h *Sesion) vincular(w http.ResponseWriter, r *http.Request) {
	var solicitud solicitudVinculacion

	if err := decodificarJSON(w, r, &solicitud); err != nil {
		responderValidacion(w, err)

		return
	}

	if strings.TrimSpace(solicitud.VM) == "" {
		responderValidacion(w, campoInvalido("vm", "es obligatorio"))

		return
	}

	maquina, err := h.vinculador.Vincular(r.Context(), solicitud.VM)
	if err != nil {
		if errors.Is(err, services.ErrMaquinaNoEncontrada) {
			escribirJSON(w, http.StatusNotFound, respuestaError{
				Mensaje: "La máquina indicada no está registrada en el hipervisor.",
				Tecnico: err.Error(),
			})

			return
		}

		slog.Error("vincular la máquina virtual", claveError, err)
		escribirJSON(w, http.StatusInternalServerError, respuestaError{
			Mensaje: "No fue posible vincular la máquina virtual.",
			Tecnico: err.Error(),
		})

		return
	}

	escribirJSON(w, http.StatusOK, nuevaMaquinaRespuesta(maquina))
}

// vinculadorActivo es el nombre de la máquina vinculada, si el handler
// tiene vinculador.
func (h *Sesion) vinculadorActivo() string {
	if h.vinculador == nil {
		return ""
	}

	return h.vinculador.Activa()
}

// estaRegistrada informa si la máquina vinculada ya fue registrada.
func (h *Sesion) estaRegistrada() bool {
	if h.registro == nil {
		return false
	}

	return h.registro.Registrada(h.vinculadorActivo())
}

// responderError clasifica el fallo del descubrimiento y escribe la
// respuesta HTTP correspondiente, conservando el detalle técnico.
func (h *Sesion) responderError(w http.ResponseWriter, err error) {
	if errors.Is(err, services.ErrSinMaquina) {
		escribirJSON(w, http.StatusServiceUnavailable, respuestaError{
			Mensaje: "La consola no está vinculada con ninguna máquina: elige una del inventario.",
			Tecnico: err.Error(),
		})

		return
	}

	if errors.Is(err, services.ErrSesionNoDisponible) {
		escribirJSON(w, http.StatusServiceUnavailable, respuestaError{
			Mensaje: "No hay una sesión disponible con la máquina: enciéndela o revisa la red.",
			Tecnico: err.Error(),
		})

		return
	}

	slog.Error("descubrir la sesión", claveError, err)
	escribirJSON(w, http.StatusInternalServerError, respuestaError{
		Mensaje: "No fue posible determinar el estado de la máquina.",
		Tecnico: err.Error(),
	})
}
