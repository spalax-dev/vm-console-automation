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
	rutaSesion  = "/api/sesion"
	conectorSSH = "ssh"
	ipPrueba    = "192.168.1.50"
	msgCodigo   = "código = %d, want %d"
)

// stubDescubridor es un descubridor falso que devuelve una sesión o un
// error preprogramados.
type stubDescubridor struct {
	info services.InfoSesion
	err  error
}

// Descubrir devuelve la información o el error preprogramados.
func (s *stubDescubridor) Descubrir(context.Context) (services.InfoSesion, error) {
	return s.info, s.err
}

// TestSesionHandlerRespondeJSON comprueba el contrato de GET
// /api/sesion: conector, IP y estado en JSON.
func TestSesionHandlerRespondeJSON(t *testing.T) {
	t.Parallel()

	desc := &stubDescubridor{info: services.InfoSesion{
		Conector: conectorSSH,
		IP:       netip.MustParseAddr(ipPrueba),
		Estado:   domain.StateRunning,
	}}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, rutaSesion, http.NoBody)
	rec := httptest.NewRecorder()

	NuevoSesion(desc).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(msgCodigo, rec.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("respuesta no es JSON válido: %v", err)
	}

	if body["conector"] != conectorSSH {
		t.Errorf("conector = %v, want %q", body["conector"], conectorSSH)
	}

	if body["ip"] != ipPrueba {
		t.Errorf("ip = %v, want %q", body["ip"], ipPrueba)
	}

	if body["estado"] != string(domain.StateRunning) {
		t.Errorf("estado = %v, want %q", body["estado"], domain.StateRunning)
	}
}

// TestSesionHandlerSinSesionResponde503 comprueba que la ausencia de
// conector se traduce en 503 con un mensaje comprensible.
func TestSesionHandlerSinSesionResponde503(t *testing.T) {
	t.Parallel()

	desc := &stubDescubridor{err: services.ErrSesionNoDisponible}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, rutaSesion, http.NoBody)
	rec := httptest.NewRecorder()

	NuevoSesion(desc).ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf(msgCodigo, rec.Code, http.StatusServiceUnavailable)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("respuesta no es JSON válido: %v", err)
	}

	if body["ok"] != false {
		t.Errorf("ok = %v, want false", body["ok"])
	}

	if body["mensaje"] == "" {
		t.Error("mensaje vacío, want un texto comprensible para el usuario")
	}
}

// TestSesionHandlerRechazaMetodosNoGET comprueba que solo se admite GET.
func TestSesionHandlerRechazaMetodosNoGET(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, rutaSesion, http.NoBody)
	rec := httptest.NewRecorder()

	NuevoSesion(&stubDescubridor{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf(msgCodigo, rec.Code, http.StatusMethodNotAllowed)
	}
}

// TestSesionHandlerErrorInternoResponde500 comprueba que un fallo del
// descubrimiento distinto de «sin sesión» se responde como 500.
func TestSesionHandlerErrorInternoResponde500(t *testing.T) {
	t.Parallel()

	desc := &stubDescubridor{err: errors.New("fallo inesperado")}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, rutaSesion, http.NoBody)
	rec := httptest.NewRecorder()

	NuevoSesion(desc).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCodigo, rec.Code, http.StatusInternalServerError)
	}
}
