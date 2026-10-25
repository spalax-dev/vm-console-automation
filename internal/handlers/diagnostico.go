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

// ServicioDiagnostico explica por qué una máquina no se puede registrar.
type ServicioDiagnostico interface {
	Obtener(ctx context.Context) (services.Diagnostico, error)
}

// Diagnostico expone GET /api/session/diagnostico.
type Diagnostico struct {
	servicio ServicioDiagnostico
}

// NuevoDiagnostico construye el handler del diagnóstico.
func NuevoDiagnostico(servicio ServicioDiagnostico) *Diagnostico {
	return &Diagnostico{servicio: servicio}
}

// diagnosticoRespuesta es el cuerpo JSON del diagnóstico.
type diagnosticoRespuesta struct {
	VM                     string   `json:"vm"`
	Estado                 string   `json:"estado"`
	Red                    string   `json:"red"`
	IP                     string   `json:"ip,omitempty"`
	Registrada             bool     `json:"registrada"`
	GuestAdditionsVersion  string   `json:"guest_additions_version,omitempty"`
	GuestAdditionsRunLevel int      `json:"guest_additions_run_level"`
	UsuarioSugerido        string   `json:"usuario_sugerido,omitempty"`
	SSHAbierto             bool     `json:"ssh_abierto"`
	Problemas              []string `json:"problemas"`
	Sugerencias            []string `json:"sugerencias"`
}

// ServeHTTP implementa http.Handler.
func (h *Diagnostico) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !exigirMetodo(w, r, http.MethodGet) {
		return
	}

	informe, err := h.servicio.Obtener(r.Context())
	if errors.Is(err, services.ErrSinMaquina) {
		escribirJSON(w, http.StatusConflict, respuestaOperacion{
			Mensaje: "Primero elige una máquina virtual.",
			Tecnico: err.Error(),
		})

		return
	}

	if err != nil {
		slog.Error("diagnosticar la máquina", claveError, err)
		escribirJSON(w, http.StatusInternalServerError, respuestaOperacion{
			Mensaje: "No fue posible diagnosticar la máquina.",
			Tecnico: err.Error(),
		})

		return
	}

	ip := ""
	if informe.IP.IsValid() {
		ip = informe.IP.String()
	}

	escribirJSON(w, http.StatusOK, diagnosticoRespuesta{
		VM:                     informe.VM,
		Estado:                 string(informe.Estado),
		Red:                    string(informe.Red),
		IP:                     ip,
		Registrada:             informe.Registrada,
		GuestAdditionsVersion:  informe.GuestAdditionsVersion,
		GuestAdditionsRunLevel: informe.GuestAdditionsRunLevel,
		UsuarioSugerido:        informe.UsuarioSugerido,
		SSHAbierto:             informe.SSHAbierto,
		Problemas:              informe.Problemas,
		Sugerencias:            informe.Sugerencias,
	})
}
