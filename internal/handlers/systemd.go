// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"context"
	"errors"
	"net/http"

	"consola-vm/internal/services"
)

// SystemdService ejecuta las operaciones sobre las unidades de systemd.
type SystemdService interface {
	// List devuelve las unidades con su estado.
	List(ctx context.Context) ([]services.UnitState, error)
	// Get devuelve el estado de una unidad.
	Get(ctx context.Context, unit string) (services.UnitState, error)
	// Deploy transfiere los archivos de la unidad.
	Deploy(ctx context.Context, unit string, files []services.DeploymentFile, user, group string) (services.Outcome, error)
	// ChangeState activa, desactiva, habilita o deshabilita la unidad.
	ChangeState(ctx context.Context, unit string, active, enabled *bool) (services.Outcome, error)
	// Restart reinicia la unidad.
	Restart(ctx context.Context, unit string) (services.Outcome, error)
}

// Systemd agrupa los recursos de gestión de unidades con systemd
// (numeral 3.3).
type Systemd struct {
	service SystemdService
}

// NewSystemd construye el handler de gestión de unidades systemd.
func NewSystemd(service SystemdService) *Systemd {
	return &Systemd{service: service}
}

// fileDespliegue describe un archivo que se transfiere a la máquina.
type fileDespliegue struct {
	Source      string `json:"origen"`
	Destination string `json:"destino"`
	Mode        string `json:"modo"`
}

// deployRequest es el cuerpo de PUT /api/units/{unit}.
type deployRequest struct {
	Files []fileDespliegue `json:"archivos"`
	User  string           `json:"usuario"`
	Group string           `json:"grupo"`
}

// unitStateRequest es el cuerpo de PATCH /api/units/{unit}: indica
// exactamente una de las dos propiedades.
type unitStateRequest struct {
	Active  *bool `json:"activo"`
	Enabled *bool `json:"habilitado"`
}

// unitRow es una unidad en la respuesta JSON.
type unitRow struct {
	Unit    string `json:"unidad"`
	Active  bool   `json:"activo"`
	Enabled bool   `json:"habilitado"`
	State   string `json:"estado"`
	Detail  string `json:"detalle,omitempty"`
}

// unitList es el cuerpo de GET /api/units.
type unitList struct {
	Units []unitRow `json:"unidades"`
}

// Units atiende GET /api/units: lista las unidades administradas.
func (h *Systemd) Units(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	units, err := h.service.List(r.Context())
	if err != nil {
		respondErrorOperation(w, services.Outcome{}, err)

		return
	}

	response := unitList{Units: make([]unitRow, 0, len(units))}
	for _, unit := range units {
		response.Units = append(response.Units, unitResponse(unit))
	}

	writeJSON(w, http.StatusOK, response)
}

// Unit atiende GET, PUT y PATCH /api/units/{unit}: consulta la
// unidad, despliega sus archivos o cambia su estado.
func (h *Systemd) Unit(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet, http.MethodPut, http.MethodPatch) {
		return
	}

	unit := r.PathValue(unitParam)
	if err := validateUnit(unit); err != nil {
		respondValidacion(w, err)

		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getUnit(w, r, unit)
	case http.MethodPut:
		h.deployUnit(w, r, unit)
	case http.MethodPatch:
		h.changeStateUnit(w, r, unit)
	}
}

// getUnit responde el estado de la unidad.
func (h *Systemd) getUnit(w http.ResponseWriter, r *http.Request, unit string) {
	state, err := h.service.Get(r.Context(), unit)
	if h.respondNotFound(w, unit, err) {
		return
	}

	if err != nil {
		respondErrorOperation(w, services.Outcome{}, err)

		return
	}

	writeJSON(w, http.StatusOK, unitResponse(state))
}

// deployUnit valida y transfiere la unidad y sus archivos.
func (h *Systemd) deployUnit(w http.ResponseWriter, r *http.Request, unit string) {
	var request deployRequest
	if err := decodeJSON(w, r, &request); err != nil {
		respondValidacion(w, err)

		return
	}

	if err := validateFiles(request.Files); err != nil {
		respondValidacion(w, err)

		return
	}

	if request.User != "" {
		if err := validateUser("usuario", request.User); err != nil {
			respondValidacion(w, err)

			return
		}
	}

	if request.Group != "" {
		if err := validateUser("grupo", request.Group); err != nil {
			respondValidacion(w, err)

			return
		}
	}

	files := make([]services.DeploymentFile, 0, len(request.Files))
	for _, file := range request.Files {
		files = append(files, services.DeploymentFile{
			Source:      file.Source,
			Destination: file.Destination,
			Mode:        file.Mode,
		})
	}

	result, err := h.service.Deploy(r.Context(), unit, files, request.User, request.Group)
	if err != nil {
		respondErrorOperation(w, result, err)

		return
	}

	writeResult(w, http.StatusOK, result)
}

// changeStateUnit aplica el cambio y responde el estado resultante.
func (h *Systemd) changeStateUnit(w http.ResponseWriter, r *http.Request, unit string) {
	var request unitStateRequest
	if err := decodeJSON(w, r, &request); err != nil {
		respondValidacion(w, err)

		return
	}

	if err := validateStateUnit(request); err != nil {
		respondValidacion(w, err)

		return
	}

	result, err := h.service.ChangeState(r.Context(), unit, request.Active, request.Enabled)
	if h.respondNotFound(w, unit, err) {
		return
	}

	if err != nil {
		respondErrorOperation(w, result, err)

		return
	}

	state, err := h.service.Get(r.Context(), unit)
	if err != nil {
		respondErrorOperation(w, services.Outcome{}, err)

		return
	}

	writeJSON(w, http.StatusOK, unitResponse(state))
}

// validateStateUnit exige exactamente una de las dos propiedades.
func validateStateUnit(request unitStateRequest) error {
	if request.Active == nil && request.Enabled == nil {
		return invalidField(activeField, "debe indicar «activo» o «habilitado»")
	}

	if request.Active != nil && request.Enabled != nil {
		return invalidField(enabledField, "indique solo «activo» o «habilitado»")
	}

	return nil
}

// validateFiles comprueba que haya al menos un archivo y que cada uno
// tenga origen, destino absoluto y modo octal.
func validateFiles(files []fileDespliegue) error {
	if len(files) == 0 {
		return invalidField(filesField, "debe indicar al menos un archivo")
	}

	for _, file := range files {
		if file.Source == "" {
			return invalidField(sourceField, "no puede estar vacío")
		}

		if err := validatePath(destinationField, file.Destination); err != nil {
			return err
		}

		if err := validateMode(modeField, file.Mode); err != nil {
			return err
		}
	}

	return nil
}

// Restarts atiende POST /api/units/{unit}/restarts: reinicia la
// unidad indicada.
func (h *Systemd) Restarts(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	unit := r.PathValue(unitParam)
	if err := validateUnit(unit); err != nil {
		respondValidacion(w, err)

		return
	}

	result, err := h.service.Restart(r.Context(), unit)
	if h.respondNotFound(w, unit, err) {
		return
	}

	if err != nil {
		respondErrorOperation(w, result, err)

		return
	}

	writeResult(w, http.StatusAccepted, result)
}

// respondNotFound responde 404 si el error indica que la unidad no
// existe y devuelve true en ese caso.
func (h *Systemd) respondNotFound(w http.ResponseWriter, unit string, err error) bool {
	if !errors.Is(err, services.ErrUnitNotFound) {
		return false
	}

	writeJSON(w, http.StatusNotFound, errorResponse{
		Message:   "La unidad «" + unit + "» no existe.",
		Technical: err.Error(),
	})

	return true
}

// unitResponse convierte el estado del dominio en la respuesta.
func unitResponse(state services.UnitState) unitRow {
	return unitRow{
		Unit:    state.Unit,
		Active:  state.Active,
		Enabled: state.Enabled,
		State:   state.State,
		Detail:  state.Detail,
	}
}
