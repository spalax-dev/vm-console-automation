// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"consola-vm/internal/services"
)

// errUnit simula un fallo inesperado del servicio de systemd.
var errUnit = errors.New("fallo de systemd")

// stubSystemd es un servicio de systemd falso con resultados
// preprogramados.
type stubSystemd struct {
	lists    []services.UnitState
	state    services.UnitState
	result   services.Outcome
	err      error
	getErr   error
	unit     string
	files    []services.DeploymentFile
	user     string
	group    string
	active   *bool
	enabled  *bool
	restarts int
}

// List devuelve la lista preprogramada.
func (s *stubSystemd) List(context.Context) ([]services.UnitState, error) {
	return s.lists, s.err
}

// Get devuelve el estado preprogramado.
func (s *stubSystemd) Get(_ context.Context, unit string) (services.UnitState, error) {
	s.unit = unit

	return s.state, s.getErr
}

// Deploy registra los argumentos y devuelve el resultado.
func (s *stubSystemd) Deploy(
	_ context.Context, unit string, files []services.DeploymentFile, user, group string,
) (services.Outcome, error) {
	s.unit = unit
	s.files = files
	s.user = user
	s.group = group

	return s.result, s.err
}

// ChangeState registra los argumentos y devuelve el resultado.
func (s *stubSystemd) ChangeState(
	_ context.Context, unit string, active, enabled *bool,
) (services.Outcome, error) {
	s.unit = unit
	s.active = active
	s.enabled = enabled

	return s.result, s.err
}

// Restart registra la llamada y devuelve el resultado.
func (s *stubSystemd) Restart(_ context.Context, unit string) (services.Outcome, error) {
	s.unit = unit
	s.restarts++

	return s.result, s.err
}

// TestUnitsList cubre el listado de unidades.
func TestUnitsList(t *testing.T) {
	t.Parallel()

	service := &stubSystemd{lists: []services.UnitState{
		{Unit: unitCatalogo, Active: true, State: stateActiveTests, Detail: "running"},
	}}

	rec := runTestCase(t, NewSystemd(service).Units, "/api/units", testCaseRequest{
		name: testCaseList, method: http.MethodGet,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf(msgCode, rec.Code, http.StatusOK)
	}

	var body unitList
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf(msgJSONInvalid, err)
	}

	if len(body.Units) != 1 || !body.Units[0].Active {
		t.Errorf("unidades = %+v, want la unidad activa", body.Units)
	}
}

// TestUnitGet cubre la consulta del estado de una unidad.
func TestUnitGet(t *testing.T) {
	t.Parallel()

	service := &stubSystemd{state: services.UnitState{
		Unit: unitCatalogo, Active: true, Enabled: true, State: stateActiveTests, Detail: "running",
	}}

	rec := runTestCaseWithPath(t, NewSystemd(service).Unit, pathUnit, unitParam, unitCatalogo, testCaseRequest{
		name: testCaseGet, method: http.MethodGet,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf(msgCode, rec.Code, http.StatusOK)
	}

	var body unitRow
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf(msgJSONInvalid, err)
	}

	if !body.Active || !body.Enabled {
		t.Errorf("unidad = %+v, want activa y habilitada", body)
	}
}

// TestUnitNotFound cubre la consulta de una unidad inexistente.
func TestUnitNotFound(t *testing.T) {
	t.Parallel()

	service := &stubSystemd{getErr: services.ErrUnitNotFound}

	rec := runTestCaseWithPath(t, NewSystemd(service).Unit, pathUnit, unitParam, unitCatalogo, testCaseRequest{
		name: testCaseInexistente, method: http.MethodGet,
	})

	if rec.Code != http.StatusNotFound {
		t.Fatalf(msgCode, rec.Code, http.StatusNotFound)
	}
}

// TestUnitError cubre un fallo inesperado al consultar.
func TestUnitError(t *testing.T) {
	t.Parallel()

	service := &stubSystemd{getErr: errUnit}

	rec := runTestCaseWithPath(t, NewSystemd(service).Unit, pathUnit, unitParam, unitCatalogo, testCaseRequest{
		name: "error", method: http.MethodGet,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCode, rec.Code, http.StatusInternalServerError)
	}
}

// TestUnitDeploy cubre la transferencia de archivos.
func TestUnitDeploy(t *testing.T) {
	t.Parallel()

	service := &stubSystemd{result: services.Outcome{Message: "Unidad desplegada."}}

	body := `{"archivos":[{"origen":"./app/catalogo","destino":"/opt/catalogo/catalogo","modo":"755"}],"usuario":"catalogo","grupo":"catalogo"}`

	rec := runTestCaseWithPath(t, NewSystemd(service).Unit, pathUnit, unitParam, unitCatalogo, testCaseRequest{
		name: "desplegar", method: http.MethodPut, body: body,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf(msgCode, rec.Code, http.StatusOK)
	}

	if len(service.files) != 1 || service.files[0].Destination != "/opt/catalogo/catalogo" {
		t.Errorf("archivos = %+v, want el archivo transferido", service.files)
	}

	if service.user != propietarioCatalogo || service.group != propietarioCatalogo {
		t.Errorf("propietario = %s:%s, want catalogo:catalogo", service.user, service.group)
	}
}

// TestUnitChangeState cubre la activación de una unidad.
func TestUnitChangeState(t *testing.T) {
	t.Parallel()

	service := &stubSystemd{state: services.UnitState{
		Unit: unitCatalogo, Active: true, State: stateActiveTests,
	}}

	rec := runTestCaseWithPath(t, NewSystemd(service).Unit, pathUnit, unitParam, unitCatalogo, testCaseRequest{
		name: "activar", method: http.MethodPatch, body: bodyActivar,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf(msgCode, rec.Code, http.StatusOK)
	}

	if service.active == nil || !*service.active {
		t.Errorf("activo = %v, want true", service.active)
	}

	var body unitRow
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf(msgJSONInvalid, err)
	}

	if !body.Active {
		t.Errorf("estado = %+v, want activa", body)
	}
}

// TestUnitValidacion cubre las reglas de unidad, archivos y estado.
func TestUnitValidacion(t *testing.T) {
	t.Parallel()

	handler := NewSystemd(&stubSystemd{}).Unit

	testCases := []struct {
		testCaseRequest
		unit string
	}{
		{testCaseRequest{name: testCaseUnitInvalida, method: http.MethodGet, code: http.StatusBadRequest, field: unitParam}, "../etc/passwd"},
		{
			testCaseRequest{
				name:   "sin archivos",
				method: http.MethodPut,
				body:   `{"archivos":[]}`,
				code:   http.StatusBadRequest,
				field:  filesField,
			},
			unitCatalogo,
		},
		{
			testCaseRequest{
				name:   "destino relativo",
				method: http.MethodPut,
				body:   `{"archivos":[{"origen":"./app","destino":"opt/app","modo":"755"}]}`,
				code:   http.StatusBadRequest,
				field:  destinationField,
			},
			unitCatalogo,
		},
		{
			testCaseRequest{
				name:   "modo inválido",
				method: http.MethodPut,
				body:   `{"archivos":[{"origen":"./app","destino":"/opt/app","modo":"rwx"}]}`,
				code:   http.StatusBadRequest,
				field:  modeField,
			},
			unitCatalogo,
		},
		{
			testCaseRequest{
				name:   testCaseUserInvalid,
				method: http.MethodPut,
				body:   `{"archivos":[{"origen":"./app","destino":"/opt/app","modo":"755"}],"usuario":"José"}`,
				code:   http.StatusBadRequest,
				field:  "usuario",
			},
			unitCatalogo,
		},
		{
			testCaseRequest{
				name:   "sin estado",
				method: http.MethodPatch,
				body:   `{}`,
				code:   http.StatusBadRequest,
				field:  activeField,
			},
			unitCatalogo,
		},
		{
			testCaseRequest{
				name:   "dos estados",
				method: http.MethodPatch,
				body:   `{"activo":true,"habilitado":true}`,
				code:   http.StatusBadRequest,
				field:  enabledField,
			},
			unitCatalogo,
		},
		{testCaseRequest{name: testCaseBodyInvalid, method: http.MethodPut, body: `{`, code: http.StatusBadRequest}, unitCatalogo},
		{testCaseRequest{name: testCaseMethodNotPermitido, method: http.MethodDelete, code: http.StatusMethodNotAllowed}, unitCatalogo},
	}

	for _, testCase := range testCases {
		rec := runTestCaseWithPath(t, handler, pathUnit, unitParam, testCase.unit, testCase.testCaseRequest)
		checkTestCase(t, rec, testCase.testCaseRequest)
	}
}

// TestRestarts cubre el reinicio de una unidad.
func TestRestarts(t *testing.T) {
	t.Parallel()

	service := &stubSystemd{result: services.Outcome{Message: "Unidad reiniciada."}}

	rec := runTestCaseWithPath(t, NewSystemd(service).Restarts, pathRestarts, unitParam, unitCatalogo, testCaseRequest{
		name: "reiniciar", method: http.MethodPost,
	})

	if rec.Code != http.StatusAccepted {
		t.Fatalf(msgCode, rec.Code, http.StatusAccepted)
	}

	if service.restarts != 1 {
		t.Errorf("Reiniciar() se llamó %d veces, want 1", service.restarts)
	}
}

// TestRestartsNotFound cubre el reinicio de una unidad inexistente.
func TestRestartsNotFound(t *testing.T) {
	t.Parallel()

	service := &stubSystemd{err: services.ErrUnitNotFound}

	rec := runTestCaseWithPath(t, NewSystemd(service).Restarts, pathRestarts, unitParam, unitCatalogo, testCaseRequest{
		name: testCaseInexistente, method: http.MethodPost,
	})

	if rec.Code != http.StatusNotFound {
		t.Fatalf(msgCode, rec.Code, http.StatusNotFound)
	}
}

// TestUnitsListError cubre el fallo al listar unidades.
func TestUnitsListError(t *testing.T) {
	t.Parallel()

	rec := runTestCase(t, NewSystemd(&stubSystemd{err: errUnit}).Units, "/api/units", testCaseRequest{
		name: testCaseList, method: http.MethodGet,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCode, rec.Code, http.StatusInternalServerError)
	}
}

// TestUnitDeployError cubre el fallo al transferir los archivos.
func TestUnitDeployError(t *testing.T) {
	t.Parallel()

	service := &stubSystemd{err: errUnit}
	body := `{"archivos":[{"origen":"./app","destino":"/opt/app","modo":"755"}]}`

	rec := runTestCaseWithPath(t, NewSystemd(service).Unit, pathUnit, unitParam, unitCatalogo, testCaseRequest{
		name: "fallo de despliegue", method: http.MethodPut, body: body,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCode, rec.Code, http.StatusInternalServerError)
	}
}

// TestUnitChangeStateError cubre el fallo de systemctl al cambiar el
// estado.
func TestUnitChangeStateError(t *testing.T) {
	t.Parallel()

	service := &stubSystemd{err: errUnit}

	rec := runTestCaseWithPath(t, NewSystemd(service).Unit, pathUnit, unitParam, unitCatalogo, testCaseRequest{
		name: "fallo de estado", method: http.MethodPatch, body: bodyActivar,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCode, rec.Code, http.StatusInternalServerError)
	}
}

// TestUnitChangeStateWithoutConsulta cubre el fallo al releer el estado.
func TestUnitChangeStateWithoutConsulta(t *testing.T) {
	t.Parallel()

	service := &stubSystemd{getErr: errUnit}

	rec := runTestCaseWithPath(t, NewSystemd(service).Unit, pathUnit, unitParam, unitCatalogo, testCaseRequest{
		name: "fallo de consulta", method: http.MethodPatch, body: bodyActivar,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCode, rec.Code, http.StatusInternalServerError)
	}
}

// TestRestartsError cubre el fallo al reiniciar una unidad.
func TestRestartsError(t *testing.T) {
	t.Parallel()

	rec := runTestCaseWithPath(t, NewSystemd(&stubSystemd{err: errUnit}).Restarts, pathRestarts, unitParam, unitCatalogo, testCaseRequest{
		name: "fallo de reinicio", method: http.MethodPost,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCode, rec.Code, http.StatusInternalServerError)
	}
}

// TestUnitChangeStateNotFound cubre el cambio de estado de una
// unidad inexistente.
func TestUnitChangeStateNotFound(t *testing.T) {
	t.Parallel()

	service := &stubSystemd{err: services.ErrUnitNotFound}

	rec := runTestCaseWithPath(t, NewSystemd(service).Unit, pathUnit, unitParam, unitCatalogo, testCaseRequest{
		name: "inexistente", method: http.MethodPatch, body: bodyActivar,
	})

	if rec.Code != http.StatusNotFound {
		t.Fatalf(msgCode, rec.Code, http.StatusNotFound)
	}
}

// TestUnitDeployGroupInvalid cubre el grupo no válido.
func TestUnitDeployGroupInvalid(t *testing.T) {
	t.Parallel()

	body := `{"archivos":[{"origen":"./app","destino":"/opt/app","modo":"755"}],"grupo":"José"}`

	rec := runTestCaseWithPath(t, NewSystemd(&stubSystemd{}).Unit, pathUnit, unitParam, unitCatalogo, testCaseRequest{
		name: "grupo inválido", method: http.MethodPut, body: body,
	})

	checkTestCase(t, rec, testCaseRequest{name: "grupo inválido", code: http.StatusBadRequest, field: "grupo"})
}

// TestUnitDeployWithoutSource cubre el archivo sin ruta de origen.
func TestUnitDeployWithoutSource(t *testing.T) {
	t.Parallel()

	body := `{"archivos":[{"origen":"","destino":"/opt/app","modo":"755"}]}`

	rec := runTestCaseWithPath(t, NewSystemd(&stubSystemd{}).Unit, pathUnit, unitParam, unitCatalogo, testCaseRequest{
		name: "sin origen", method: http.MethodPut, body: body,
	})

	checkTestCase(t, rec, testCaseRequest{name: "sin origen", code: http.StatusBadRequest, field: sourceField})
}

// TestRestartsValidacion cubre el método y la unidad inválidos.
func TestRestartsValidacion(t *testing.T) {
	t.Parallel()

	handler := NewSystemd(&stubSystemd{}).Restarts

	method := runTestCaseWithPath(t, handler, pathRestarts, unitParam, unitCatalogo, testCaseRequest{
		name: testCaseMethodNotPermitido, method: http.MethodGet,
	})
	checkTestCase(t, method, testCaseRequest{name: testCaseMethodNotPermitido, code: http.StatusMethodNotAllowed})

	invalida := runTestCaseWithPath(t, handler, pathRestarts, unitParam, "catalogo", testCaseRequest{
		name: testCaseUnitInvalida, method: http.MethodPost,
	})
	checkTestCase(t, invalida, testCaseRequest{name: testCaseUnitInvalida, code: http.StatusBadRequest, field: unitParam})
}
