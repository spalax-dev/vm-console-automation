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

// MachinesService es el caso de uso del inventario de máquinas
// virtuales: buscar en el hipervisor y vincular una con la consola.
type MachinesService interface {
	// List devuelve las máquinas que coinciden con el filtro.
	List(ctx context.Context, filter string) ([]services.VirtualMachine, error)
	// Bind establece la máquina que administra la consola.
	Bind(ctx context.Context, identifier string) (services.VirtualMachine, error)
	// PowerOn arranca la máquina indicada.
	PowerOn(ctx context.Context, identifier string) (services.VirtualMachine, error)
}

// Machines expone GET /api/vms: el inventario que alimenta el selector
// de la barra superior.
type Machines struct {
	service MachinesService
}

// NewMachines construye el handler del inventario.
func NewMachines(service MachinesService) *Machines {
	return &Machines{service: service}
}

// machineResponse es una máquina del inventario en la respuesta JSON.
type machineResponse struct {
	Name         string `json:"nombre"`
	UUID         string `json:"uuid"`
	State        string `json:"estado"`
	Active       bool   `json:"activa"`
	IsRegistered bool   `json:"registrada"`
}

// machineList es el cuerpo de GET /api/vms.
type machineList struct {
	Machines []machineResponse `json:"maquinas"`
}

// newMachineResponse traduce la máquina del servicio al contrato.
func newMachineResponse(machine services.VirtualMachine) machineResponse {
	return machineResponse{
		Name:         machine.Name,
		UUID:         machine.UUID,
		State:        string(machine.State),
		Active:       machine.Active,
		IsRegistered: machine.IsRegistered,
	}
}

// ServeHTTP implementa http.Handler: solo admite GET.
func (h *Machines) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	machines, err := h.service.List(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		slog.Error("listar las máquinas virtuales", errorKey, err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Message:   "No fue posible consultar las máquinas virtuales del hipervisor.",
			Technical: err.Error(),
		})

		return
	}

	body := machineList{Machines: make([]machineResponse, 0, len(machines))}

	for _, machine := range machines {
		body.Machines = append(body.Machines, newMachineResponse(machine))
	}

	writeJSON(w, http.StatusOK, body)
}

// PowerOn atiende POST /api/vms/{vm}/poweron: arranca la máquina sin
// interfaz gráfica y devuelve su estado actualizado.
func (h *Machines) PowerOn(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	identifier := r.PathValue("vm")

	if strings.TrimSpace(identifier) == "" {
		respondValidacion(w, invalidField("vm", "es obligatorio"))

		return
	}

	machine, err := h.service.PowerOn(r.Context(), identifier)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrMachineNotFound):
			writeJSON(w, http.StatusNotFound, errorResponse{
				Message:   "La máquina indicada no está registrada en el hipervisor.",
				Technical: err.Error(),
			})
		case errors.Is(err, services.ErrMachineRunning):
			writeJSON(w, http.StatusConflict, errorResponse{
				Message:   "La máquina ya está en ejecución.",
				Technical: err.Error(),
			})
		default:
			slog.Error("encender la máquina virtual", errorKey, err)
			writeJSON(w, http.StatusInternalServerError, errorResponse{
				Message:   "No fue posible encender la máquina virtual.",
				Technical: err.Error(),
			})
		}

		return
	}

	writeJSON(w, http.StatusOK, newMachineResponse(machine))
}
