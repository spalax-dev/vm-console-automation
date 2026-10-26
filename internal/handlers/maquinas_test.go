// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"consola-vm/internal/core/domain"
	"consola-vm/internal/services"
)

// stateRunning es el estado de encendido usado en las pruebas.
const stateRunning = "running"

// Valores repetidos en las pruebas del inventario.
const (
	pathMachines = "/api/vms"
	testMachine  = "vm-1"
)

// stubMachines implementa MachinesService para las pruebas.
type stubMachines struct {
	machines   []services.VirtualMachine
	filter     string
	err        error
	encendida  string
	errPowerOn error
}

// List devuelve el inventario preprogramado y registra el filtro.
func (s *stubMachines) List(_ context.Context, filter string) ([]services.VirtualMachine, error) {
	s.filter = filter

	return s.machines, s.err
}

// PowerOn registra la máquina y devuelve el error preprogramado.
func (s *stubMachines) PowerOn(_ context.Context, identifier string) (services.VirtualMachine, error) {
	s.encendida = identifier

	if s.errPowerOn != nil {
		return services.VirtualMachine{}, s.errPowerOn
	}

	return services.VirtualMachine{Name: identifier, State: domain.StateRunning, Active: true}, nil
}

// Bind devuelve la primera máquina o el error preprogramado.
func (s *stubMachines) Bind(_ context.Context, _ string) (services.VirtualMachine, error) {
	if s.err != nil {
		return services.VirtualMachine{}, s.err
	}

	return s.machines[0], nil
}

// TestMachinesListsRespondsJSON comprueba el contrato de GET /api/vms.
func TestMachinesListsRespondsJSON(t *testing.T) {
	t.Parallel()

	service := &stubMachines{machines: []services.VirtualMachine{
		{Name: testMachine, UUID: "uuid-1", State: domain.StateRunning, Active: true},
	}}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, pathMachines+"?q=deb", http.NoBody)
	rec := httptest.NewRecorder()

	NewMachines(service).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(msgCode, rec.Code, http.StatusOK)
	}

	if service.filter != "deb" {
		t.Errorf("filtro = %q, want %q", service.filter, "deb")
	}

	var body struct {
		Machines []map[string]any `json:"maquinas"`
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf(msgJSONInvalid, err)
	}

	if len(body.Machines) != 1 {
		t.Fatalf("len(maquinas) = %d, want 1", len(body.Machines))
	}

	if body.Machines[0]["nombre"] != testMachine {
		t.Errorf("nombre = %v, want %q", body.Machines[0]["nombre"], testMachine)
	}

	if body.Machines[0]["activa"] != true {
		t.Errorf("activa = %v, want true", body.Machines[0]["activa"])
	}
}

// TestMachinesRejectsMethodsNotGET comprueba que el inventario es de
// solo lectura.
func TestMachinesRejectsMethodsNotGET(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, pathMachines, http.NoBody)
	rec := httptest.NewRecorder()

	NewMachines(&stubMachines{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf(msgCode, rec.Code, http.StatusMethodNotAllowed)
	}
}

// TestMachinesErrorOfHypervisorResponde500 comprueba que un fallo de
// VBoxManage no se confunde con una lista vacía.
func TestMachinesErrorOfHypervisorResponde500(t *testing.T) {
	t.Parallel()

	service := &stubMachines{err: errors.New("VBoxManage no está disponible")}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, pathMachines, http.NoBody)
	rec := httptest.NewRecorder()

	NewMachines(service).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCode, rec.Code, http.StatusInternalServerError)
	}
}

// TestSessionBindsMachine comprueba PUT /api/session.
func TestSessionBindsMachine(t *testing.T) {
	t.Parallel()

	vinculo := &stubBinding{machine: services.VirtualMachine{Name: testMachine, State: domain.StateRunning, Active: true}}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, pathSession,
		strings.NewReader(`{"vm":"vm-1"}`))
	rec := httptest.NewRecorder()

	NewSession(&stubDiscoverer{}, WithBinder(vinculo)).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(msgCode, rec.Code, http.StatusOK)
	}

	if vinculo.vinculada != testMachine {
		t.Errorf("vm vinculada = %q, want %q", vinculo.vinculada, testMachine)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf(msgJSONInvalid, err)
	}

	if body["nombre"] != testMachine {
		t.Errorf("nombre = %v, want %q", body["nombre"], testMachine)
	}
}

// TestSessionBindsMachineInexistenteResponde404 comprueba el error de
// una máquina que no está en el inventario.
func TestSessionBindsMachineInexistenteResponde404(t *testing.T) {
	t.Parallel()

	vinculo := &stubBinding{err: services.ErrMachineNotFound}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, pathSession,
		strings.NewReader(`{"vm":"fantasma"}`))
	rec := httptest.NewRecorder()

	NewSession(&stubDiscoverer{}, WithBinder(vinculo)).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf(msgCode, rec.Code, http.StatusNotFound)
	}
}

// TestSessionBindsWithoutNameResponde400 comprueba la validación del
// campo obligatorio.
func TestSessionBindsWithoutNameResponde400(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, pathSession,
		strings.NewReader(`{"vm":"  "}`))
	rec := httptest.NewRecorder()

	NewSession(&stubDiscoverer{}, WithBinder(&stubBinding{})).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(msgCode, rec.Code, http.StatusBadRequest)
	}
}

// TestSessionBindsBodyInvalidResponde400 comprueba el rechazo de un
// JSON que no cumple el esquema.
func TestSessionBindsBodyInvalidResponde400(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, pathSession,
		strings.NewReader(`{"maquina":testMachine}`))
	rec := httptest.NewRecorder()

	NewSession(&stubDiscoverer{}, WithBinder(&stubBinding{})).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(msgCode, rec.Code, http.StatusBadRequest)
	}
}

// TestSessionWithoutBinderRejectsPUT evita aceptar una vinculación que
// no se puede aplicar.
func TestSessionWithoutBinderRejectsPUT(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, pathSession,
		strings.NewReader(`{"vm":testMachine}`))
	rec := httptest.NewRecorder()

	NewSession(&stubDiscoverer{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf(msgCode, rec.Code, http.StatusMethodNotAllowed)
	}
}

// TestSessionPublicMachineVinculada comprueba el campo vm de GET.
func TestSessionPublicMachineVinculada(t *testing.T) {
	t.Parallel()

	desc := &stubDiscoverer{info: services.SessionInfo{Connector: connectorSSH, State: domain.StateRunning}}
	vinculo := &stubBinding{active: testMachine}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, pathSession, http.NoBody)
	rec := httptest.NewRecorder()

	NewSession(desc, WithBinder(vinculo)).ServeHTTP(rec, req)

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf(msgJSONInvalid, err)
	}

	if body["vm"] != testMachine {
		t.Errorf("vm = %v, want %q", body["vm"], testMachine)
	}
}

// stubBinding implementa SessionBinder.
type stubBinding struct {
	machine   services.VirtualMachine
	vinculada string
	active    string
	err       error
}

// Bind registra el identificador y devuelve la máquina o el error.
func (s *stubBinding) Bind(_ context.Context, id string) (services.VirtualMachine, error) {
	s.vinculada = id

	if s.err != nil {
		return services.VirtualMachine{}, s.err
	}

	return s.machine, nil
}

// Active devuelve la máquina vinculada.
func (s *stubBinding) Active() string { return s.active }

// TestMachinesPowersOnRespondsMachine comprueba POST
// /api/vms/{vm}/poweron.
func TestMachinesPowersOnRespondsMachine(t *testing.T) {
	t.Parallel()

	service := &stubMachines{}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, pathMachines+"/vm-1/poweron", http.NoBody)
	req.SetPathValue("vm", "vm-1")

	rec := httptest.NewRecorder()

	NewMachines(service).PowerOn(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(msgCode, rec.Code, http.StatusOK)
	}

	if service.encendida != testMachine {
		t.Errorf("encendida = %q, want %q", service.encendida, testMachine)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf(msgJSONInvalid, err)
	}

	if body["estado"] != stateRunning {
		t.Errorf("estado = %v, want %q", body["estado"], stateRunning)
	}
}

// TestMachinesPowersOnMachineYaEncendidaResponde409 comprueba el
// conflicto de estado.
func TestMachinesPowersOnMachineYaEncendidaResponde409(t *testing.T) {
	t.Parallel()

	service := &stubMachines{errPowerOn: services.ErrMachineRunning}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, pathMachines+"/vm-1/poweron", http.NoBody)
	req.SetPathValue("vm", testMachine)

	rec := httptest.NewRecorder()

	NewMachines(service).PowerOn(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf(msgCode, rec.Code, http.StatusConflict)
	}
}

// TestMachinesPowersOnMachineUnknownResponde404 comprueba el
// recurso inexistente.
func TestMachinesPowersOnMachineUnknownResponde404(t *testing.T) {
	t.Parallel()

	service := &stubMachines{errPowerOn: services.ErrMachineNotFound}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, pathMachines+"/fantasma/poweron", http.NoBody)
	req.SetPathValue("vm", "fantasma")

	rec := httptest.NewRecorder()

	NewMachines(service).PowerOn(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf(msgCode, rec.Code, http.StatusNotFound)
	}
}

// TestMachinesPowersOnRejectsMethodsNotPOST comprueba el verbo.
func TestMachinesPowersOnRejectsMethodsNotPOST(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, pathMachines+"/vm-1/poweron", http.NoBody)
	req.SetPathValue("vm", testMachine)

	rec := httptest.NewRecorder()

	NewMachines(&stubMachines{}).PowerOn(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf(msgCode, rec.Code, http.StatusMethodNotAllowed)
	}
}
