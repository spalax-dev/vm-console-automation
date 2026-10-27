// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"consola-vm/internal/services"
)

// HAProxyService ejecuta las operaciones sobre el back-end de HAProxy.
type HAProxyService interface {
	// List devuelve los servidores del back-end.
	List(ctx context.Context) ([]services.Server, error)
	// Add agrega un servidor al back-end.
	Add(ctx context.Context, address string, port int) (services.Outcome, error)
	// Delete elimina un servidor del back-end.
	Delete(ctx context.Context, address string, port int) (services.Outcome, error)
	// Status informa el estado de HAProxy.
	Status(ctx context.Context) (services.HAProxyStatus, error)
	// Setup instala o repara la configuración mínima de HAProxy.
	Setup(ctx context.Context) (services.Outcome, error)
}

// HAProxy agrupa los recursos de balanceo de carga con HAProxy
// (numeral 3.4).
type HAProxy struct {
	service HAProxyService
}

// NewHAProxy construye el handler de balanceo con HAProxy.
func NewHAProxy(service HAProxyService) *HAProxy {
	return &HAProxy{service: service}
}

// serverBackend es el cuerpo de POST /api/haproxy/backend/servers.
type serverBackend struct {
	Address string `json:"direccion"`
	Port    int    `json:"puerto"`
}

// serverResponse es un servidor en la respuesta JSON.
type serverResponse struct {
	Address         string `json:"direccion"`
	Port            int    `json:"puerto"`
	State           string `json:"estado"`
	Weight          int    `json:"peso"`
	Active          bool   `json:"activo"`
	Backup          bool   `json:"respaldo"`
	CurrentSessions int    `json:"sesiones"`
	MaxSessions     int    `json:"sesiones_max"`
	TotalSessions   int64  `json:"sesiones_total"`
	QueueCurrent    int    `json:"cola"`
	QueueMax        int    `json:"cola_max"`
	BytesIn         int64  `json:"bytes_entrada"`
	BytesOut        int64  `json:"bytes_salida"`
	CheckStatus     string `json:"check,omitempty"`
	CheckCode       int    `json:"check_codigo,omitempty"`
	CheckDuration   int    `json:"check_duracion"`
	Downtime        int64  `json:"downtime"`
	LastChange      int64  `json:"ultimo_cambio"`
	LastSession     int64  `json:"ultima_sesion"`
	LastCheck       string `json:"ultimo_check,omitempty"`
}

// backendList es el cuerpo de GET /api/haproxy/backend/servers.
type backendList struct {
	Servers []serverResponse `json:"servidores"`
}

// statusResponse es el cuerpo de GET /api/haproxy/status.
type statusResponse struct {
	Installed     bool     `json:"instalado"`
	ConfigExists  bool     `json:"configuracion"`
	ConfigValid   bool     `json:"valida"`
	Backend       bool     `json:"backend"`
	ServiceActive bool     `json:"servicio_activo"`
	Socat         bool     `json:"socat"`
	Port          int      `json:"puerto,omitempty"`
	Problems      []string `json:"problemas"`
	Suggestions   []string `json:"sugerencias"`
}

// Status atiende GET /api/haproxy/status.
func (h *HAProxy) Status(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	status, err := h.service.Status(r.Context())
	if err != nil {
		respondErrorOperation(w, services.Outcome{}, err)

		return
	}

	writeJSON(w, http.StatusOK, statusResponse{
		Installed:     status.Installed,
		ConfigExists:  status.ConfigExists,
		ConfigValid:   status.ConfigValid,
		Backend:       status.Backend,
		ServiceActive: status.ServiceActive,
		Socat:         status.Socat,
		Port:          status.Port,
		Problems:      status.Problems,
		Suggestions:   status.Suggestions,
	})
}

// Setup atiende POST /api/haproxy/setup: instala o repara HAProxy.
func (h *HAProxy) Setup(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	result, err := h.service.Setup(r.Context())
	if err != nil {
		respondErrorOperation(w, result, err)

		return
	}

	writeResult(w, http.StatusOK, result)
}

// Servers atiende GET y POST /api/haproxy/backend/servers: lista o
// agrega servidores al back-end.
func (h *HAProxy) Servers(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet, http.MethodPost) {
		return
	}

	if r.Method == http.MethodGet {
		h.listBackend(w, r)

		return
	}

	var request serverBackend
	if err := decodeJSON(w, r, &request); err != nil {
		respondValidacion(w, err)

		return
	}

	if err := validateIPv4(addressField, request.Address); err != nil {
		respondValidacion(w, err)

		return
	}

	if err := validatePort(request.Port); err != nil {
		respondValidacion(w, err)

		return
	}

	result, err := h.service.Add(r.Context(), request.Address, request.Port)
	if errors.Is(err, services.ErrServerExists) {
		writeJSON(w, http.StatusConflict,
			responseFromError(result, "El servidor ya está en el back-end.", err))

		return
	}

	if err != nil {
		respondErrorOperation(w, result, err)

		return
	}

	w.Header().Set(headerLocation, "/api/haproxy/backend/servers/"+request.Address+":"+strconv.Itoa(request.Port))
	writeResult(w, http.StatusCreated, result)
}

// listBackend responde la colección de servidores.
func (h *HAProxy) listBackend(w http.ResponseWriter, r *http.Request) {
	servers, err := h.service.List(r.Context())
	if err != nil {
		respondErrorOperation(w, services.Outcome{}, err)

		return
	}

	response := backendList{Servers: make([]serverResponse, 0, len(servers))}
	for i := range servers {
		server := &servers[i]
		response.Servers = append(response.Servers, serverResponse{
			Address:         server.Address,
			Port:            server.Port,
			State:           server.State,
			Weight:          server.Stats.Weight,
			Active:          server.Stats.Active,
			Backup:          server.Stats.Backup,
			CurrentSessions: server.Stats.CurrentSessions,
			MaxSessions:     server.Stats.MaxSessions,
			TotalSessions:   server.Stats.TotalSessions,
			QueueCurrent:    server.Stats.QueueCurrent,
			QueueMax:        server.Stats.QueueMax,
			BytesIn:         server.Stats.BytesIn,
			BytesOut:        server.Stats.BytesOut,
			CheckStatus:     server.Stats.CheckStatus,
			CheckCode:       server.Stats.CheckCode,
			CheckDuration:   server.Stats.CheckDuration,
			Downtime:        server.Stats.Downtime,
			LastChange:      server.Stats.LastChange,
			LastSession:     server.Stats.LastSession,
			LastCheck:       server.Stats.LastCheck,
		})
	}

	writeJSON(w, http.StatusOK, response)
}

// Server atiende DELETE /api/haproxy/backend/servers/{server}: elimina
// el servidor identificado como «direccion:puerto».
func (h *HAProxy) Server(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodDelete) {
		return
	}

	server := r.PathValue(serverParam)

	destination, err := validateServer(serverParam, server)
	if err != nil {
		respondValidacion(w, err)

		return
	}

	result, err := h.service.Delete(r.Context(), destination.Addr().String(), int(destination.Port()))
	if errors.Is(err, services.ErrServerNotFound) {
		writeJSON(w, http.StatusNotFound,
			responseFromError(result, "El servidor no está en el back-end.", err))

		return
	}

	if err != nil {
		respondErrorOperation(w, result, err)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}
