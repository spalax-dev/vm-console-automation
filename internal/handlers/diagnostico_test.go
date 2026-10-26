// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/netip"
	"testing"

	"consola-vm/internal/core/domain"
	"consola-vm/internal/services"
)

// pathDiagnostics es la ruta del recurso de diagnóstico.
const pathDiagnostics = "/api/session/diagnostico"

// stubDiagnostics devuelve un informe preprogramado.
type stubDiagnostics struct {
	report services.Diagnostics
	err    error
}

// Get devuelve el informe o el error configurados.
func (s *stubDiagnostics) Get(context.Context) (services.Diagnostics, error) {
	return s.report, s.err
}

// TestDiagnosticsHandlerRespondsReport cubre la respuesta del informe.
func TestDiagnosticsHandlerRespondsReport(t *testing.T) {
	t.Parallel()

	service := &stubDiagnostics{report: services.Diagnostics{
		VM:                     testMachine,
		State:                  domain.StateRunning,
		Network:                domain.NetworkBridged,
		IP:                     netipMustParse(t, ipTests),
		GuestAdditionsVersion:  "6.0.0 r127566",
		GuestAdditionsRunLevel: 2,
		UserSugerido:           userJorge,
		Problemas:              []string{"SSH no responde"},
		Sugerencias:            []string{"Instala openssh-server"},
	}}

	rec := runTestCase(t, NewDiagnostics(service).ServeHTTP, pathDiagnostics, testCaseRequest{
		name: "diagnóstico", method: http.MethodGet,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf(msgCode, rec.Code, http.StatusOK)
	}

	var body diagnosticsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf(msgJSONInvalid, err)
	}

	if body.VM != testMachine || body.UserSugerido != userJorge || body.SSHOpen {
		t.Errorf("cuerpo = %+v, want vm1/jorge/SSH cerrado", body)
	}
}

// TestDiagnosticsHandlerWithoutMachine cubre la consola sin vínculo.
func TestDiagnosticsHandlerWithoutMachine(t *testing.T) {
	t.Parallel()

	rec := runTestCase(t, NewDiagnostics(&stubDiagnostics{err: services.ErrNoMachine}).ServeHTTP, pathDiagnostics, testCaseRequest{
		name: "sin máquina", method: http.MethodGet,
	})

	if rec.Code != http.StatusConflict {
		t.Fatalf(msgCode, rec.Code, http.StatusConflict)
	}
}

// TestDiagnosticsHandlerMethod cubre el método no permitido.
func TestDiagnosticsHandlerMethod(t *testing.T) {
	t.Parallel()

	rec := runTestCase(t, NewDiagnostics(&stubDiagnostics{}).ServeHTTP, pathDiagnostics, testCaseRequest{
		name: testCaseMethodNotPermitido, method: http.MethodPost,
	})

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf(msgCode, rec.Code, http.StatusMethodNotAllowed)
	}
}

// netipMustParse evita importar net/netip en varias aserciones.
func netipMustParse(t *testing.T, value string) (address netip.Addr) {
	t.Helper()

	address, err := netip.ParseAddr(value)
	if err != nil {
		t.Fatalf("ParseAddr(%q): %v", value, err)
	}

	return address
}
