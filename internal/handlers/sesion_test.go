// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"consola-vm/internal/core/domain"
	"consola-vm/internal/services"
)

// Valores repetidos en las pruebas del handler de sesión.
const (
	pathSession    = "/api/session"
	connectorSSH   = "ssh"
	ipTests        = "192.168.1.50"
	msgCode        = "código = %d, want %d"
	msgJSONInvalid = "respuesta no es JSON válido: %v"
)

// stubDiscoverer es un descubridor falso que devuelve una sesión o un
// error preprogramados.
type stubDiscoverer struct {
	info services.SessionInfo
	err  error
}

// Discover devuelve la información o el error preprogramados.
func (s *stubDiscoverer) Discover(context.Context) (services.SessionInfo, error) {
	return s.info, s.err
}

// TestSessionHandlerRespondsJSON comprueba el contrato de GET
// /api/session: conector, IP y estado en JSON.
func TestSessionHandlerRespondsJSON(t *testing.T) {
	t.Parallel()

	desc := &stubDiscoverer{info: services.SessionInfo{
		Connector: connectorSSH,
		IP:        netip.MustParseAddr(ipTests),
		State:     domain.StateRunning,
	}}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, pathSession, http.NoBody)
	rec := httptest.NewRecorder()

	NewSession(desc).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(msgCode, rec.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf(msgJSONInvalid, err)
	}

	if body["conector"] != connectorSSH {
		t.Errorf("conector = %v, want %q", body["conector"], connectorSSH)
	}

	if body["ip"] != ipTests {
		t.Errorf("ip = %v, want %q", body["ip"], ipTests)
	}

	if body["estado"] != string(domain.StateRunning) {
		t.Errorf("estado = %v, want %q", body["estado"], domain.StateRunning)
	}
}

// TestSessionHandlerWithoutSessionResponde503 comprueba que la ausencia de
// conector se traduce en 503 con un mensaje comprensible.
func TestSessionHandlerWithoutSessionResponde503(t *testing.T) {
	t.Parallel()

	desc := &stubDiscoverer{err: services.ErrSessionUnavailable}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, pathSession, http.NoBody)
	rec := httptest.NewRecorder()

	NewSession(desc).ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf(msgCode, rec.Code, http.StatusServiceUnavailable)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf(msgJSONInvalid, err)
	}

	if body["ok"] != false {
		t.Errorf("ok = %v, want false", body["ok"])
	}

	if body["mensaje"] == "" {
		t.Error("mensaje vacío, want un texto comprensible para el usuario")
	}
}

// TestSessionHandlerRejectsMethodsNotGET comprueba que solo se admite GET.
func TestSessionHandlerRejectsMethodsNotGET(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, pathSession, http.NoBody)
	rec := httptest.NewRecorder()

	NewSession(&stubDiscoverer{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf(msgCode, rec.Code, http.StatusMethodNotAllowed)
	}
}

// TestSessionHandlerErrorInternalResponde500 comprueba que un fallo del
// descubrimiento distinto de «sin sesión» se responde como 500.
func TestSessionHandlerErrorInternalResponde500(t *testing.T) {
	t.Parallel()

	desc := &stubDiscoverer{err: errors.New("fallo inesperado")}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, pathSession, http.NoBody)
	rec := httptest.NewRecorder()

	NewSession(desc).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCode, rec.Code, http.StatusInternalServerError)
	}
}

// stubRegistrationSession informa si la máquina está registrada.
type stubRegistrationSession struct {
	registered bool
}

// IsRegistered devuelve el estado preprogramado.
func (s *stubRegistrationSession) IsRegistered(string) bool {
	return s.registered
}

// TestSessionHandlerInformaRegistration comprueba el campo registrada.
func TestSessionHandlerInformaRegistration(t *testing.T) {
	t.Parallel()

	desc := &stubDiscoverer{info: services.SessionInfo{
		Connector: connectorSSH,
		IP:        netip.MustParseAddr(ipTests),
		State:     domain.StateRunning,
	}}

	handler := NewSession(desc, WithRegistration(&stubRegistrationSession{registered: true}))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, pathSession, http.NoBody)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf(msgJSONInvalid, err)
	}

	if body["registrada"] != true {
		t.Errorf("registrada = %v, want true", body["registrada"])
	}
}
