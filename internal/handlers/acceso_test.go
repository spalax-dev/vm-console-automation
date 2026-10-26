// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"context"
	"net/http"
	"testing"

	"consola-vm/internal/services"
)

// pathRegistration es la ruta del recurso de registro.
const pathRegistration = "/api/session/register"

// stubAccess es un servicio de credenciales falso.
type stubAccess struct {
	result     services.Outcome
	err        error
	user       string
	password   string
	registered int
	olvidados  int
}

// Register registra los argumentos y devuelve el resultado.
func (s *stubAccess) Register(_ context.Context, user, password string) (services.Outcome, error) {
	s.user = user
	s.password = password
	s.registered++

	return s.result, s.err
}

// Forget registra la llamada.
func (s *stubAccess) Forget() error {
	s.olvidados++

	return s.err
}

// TestAccessStart cubre el guardado correcto de credenciales.
func TestAccessStart(t *testing.T) {
	t.Parallel()

	service := &stubAccess{result: services.Outcome{Message: "Credenciales verificadas."}}

	rec := runTestCase(t, NewAccess(service).ServeHTTP, pathRegistration, testCaseRequest{
		name: "registrar", method: http.MethodPost, body: `{"usuario":"root","password":"secreto"}`,
	})

	if rec.Code != http.StatusCreated {
		t.Fatalf(msgCode, rec.Code, http.StatusCreated)
	}

	if service.registered != 1 || service.user != "root" {
		t.Errorf("Registrar() = %+v, want root", service)
	}
}

// TestAccessStartRechazado cubre las credenciales inválidas.
func TestAccessStartRechazado(t *testing.T) {
	t.Parallel()

	service := &stubAccess{err: services.ErrInvalidCredentials}

	rec := runTestCase(t, NewAccess(service).ServeHTTP, pathRegistration, testCaseRequest{
		name: "credenciales inválidas", method: http.MethodPost, body: `{"usuario":"root","password":"mala"}`,
	})

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf(msgCode, rec.Code, http.StatusUnauthorized)
	}
}

// TestAccessStartWithoutSession cubre la máquina apagada o sin Guest
// Additions.
func TestAccessStartWithoutSession(t *testing.T) {
	t.Parallel()

	service := &stubAccess{err: services.ErrSessionUnavailable}

	rec := runTestCase(t, NewAccess(service).ServeHTTP, pathRegistration, testCaseRequest{
		name: "sin sesión", method: http.MethodPost, body: `{"usuario":"root","password":"x"}`,
	})

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf(msgCode, rec.Code, http.StatusServiceUnavailable)
	}
}

// TestAccessStartWithoutMachine cubre la consola sin vínculo.
func TestAccessStartWithoutMachine(t *testing.T) {
	t.Parallel()

	service := &stubAccess{err: services.ErrNoMachine}

	rec := runTestCase(t, NewAccess(service).ServeHTTP, pathRegistration, testCaseRequest{
		name: "sin máquina", method: http.MethodPost, body: `{"usuario":"root","password":"x"}`,
	})

	if rec.Code != http.StatusConflict {
		t.Fatalf(msgCode, rec.Code, http.StatusConflict)
	}
}

// TestAccessValidacion cubre usuario, puerto y método.
func TestAccessValidacion(t *testing.T) {
	t.Parallel()

	handler := NewAccess(&stubAccess{}).ServeHTTP
	testCases := []testCaseRequest{
		{name: testCaseUserInvalid, method: http.MethodPost, body: `{"usuario":"Root","password":"x"}`, code: http.StatusBadRequest, field: userField},
		{name: testCaseBodyInvalid, method: http.MethodPost, body: `{`, code: http.StatusBadRequest},
		{name: testCaseMethodNotPermitido, method: http.MethodGet, code: http.StatusMethodNotAllowed},
	}

	for _, testCase := range testCases {
		rec := runTestCase(t, handler, pathRegistration, testCase)
		checkTestCase(t, rec, testCase)
	}
}

// TestAccessForget elimina las credenciales.
func TestAccessForget(t *testing.T) {
	t.Parallel()

	service := &stubAccess{}

	rec := runTestCase(t, NewAccess(service).ServeHTTP, pathRegistration, testCaseRequest{
		name: "olvidar", method: http.MethodDelete,
	})

	if rec.Code != http.StatusNoContent {
		t.Fatalf(msgCode, rec.Code, http.StatusNoContent)
	}

	if service.olvidados != 1 {
		t.Errorf("Olvidar() se llamó %d veces, want 1", service.olvidados)
	}
}

// TestAccessRecortaUser comprueba que los espacios del autocompletado
// no invalidan un usuario correcto.
func TestAccessRecortaUser(t *testing.T) {
	t.Parallel()

	service := &stubAccess{result: services.Outcome{Message: "Credenciales verificadas."}}

	rec := runTestCase(t, NewAccess(service).ServeHTTP, pathRegistration, testCaseRequest{
		name: "usuario con espacios", method: http.MethodPost, body: `{"usuario":" jorge ","password":"clave"}`,
	})

	if rec.Code != http.StatusCreated {
		t.Fatalf(msgCode, rec.Code, http.StatusCreated)
	}

	if service.user != userJorge {
		t.Errorf("usuario = %q, want jorge", service.user)
	}
}
