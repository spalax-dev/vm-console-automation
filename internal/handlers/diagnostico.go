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

// DiagnosticsService explica por qué una máquina no se puede registrar.
type DiagnosticsService interface {
	Get(ctx context.Context) (services.Diagnostics, error)
}

// Diagnostics expone GET /api/session/diagnostico.
type Diagnostics struct {
	service DiagnosticsService
}

// NewDiagnostics construye el handler del diagnóstico.
func NewDiagnostics(service DiagnosticsService) *Diagnostics {
	return &Diagnostics{service: service}
}

// diagnosticsResponse es el cuerpo JSON del diagnóstico.
type diagnosticsResponse struct {
	VM                     string   `json:"vm"`
	State                  string   `json:"estado"`
	Network                string   `json:"red"`
	IP                     string   `json:"ip,omitempty"`
	IsRegistered           bool     `json:"registrada"`
	GuestAdditionsVersion  string   `json:"guest_additions_version,omitempty"`
	GuestAdditionsRunLevel int      `json:"guest_additions_run_level"`
	UserSugerido           string   `json:"usuario_sugerido,omitempty"`
	SSHOpen                bool     `json:"ssh_abierto"`
	Problemas              []string `json:"problemas"`
	Sugerencias            []string `json:"sugerencias"`
}

// ServeHTTP implementa http.Handler.
func (h *Diagnostics) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	report, err := h.service.Get(r.Context())
	if errors.Is(err, services.ErrNoMachine) {
		writeJSON(w, http.StatusConflict, errorResponse{
			Message:   "Primero elige una máquina virtual.",
			Technical: err.Error(),
		})

		return
	}

	if err != nil {
		slog.Error("diagnosticar la máquina", errorKey, err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Message:   "No fue posible diagnosticar la máquina.",
			Technical: err.Error(),
		})

		return
	}

	ip := ""
	if report.IP.IsValid() {
		ip = report.IP.String()
	}

	writeJSON(w, http.StatusOK, diagnosticsResponse{
		VM:                     report.VM,
		State:                  string(report.State),
		Network:                string(report.Network),
		IP:                     ip,
		IsRegistered:           report.IsRegistered,
		GuestAdditionsVersion:  report.GuestAdditionsVersion,
		GuestAdditionsRunLevel: report.GuestAdditionsRunLevel,
		UserSugerido:           report.UserSugerido,
		SSHOpen:                report.SSHOpen,
		Problemas:              report.Problemas,
		Sugerencias:            report.Sugerencias,
	})
}
