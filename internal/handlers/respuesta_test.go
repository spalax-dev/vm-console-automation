// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"consola-vm/internal/core/domain"
	"consola-vm/internal/services"
)

// errFailure simula un error genérico sin comando asociado.
var errFailure = errors.New("fallo interno")

// Órdenes usadas en las pruebas del cuerpo de error.
const (
	operationCreateJorge  = "useradd -m jorge"
	operationGuestControl = "VBoxManage guestcontrol ejecutar"
)

// bodyOfResponse decodifica el cuerpo JSON de una respuesta grabada.
func bodyOfResponse(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf(msgJSONInvalid, err)
	}

	return body
}

// checkWithoutKeys verifica que el cuerpo JSON no incluya las claves
// dadas, para que un error no emita datos de una operación que no ocurrió.
func checkWithoutKeys(t *testing.T, body map[string]any, keys ...string) {
	t.Helper()

	for _, key := range keys {
		if value, exists := body[key]; exists {
			t.Errorf("la respuesta no debe incluir %q y vale %v", key, value)
		}
	}
}

// TestResponseValidacionOmiteDataOfOperation comprueba que un 400 de
// validación nombre el campo rechazado y no invente datos de operación.
func TestResponseValidacionOmiteDataOfOperation(t *testing.T) {
	t.Parallel()

	rec := runTestCase(t, NewUsers(&stubUsers{}).Collection, pathUsers, testCaseRequest{
		name: "nombre inválido", method: http.MethodPost,
		body: `{"nombre":"-mal-","sudo":false,"llaves":false}`,
	})

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(msgCodeBody, rec.Code, http.StatusBadRequest, rec.Body.String())
	}

	body := bodyOfResponse(t, rec)

	if body["campo"] != nameField {
		t.Errorf("campo = %v, want %q", body["campo"], nameField)
	}

	checkWithoutKeys(t, body, "operacion", "codigo", "salida", "stderr")
}

// TestResponseAutenticacionWithoutCode comprueba que los rechazos de la
// autenticación no emitan un código de salida inventado.
func TestResponseAutenticacionWithoutCode(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		service  *stubConsole
		handler  func(*Auth) http.HandlerFunc
		expected int
	}{
		{
			name:     "credenciales incorrectas",
			service:  &stubConsole{autenticado: false},
			handler:  func(a *Auth) http.HandlerFunc { return a.Login },
			expected: http.StatusUnauthorized,
		},
		{
			name:     "cuenta existente",
			service:  &stubConsole{configureErr: services.ErrConsoleConfigured},
			handler:  func(a *Auth) http.HandlerFunc { return a.Configure },
			expected: http.StatusConflict,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			rec := runTestCase(t, testCase.handler(NewAuth(testCase.service)), pathAuthLogin, testCaseRequest{
				name: testCase.name, method: http.MethodPost, body: bodyCredentials,
			})

			if rec.Code != testCase.expected {
				t.Fatalf(msgCodeBody, rec.Code, testCase.expected, rec.Body.String())
			}

			body := bodyOfResponse(t, rec)

			if body["ok"] != false {
				t.Errorf("ok = %v, want false", body["ok"])
			}

			if body["mensaje"] == "" || body["mensaje"] == nil {
				t.Errorf("la respuesta debe incluir un mensaje")
			}

			checkWithoutKeys(t, body, "codigo", "operacion", "salida", "stderr")
		})
	}
}

// TestResponseErrorOfOperationKeepsCommand comprueba que el fallo de un
// comando conserve la operación, el código, la salida y el detalle técnico.
func TestResponseErrorOfOperationKeepsCommand(t *testing.T) {
	t.Parallel()

	service := &stubUsers{
		result: services.Outcome{
			Operation: operationCreateJorge,
			Code:      9,
			Output:    "creando la cuenta",
			Stderr:    "useradd: el usuario ya existe",
		},
		err: &domain.ExitError{Code: 9, Command: operationCreateJorge},
	}

	rec := runTestCase(t, NewUsers(service).Collection, pathUsers, testCaseRequest{
		name: "fallo de useradd", method: http.MethodPost,
		body: `{"nombre":"jorge","sudo":true,"llaves":false}`,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCodeBody, rec.Code, http.StatusInternalServerError, rec.Body.String())
	}

	body := bodyOfResponse(t, rec)

	if body["ok"] != false {
		t.Errorf("ok = %v, want false", body["ok"])
	}

	if body["mensaje"] != "No fue posible completar la operación." {
		t.Errorf("mensaje = %v, want el mensaje genérico de la operación", body["mensaje"])
	}

	if body["operacion"] != operationCreateJorge {
		t.Errorf("operacion = %v, want %q", body["operacion"], operationCreateJorge)
	}

	if body["codigo"] != float64(9) {
		t.Errorf("codigo = %v, want 9", body["codigo"])
	}

	if body["salida"] != "creando la cuenta" {
		t.Errorf("salida = %v, want la salida del comando", body["salida"])
	}

	if body["stderr"] != "useradd: el usuario ya existe" {
		t.Errorf("stderr = %v, want el stderr del comando", body["stderr"])
	}

	if technical, ok := body["tecnico"].(string); !ok || technical == "" {
		t.Errorf("la respuesta debe incluir el detalle técnico, tecnico = %v", body["tecnico"])
	}

	checkWithoutKeys(t, body, "campo")
}

// TestResponseCredentialsInvalidKeepsCode comprueba que un 401
// con un comando ejecutado sí incluya su código de salida.
func TestResponseCredentialsInvalidKeepsCode(t *testing.T) {
	t.Parallel()

	service := &stubAccess{
		result: services.Outcome{
			Operation: operationGuestControl,
			Code:      5,
			Stderr:    "VBoxManage: error",
		},
		err: services.ErrInvalidCredentials,
	}

	rec := runTestCase(t, NewAccess(service).ServeHTTP, pathRegistration, testCaseRequest{
		name: "credenciales rechazadas", method: http.MethodPost,
		body: `{"usuario":"root","password":"secreto"}`,
	})

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf(msgCodeBody, rec.Code, http.StatusUnauthorized, rec.Body.String())
	}

	body := bodyOfResponse(t, rec)

	if body["operacion"] != operationGuestControl {
		t.Errorf("operacion = %v, want %q", body["operacion"], operationGuestControl)
	}

	if body["codigo"] != float64(5) {
		t.Errorf("codigo = %v, want 5", body["codigo"])
	}

	if body["stderr"] != "VBoxManage: error" {
		t.Errorf("stderr = %v, want el stderr del comando", body["stderr"])
	}
}

// TestResponseOfErrorOmiteOperationEmpty comprueba que un resultado sin
// comando no aporte campos a la respuesta.
func TestResponseOfErrorOmiteOperationEmpty(t *testing.T) {
	t.Parallel()

	body := responseFromError(services.Outcome{}, "no fue posible", errFailure)

	if body.OK {
		t.Errorf("OK = true, want false")
	}

	if body.Message != "no fue posible" {
		t.Errorf("Mensaje = %q, want %q", body.Message, "no fue posible")
	}

	if body.Technical != errFailure.Error() {
		t.Errorf("Tecnico = %q, want %q", body.Technical, errFailure.Error())
	}

	if body.Operation != "" || body.Code != 0 || body.Output != "" || body.Stderr != "" {
		t.Errorf("el resultado vacío no debe aportar datos de la operación: %+v", body)
	}
}
