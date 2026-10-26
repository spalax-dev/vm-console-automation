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

// PackagesService ejecuta las operaciones de APT sobre la máquina.
type PackagesService interface {
	// List devuelve los paquetes instalados.
	List(ctx context.Context) ([]services.Package, error)
	// Install instala el paquete indicado.
	Install(ctx context.Context, name string) (services.Outcome, error)
	// Delete elimina el paquete indicado.
	Delete(ctx context.Context, name string) (services.Outcome, error)
}

// KeysService genera y entrega las llaves SSH de root.
type KeysService interface {
	// Generate crea el par, instala la pública y deshabilita el acceso
	// por contraseña.
	Generate(ctx context.Context) (services.Outcome, error)
	// Download devuelve el nombre y el contenido de la llave privada.
	Download() (name string, content []byte, err error)
}

// IssueService lee y reemplaza /etc/issue.
type IssueService interface {
	// Get devuelve la configuración actual del banner.
	Get(ctx context.Context) (services.IssueConfig, error)
	// Write reemplaza el banner con la configuración dada.
	Write(ctx context.Context, config services.IssueConfig) (services.Outcome, error)
}

// HostnameService lee y cambia el nombre de host y /etc/hosts.
type HostnameService interface {
	// Get devuelve el nombre, la IP y el estado de /etc/hosts.
	Get(ctx context.Context) (services.HostnameConfig, error)
	// Write aplica el nombre y, si se pide, actualiza /etc/hosts.
	Write(ctx context.Context, config services.HostnameConfig) (services.Outcome, error)
}

// System agrupa los recursos de preparación del sistema (numeral 3.1
// del enunciado).
type System struct {
	packages PackagesService
	keys     KeysService
	issue    IssueService
	hostname HostnameService
}

// NewSystem construye el handler de preparación del sistema con los
// servicios dados.
func NewSystem(
	packages PackagesService,
	keys KeysService,
	issue IssueService,
	hostname HostnameService,
) *System {
	return &System{packages: packages, keys: keys, issue: issue, hostname: hostname}
}

// packageResponse es un paquete en la respuesta JSON.
type packageResponse struct {
	Name    string `json:"nombre"`
	Version string `json:"version"`
	State   string `json:"estado"`
}

// packageList es el cuerpo de GET /api/packages.
type packageList struct {
	Packages []packageResponse `json:"paquetes"`
}

// packageRequest es el cuerpo de POST /api/packages.
type packageRequest struct {
	Name string `json:"nombre"`
}

// issueRequest es el cuerpo de PUT /api/issue.
type issueRequest struct {
	Message  string `json:"mensaje"`
	NameHost bool   `json:"nombre_host"`
	Date     bool   `json:"fecha"`
	Hour     bool   `json:"hora"`
	IP       bool   `json:"ip"`
}

// hostnameRequest es el cuerpo de PUT /api/hostname.
type hostnameRequest struct {
	Name        string `json:"nombre"`
	IP          string `json:"ip"`
	UpdateHosts bool   `json:"actualizar_hosts"`
}

// Packages atiende GET y POST /api/packages: lista los paquetes
// instalados o instala uno nuevo.
func (h *System) Packages(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet, http.MethodPost) {
		return
	}

	if r.Method == http.MethodGet {
		h.listPackages(w, r)

		return
	}

	var request packageRequest
	if err := decodeJSON(w, r, &request); err != nil {
		respondValidacion(w, err)

		return
	}

	if err := validatePkg(nameField, request.Name); err != nil {
		respondValidacion(w, err)

		return
	}

	result, err := h.packages.Install(r.Context(), request.Name)
	if err != nil {
		respondErrorOperation(w, result, err)

		return
	}

	w.Header().Set(headerLocation, "/api/packages/"+request.Name)
	writeResult(w, http.StatusCreated, result)
}

// listPackages responde la colección de paquetes instalados.
func (h *System) listPackages(w http.ResponseWriter, r *http.Request) {
	packages, err := h.packages.List(r.Context())
	if err != nil {
		respondErrorOperation(w, services.Outcome{}, err)

		return
	}

	response := packageList{Packages: make([]packageResponse, 0, len(packages))}
	for _, pkg := range packages {
		response.Packages = append(response.Packages, packageResponse{
			Name:    pkg.Name,
			Version: pkg.Version,
			State:   pkg.State,
		})
	}

	writeJSON(w, http.StatusOK, response)
}

// Package atiende DELETE /api/packages/{name}: elimina el paquete
// indicado en la ruta.
func (h *System) Package(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodDelete) {
		return
	}

	name := r.PathValue(nameParam)
	if err := validatePkg(nameParam, name); err != nil {
		respondValidacion(w, err)

		return
	}

	result, err := h.packages.Delete(r.Context(), name)
	if err != nil {
		respondErrorOperation(w, result, err)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RootKey atiende GET y POST /api/keys/root: descarga la llave
// privada de root o genera un par nuevo.
func (h *System) RootKey(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet, http.MethodPost) {
		return
	}

	if r.Method == http.MethodGet {
		h.downloadSSHKeyRoot(w)

		return
	}

	result, err := h.keys.Generate(r.Context())
	if errors.Is(err, services.ErrKeyExists) {
		writeJSON(w, http.StatusConflict,
			responseFromError(result, "La llave SSH de root ya existe en el anfitrión.", err))

		return
	}

	if err != nil {
		respondErrorOperation(w, result, err)

		return
	}

	w.Header().Set(headerLocation, "/api/keys/root")
	writeResult(w, http.StatusCreated, result)
}

// downloadSSHKeyRoot entrega la llave privada como archivo adjunto.
func (h *System) downloadSSHKeyRoot(w http.ResponseWriter) {
	name, content, err := h.keys.Download()
	if errors.Is(err, services.ErrKeyNotFound) {
		writeJSON(w, http.StatusNotFound, errorResponse{
			Message: "No hay una llave SSH de root generada.",
		})

		return
	}

	if err != nil {
		slog.Error("descargar la llave de root", errorKey, err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Message:   "No fue posible leer la llave privada.",
			Technical: err.Error(),
		})

		return
	}

	w.Header().Set(headerContentType, "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)

	// #nosec G705 -- es la descarga binaria de una llave, no HTML.
	if _, err := w.Write(content); err != nil {
		slog.Error("escribir la llave de root", errorKey, err)
	}
}

// configIssue es el cuerpo de GET /api/issue.
type configIssue struct {
	Message  string `json:"mensaje"`
	NameHost bool   `json:"nombre_host"`
	Date     bool   `json:"fecha"`
	Hour     bool   `json:"hora"`
	IP       bool   `json:"ip"`
	Content  string `json:"contenido,omitempty"`
}

// Issue atiende GET y PUT /api/issue: consulta o reemplaza el banner
// previo al inicio de sesión.
func (h *System) Issue(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet, http.MethodPut) {
		return
	}

	if r.Method == http.MethodGet {
		h.getIssue(w, r)

		return
	}

	var request issueRequest
	if err := decodeJSON(w, r, &request); err != nil {
		respondValidacion(w, err)

		return
	}

	if err := validateMessageIssue(request.Message); err != nil {
		respondValidacion(w, err)

		return
	}

	if !hasIssueOptions(request) {
		respondValidacion(w, invalidField(messageField, "indique un mensaje o al menos una opción"))

		return
	}

	result, err := h.issue.Write(r.Context(), services.IssueConfig{
		Message:  request.Message,
		NameHost: request.NameHost,
		Date:     request.Date,
		Hour:     request.Hour,
		IP:       request.IP,
	})
	if err != nil {
		respondErrorOperation(w, result, err)

		return
	}

	writeResult(w, http.StatusOK, result)
}

// hasIssueOptions informa si la solicitud incluye algún elemento.
func hasIssueOptions(request issueRequest) bool {
	return strings.TrimSpace(request.Message) != "" ||
		request.NameHost || request.Date || request.Hour || request.IP
}

// getIssue responde la configuración actual de /etc/issue.
func (h *System) getIssue(w http.ResponseWriter, r *http.Request) {
	config, err := h.issue.Get(r.Context())
	if err != nil {
		respondErrorOperation(w, services.Outcome{}, err)

		return
	}

	writeJSON(w, http.StatusOK, configIssue{
		Message:  config.Message,
		NameHost: config.NameHost,
		Date:     config.Date,
		Hour:     config.Hour,
		IP:       config.IP,
		Content:  config.Content,
	})
}

// configHostname es el cuerpo de GET /api/hostname.
type configHostname struct {
	Name        string `json:"nombre"`
	IP          string `json:"ip"`
	UpdateHosts bool   `json:"actualizar_hosts"`
}

// Hostname atiende GET y PUT /api/hostname: consulta o reemplaza el
// nombre de host y su entrada en /etc/hosts.
func (h *System) Hostname(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet, http.MethodPut) {
		return
	}

	if r.Method == http.MethodGet {
		h.getHostname(w, r)

		return
	}

	var request hostnameRequest
	if err := decodeJSON(w, r, &request); err != nil {
		respondValidacion(w, err)

		return
	}

	if err := validateHostname(request.Name); err != nil {
		respondValidacion(w, err)

		return
	}

	if err := validateIPv4(ipField, request.IP); err != nil {
		respondValidacion(w, err)

		return
	}

	result, err := h.hostname.Write(r.Context(), services.HostnameConfig{
		Name:        request.Name,
		IP:          request.IP,
		UpdateHosts: request.UpdateHosts,
	})
	if err != nil {
		respondErrorOperation(w, result, err)

		return
	}

	writeResult(w, http.StatusOK, result)
}

// getHostname responde el nombre, la IP y el estado de /etc/hosts.
func (h *System) getHostname(w http.ResponseWriter, r *http.Request) {
	config, err := h.hostname.Get(r.Context())
	if err != nil {
		respondErrorOperation(w, services.Outcome{}, err)

		return
	}

	writeJSON(w, http.StatusOK, configHostname{
		Name:        config.Name,
		IP:          config.IP,
		UpdateHosts: config.UpdateHosts,
	})
}
