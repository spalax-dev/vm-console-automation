// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"consola-vm/internal/services"
)

// stubHAProxy es un servicio de HAProxy falso con resultados
// preprogramados.
type stubHAProxy struct {
	lists     []services.Server
	result    services.Outcome
	err       error
	address   string
	port      int
	agregados int
	deleted   int
	status    services.HAProxyStatus
	setup     int
}

// List devuelve la lista preprogramada.
func (s *stubHAProxy) List(context.Context) ([]services.Server, error) {
	return s.lists, s.err
}

// Add registra los argumentos y devuelve el resultado.
func (s *stubHAProxy) Add(_ context.Context, address string, port int) (services.Outcome, error) {
	s.address = address
	s.port = port
	s.agregados++

	return s.result, s.err
}

// Delete registra los argumentos y devuelve el resultado.
func (s *stubHAProxy) Delete(_ context.Context, address string, port int) (services.Outcome, error) {
	s.address = address
	s.port = port
	s.deleted++

	return s.result, s.err
}

// Status devuelve el estado preprogramado.
func (s *stubHAProxy) Status(context.Context) (services.HAProxyStatus, error) {
	return s.status, s.err
}

// Setup registra la llamada y devuelve el resultado.
func (s *stubHAProxy) Setup(context.Context) (services.Outcome, error) {
	s.setup++

	return s.result, s.err
}

// TestServersList cubre el listado del back-end.
func TestServersList(t *testing.T) {
	t.Parallel()

	service := &stubHAProxy{lists: []services.Server{
		{Address: ipServer, Port: 8080, State: "unknown"},
	}}

	rec := runTestCase(t, NewHAProxy(service).Servers, pathServers, testCaseRequest{
		name: testCaseList, method: http.MethodGet,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf(msgCode, rec.Code, http.StatusOK)
	}

	var body backendList
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf(msgJSONInvalid, err)
	}

	if len(body.Servers) != 1 || body.Servers[0].Port != 8080 {
		t.Errorf("servidores = %+v, want 192.168.1.61:8080", body.Servers)
	}
}

// TestServersAdd cubre el alta de un servidor.
func TestServersAdd(t *testing.T) {
	t.Parallel()

	service := &stubHAProxy{result: services.Outcome{Message: "Servidor agregado."}}

	rec := runTestCase(t, NewHAProxy(service).Servers, pathServers, testCaseRequest{
		name: "agregar", method: http.MethodPost, body: bodyServer,
	})

	if rec.Code != http.StatusCreated {
		t.Fatalf(msgCode, rec.Code, http.StatusCreated)
	}

	if got := rec.Header().Get(headerLocation); got != "/api/haproxy/backend/servers/192.168.1.61:8080" {
		t.Errorf("Location = %q, want la ruta del servidor", got)
	}

	if service.agregados != 1 || service.port != 8080 {
		t.Errorf("Agregar() = %+v, want 192.168.1.61:8080", service)
	}
}

// TestServersDuplicate cubre el servidor que ya existe.
func TestServersDuplicate(t *testing.T) {
	t.Parallel()

	service := &stubHAProxy{err: services.ErrServerExists}

	rec := runTestCase(t, NewHAProxy(service).Servers, pathServers, testCaseRequest{
		name: "duplicado", method: http.MethodPost, body: bodyServer,
	})

	if rec.Code != http.StatusConflict {
		t.Fatalf(msgCode, rec.Code, http.StatusConflict)
	}
}

// TestServersValidacion cubre la dirección y el puerto.
func TestServersValidacion(t *testing.T) {
	t.Parallel()

	handler := NewHAProxy(&stubHAProxy{}).Servers
	testCases := []testCaseRequest{
		{name: "hostname", method: http.MethodPost, body: `{"direccion":"localhost","puerto":8080}`, code: http.StatusBadRequest, field: addressField},
		{name: "puerto fuera de rango", method: http.MethodPost, body: `{"direccion":"192.168.1.61","puerto":70000}`, code: http.StatusBadRequest, field: portField},
		{name: testCaseBodyInvalid, method: http.MethodPost, body: `{`, code: http.StatusBadRequest},
		{name: testCaseMethodNotPermitido, method: http.MethodPut, code: http.StatusMethodNotAllowed},
	}

	for _, testCase := range testCases {
		rec := runTestCase(t, handler, pathServers, testCase)
		checkTestCase(t, rec, testCase)
	}
}

// TestServerDelete cubre la eliminación de un servidor.
func TestServerDelete(t *testing.T) {
	t.Parallel()

	service := &stubHAProxy{}

	rec := runTestCaseWithPath(t, NewHAProxy(service).Server, pathServer, serverParam, serverTests, testCaseRequest{
		name: testCaseDelete, method: http.MethodDelete,
	})

	if rec.Code != http.StatusNoContent {
		t.Fatalf(msgCode, rec.Code, http.StatusNoContent)
	}

	if service.deleted != 1 || service.address != ipServer || service.port != 8080 {
		t.Errorf("Eliminar() = %+v, want 192.168.1.61:8080", service)
	}
}

// TestServerNotFound cubre un servidor ausente.
func TestServerNotFound(t *testing.T) {
	t.Parallel()

	service := &stubHAProxy{err: services.ErrServerNotFound}

	rec := runTestCaseWithPath(t, NewHAProxy(service).Server, pathServer, serverParam, serverTests, testCaseRequest{
		name: testCaseInexistente, method: http.MethodDelete,
	})

	if rec.Code != http.StatusNotFound {
		t.Fatalf(msgCode, rec.Code, http.StatusNotFound)
	}
}

// TestServerValidacion cubre el identificador de la ruta.
func TestServerValidacion(t *testing.T) {
	t.Parallel()

	handler := NewHAProxy(&stubHAProxy{}).Server
	testCases := []struct {
		testCaseRequest
		server string
	}{
		{testCaseRequest{name: "sin puerto", method: http.MethodDelete, code: http.StatusBadRequest, field: serverParam}, ipServer},
		{testCaseRequest{name: "puerto cero", method: http.MethodDelete, code: http.StatusBadRequest, field: serverParam}, "192.168.1.61:0"},
		{testCaseRequest{name: testCaseMethodNotPermitido, method: http.MethodPost, code: http.StatusMethodNotAllowed}, serverTests},
	}

	for _, testCase := range testCases {
		rec := runTestCaseWithPath(t, handler, pathServer, serverParam, testCase.server, testCase.testCaseRequest)
		checkTestCase(t, rec, testCase.testCaseRequest)
	}
}

// TestServersError cubre el fallo al agregar un servidor.
func TestServersError(t *testing.T) {
	t.Parallel()

	service := &stubHAProxy{err: errRead}

	rec := runTestCase(t, NewHAProxy(service).Servers, pathServers, testCaseRequest{
		name: "fallo de alta", method: http.MethodPost, body: bodyServer,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCode, rec.Code, http.StatusInternalServerError)
	}
}

// TestServersListError cubre el fallo al listar el back-end.
func TestServersListError(t *testing.T) {
	t.Parallel()

	rec := runTestCase(t, NewHAProxy(&stubHAProxy{err: errRead}).Servers, pathServers, testCaseRequest{
		name: testCaseList, method: http.MethodGet,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCode, rec.Code, http.StatusInternalServerError)
	}
}

// TestServerError cubre el fallo al eliminar un servidor.
func TestServerError(t *testing.T) {
	t.Parallel()

	rec := runTestCaseWithPath(t, NewHAProxy(&stubHAProxy{err: errRead}).Server, pathServer, serverParam, serverTests, testCaseRequest{
		name: testCaseDelete, method: http.MethodDelete,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCode, rec.Code, http.StatusInternalServerError)
	}
}

// TestHAProxyStatus cubre la consulta del estado de HAProxy.
func TestHAProxyStatus(t *testing.T) {
	t.Parallel()

	service := &stubHAProxy{status: services.HAProxyStatus{
		Installed: true, ConfigExists: true, ConfigValid: true, Backend: true, ServiceActive: true,
	}}

	rec := runTestCase(t, NewHAProxy(service).Status, "/api/haproxy/status", testCaseRequest{
		name: "estado", method: http.MethodGet, code: http.StatusOK,
	})

	if !strings.Contains(rec.Body.String(), `"instalado":true`) {
		t.Errorf("cuerpo = %q, want instalado=true", rec.Body.String())
	}
}

// TestHAProxySetup cubre la configuración rápida.
func TestHAProxySetup(t *testing.T) {
	t.Parallel()

	service := &stubHAProxy{result: services.Outcome{Message: "HAProxy ya estaba configurado."}}
	runTestCase(t, NewHAProxy(service).Setup, "/api/haproxy/setup", testCaseRequest{
		name: "configurar", method: http.MethodPost, code: http.StatusOK,
	})

	if service.setup != 1 {
		t.Errorf("Setup() = %d, want 1", service.setup)
	}
}
