// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"consola-vm/internal/services"
)

// descriptionPkgNano es el resumen esperado al instalar nano.
const descriptionPkgNano = "Instalar el paquete «nano»"

// kindJSONTests es el tipo de contenido de las peticiones de prueba.
const kindJSONTests = "application/json"

// msgRegisterLlamadas es el formato para comprobar las llamadas a Register.
const msgRegisterLlamadas = "Registrar() se llamó %d veces, want 1"

// stubHistory es un historial falso con resultados preprogramados.
type stubHistory struct {
	lists       []services.Operation
	err         error
	filter      services.HistoryFilter
	registradas []services.Operation
}

// List devuelve la lista preprogramada.
func (s *stubHistory) List(_ context.Context, filter *services.HistoryFilter) ([]services.Operation, error) {
	s.filter = *filter

	return s.lists, s.err
}

// Register acumula las operaciones registradas.
func (s *stubHistory) Register(_ context.Context, operation *services.Operation) error {
	s.registradas = append(s.registradas, *operation)

	return s.err
}

// TestHistoryList cubre la consulta y los filtros.
func TestHistoryList(t *testing.T) {
	t.Parallel()

	service := &stubHistory{lists: []services.Operation{
		{ID: 1, DateHour: "2026-10-25T10:00:00Z", VM: testMachine, Module: "paquetes", Operation: "POST /api/packages", Description: descriptionPkgNano, Outcome: "ok"},
	}}

	path := "/api/historial?q=nano&vm=vm-1&modulo=paquetes&desde=2026-10-25&hasta=2026-10-26&hora_desde=08:00&hora_hasta=18:00&limite=50"

	rec := runTestCase(t, NewHistory(service).ServeHTTP, path, testCaseRequest{
		name: testCaseList, method: http.MethodGet,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf(msgCode, rec.Code, http.StatusOK)
	}

	if service.filter.Text != "nano" || service.filter.VM != testMachine || service.filter.Limit != 50 {
		t.Errorf("filtro = %+v, want los parámetros de la query", service.filter)
	}

	var body historyList
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf(msgJSONInvalid, err)
	}

	if len(body.Operations) != 1 || body.Operations[0].VM != testMachine {
		t.Errorf("operaciones = %+v, want vm-1", body.Operations)
	}

	if body.Operations[0].Description != descriptionPkgNano {
		t.Errorf("descripcion = %q, want el resumen del historial", body.Operations[0].Description)
	}
}

// TestHistoryValidacion cubre fechas, horas, límite y método.
func TestHistoryValidacion(t *testing.T) {
	t.Parallel()

	handler := NewHistory(&stubHistory{}).ServeHTTP

	testCases := []struct {
		testCaseRequest
		path string
	}{
		{testCaseRequest{name: "fecha inválida", method: http.MethodGet, code: http.StatusBadRequest, field: "desde"}, "/api/historial?desde=25-10-2026"},
		{testCaseRequest{name: "hora inválida", method: http.MethodGet, code: http.StatusBadRequest, field: "hora_hasta"}, "/api/historial?hora_hasta=25:00"},
		{testCaseRequest{name: "límite inválido", method: http.MethodGet, code: http.StatusBadRequest, field: "limite"}, "/api/historial?limite=0"},
		{testCaseRequest{name: testCaseMethodNotPermitido, method: http.MethodPost, code: http.StatusMethodNotAllowed}, "/api/historial"},
	}

	for _, testCase := range testCases {
		rec := runTestCase(t, handler, testCase.path, testCase.testCaseRequest)
		checkTestCase(t, rec, testCase.testCaseRequest)
	}
}

// TestWithHistoryRegistersMutaciones comprueba que solo se registran las
// operaciones que no son GET.
func TestWithHistoryRegistersMutaciones(t *testing.T) {
	t.Parallel()

	service := &stubHistory{}
	handler := WithHistory(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
		}),
		service,
		func() string { return testMachine },
	)

	post := httptest.NewRequestWithContext(t.Context(), http.MethodPost, pathPackages, http.NoBody)
	postRec := httptest.NewRecorder()
	handler.ServeHTTP(postRec, post)

	if len(service.registradas) != 1 {
		t.Fatalf(msgRegisterLlamadas, len(service.registradas))
	}

	if service.registradas[0].Outcome != "ok" || service.registradas[0].VM != testMachine {
		t.Errorf("operación = %+v, want ok en vm-1", service.registradas[0])
	}

	get := httptest.NewRequestWithContext(t.Context(), http.MethodGet, pathPackages, http.NoBody)
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, get)

	if len(service.registradas) != 1 {
		t.Errorf("una consulta GET se registró en el historial")
	}
}

// TestWithHistoryDescribeYRestoresBody comprueba que el resumen usa
// el cuerpo sin consumirlo para el handler ni guardar secretos.
func TestWithHistoryDescribeYRestoresBody(t *testing.T) {
	t.Parallel()

	service := &stubHistory{}
	received := map[string]any{}

	handler := WithHistory(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("leer el cuerpo restaurado: %v", err)
			}

			if err := json.Unmarshal(body, &received); err != nil {
				t.Errorf("decodificar el cuerpo restaurado: %v", err)
			}

			w.WriteHeader(http.StatusCreated)
		}),
		service,
		func() string { return testMachine },
	)

	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, pathPackages,
		strings.NewReader(`{"nombre":"nano","password":"secreto"}`))
	request.Header.Set(headerContentType, kindJSONTests)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, request)

	if received["nombre"] != "nano" {
		t.Errorf("el handler recibió %+v, want el cuerpo restaurado", received)
	}

	if len(service.registradas) != 1 {
		t.Fatalf(msgRegisterLlamadas, len(service.registradas))
	}

	operation := service.registradas[0]
	if operation.Description != descriptionPkgNano {
		t.Errorf("Descripcion = %q, want el resumen del paquete", operation.Description)
	}

	if strings.Contains(operation.Description, "secreto") || strings.Contains(operation.Operation, "secreto") {
		t.Error("el historial guardó la contraseña")
	}
}

// TestWithHistoryUsesMachineTarget comprueba que las operaciones de
// sesión y de encendido se atribuyen a la máquina solicitada.
func TestWithHistoryUsesMachineTarget(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		method   string
		path     string
		body     string
		esperada string
	}{
		{
			name:     "registro",
			method:   http.MethodPost,
			path:     "/api/session/register",
			body:     `{"vm":"ubuntu-lab","usuario":"jorge"}`,
			esperada: "ubuntu-lab",
		},
		{
			name:     "encendido",
			method:   http.MethodPost,
			path:     "/api/vms/ubuntu-lab/poweron",
			esperada: "ubuntu-lab",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := &stubHistory{}
			handler := WithHistory(
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusAccepted)
				}),
				service,
				func() string { return testMachine },
			)

			var body io.Reader = http.NoBody
			if testCase.body != "" {
				body = io.NopCloser(strings.NewReader(testCase.body))
			}

			request := httptest.NewRequestWithContext(t.Context(), testCase.method, testCase.path, body)
			if testCase.body != "" {
				request.Header.Set(headerContentType, kindJSONTests)
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, request)

			if len(service.registradas) != 1 {
				t.Fatalf(msgRegisterLlamadas, len(service.registradas))
			}

			if obtenida := service.registradas[0].VM; obtenida != testCase.esperada {
				t.Errorf("VM = %q, want %q", obtenida, testCase.esperada)
			}
		})
	}
}

// TestWithHistoryBodyNulo comprueba que un JSON null no rompe el
// middleware ni el resumen.
func TestWithHistoryBodyNulo(t *testing.T) {
	t.Parallel()

	service := &stubHistory{}
	handler := WithHistory(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
		}),
		service,
		func() string { return testMachine },
	)

	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost,
		pathPackages+"?files=true", strings.NewReader("null"))
	request.Header.Set(headerContentType, kindJSONTests)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, request)

	if len(service.registradas) != 1 {
		t.Fatalf(msgRegisterLlamadas, len(service.registradas))
	}

	if obtained := service.registradas[0].Description; obtained != "Instalar un paquete" {
		t.Errorf("Descripcion = %q, want el resumen genérico", obtained)
	}
}

// TestWithHistoryDescribeBorradoWithFiles comprueba que la opción
// files de la query se refleja en el resumen.
func TestWithHistoryDescribeBorradoWithFiles(t *testing.T) {
	t.Parallel()

	service := &stubHistory{}
	handler := WithHistory(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
		service,
		func() string { return testMachine },
	)

	request := httptest.NewRequestWithContext(t.Context(), http.MethodDelete,
		"/api/users/jorge?files=true", http.NoBody)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, request)

	if len(service.registradas) != 1 {
		t.Fatalf(msgRegisterLlamadas, len(service.registradas))
	}

	if obtained := service.registradas[0].Description; obtained != "Eliminar el usuario «jorge» y sus archivos" {
		t.Errorf("Descripcion = %q, want el borrado con archivos", obtained)
	}
}
