// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"consola-vm/internal/services"
)

// TestDecodeJSONRejectsBodiesInvalid comprueba que la validación
// del cuerpo ocurre antes de llamar al servicio y que la respuesta es
// JSON incluso en los errores.
func TestDecodeJSONRejectsBodiesInvalid(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		body string
	}{
		{"campo desconocido", `{"nombre":"jorge","inesperado":true}`},
		{"cuerpo vacío", ""},
		{"cuerpo mal formado", `{`},
		{"tipo incorrecto", `{"nombre":123}`},
		{"cuerpo demasiado grande", `{"nombre":"` + strings.Repeat("a", maxBody) + `"}`},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := &stubUsers{}

			rec := runTestCase(t, NewUsers(service).Collection, pathUsers, testCaseRequest{
				name: testCase.name, method: http.MethodPost, body: testCase.body,
			})

			if rec.Code != http.StatusBadRequest {
				t.Fatalf(msgCodeBody, rec.Code, http.StatusBadRequest, rec.Body.String())
			}

			if service.created != 0 {
				t.Errorf("Crear() se llamó %d veces, want 0 con un cuerpo inválido", service.created)
			}

			if kind := rec.Header().Get(headerContentType); kind != "application/json; charset=utf-8" {
				t.Errorf("%s = %q, want JSON", headerContentType, kind)
			}
		})
	}

	service := &stubUsers{}
	rec := runTestCase(t, NewUsers(service).Collection, pathUsers, testCaseRequest{
		name: "cuerpo válido", method: http.MethodPost,
		body: `{"nombre":"jorge","sudo":true,"llaves":false}`,
	})

	if rec.Code != http.StatusCreated {
		t.Fatalf(msgCodeBody, rec.Code, http.StatusCreated, rec.Body.String())
	}

	if service.created != 1 {
		t.Errorf("Crear() se llamó %d veces, want 1 con un cuerpo válido", service.created)
	}
}

// TestColeccionesVaciasDevuelvenArray protege el contrato del frontend:
// una lista sin elementos debe serializarse como [] y no como null.
func TestColeccionesVaciasDevuelvenArray(t *testing.T) {
	t.Parallel()

	system := NewSystem(&stubPackages{}, &stubKeys{}, &stubIssue{}, &stubHostname{})

	testCases := []struct {
		name    string
		handler http.Handler
		path    string
		field   string
	}{
		{"máquinas", NewMachines(&stubMachines{}), pathMachines, "maquinas"},
		{"paquetes", http.HandlerFunc(system.Packages), pathPackages, "paquetes"},
		{"usuarios", http.HandlerFunc(NewUsers(&stubUsers{}).Collection), pathUsers, "usuarios"},
		{"unidades", http.HandlerFunc(NewSystemd(&stubSystemd{}).Units), pathUnitsTests, "unidades"},
		{"servidores", http.HandlerFunc(NewHAProxy(&stubHAProxy{}).Servers), pathServers, "servidores"},
		{"historial", NewHistory(&stubHistory{}), pathHistoryTests, "operaciones"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			rec := httptest.NewRecorder()
			testCase.handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(),
				http.MethodGet, testCase.path, http.NoBody))

			if rec.Code != http.StatusOK {
				t.Fatalf(msgCodeBody, rec.Code, http.StatusOK, rec.Body.String())
			}

			if kind := rec.Header().Get(headerContentType); kind != "application/json; charset=utf-8" {
				t.Errorf("%s = %q, want JSON", headerContentType, kind)
			}

			var body map[string]json.RawMessage
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf(msgJSONInvalid, err)
			}

			if obtained := string(body[testCase.field]); obtained != "[]" {
				t.Errorf("%s = %s, want []", testCase.field, obtained)
			}
		})
	}
}

// TestMethodNotPermitidoDeclaraAllow comprueba que un 405 incluye los
// métodos admitidos, que es lo que usa el frontend para explicar el error.
func TestMethodNotPermitidoDeclaraAllow(t *testing.T) {
	t.Parallel()

	system := NewSystem(&stubPackages{}, &stubKeys{}, &stubIssue{}, &stubHostname{})

	testCases := []struct {
		name    string
		handler http.Handler
		path    string
		method  string
		allow   string
	}{
		{"paquetes", http.HandlerFunc(system.Packages), pathPackages, http.MethodPut, "GET, POST"},
		{"paquete", http.HandlerFunc(system.Package), pathPkgNano, http.MethodGet, "DELETE"},
		{"usuarios", http.HandlerFunc(NewUsers(&stubUsers{}).Collection), pathUsers, http.MethodDelete, "GET, POST"},
		{"usuario", http.HandlerFunc(NewUsers(&stubUsers{}).User), pathUserJorge, http.MethodPut, "GET, DELETE"},
		{"máquinas", NewMachines(&stubMachines{}), pathMachines, http.MethodPost, "GET"},
		{"unidades", http.HandlerFunc(NewSystemd(&stubSystemd{}).Units), pathUnitsTests, http.MethodPost, "GET"},
		{"unidad", http.HandlerFunc(NewSystemd(&stubSystemd{}).Unit), pathUnit, http.MethodDelete, "GET, PUT, PATCH"},
		{"servidores", http.HandlerFunc(NewHAProxy(&stubHAProxy{}).Servers), pathServers, http.MethodPut, "GET, POST"},
		{"servidor", http.HandlerFunc(NewHAProxy(&stubHAProxy{}).Server), pathServer, http.MethodGet, "DELETE"},
		{"historial", NewHistory(&stubHistory{}), pathHistoryTests, http.MethodPost, "GET"},
		{"diagnóstico", NewDiagnostics(&stubDiagnostics{}), pathDiagnostics, http.MethodPost, "GET"},
		{"autenticación", http.HandlerFunc(NewAuth(&stubConsole{}).State), pathAuthState, http.MethodPost, "GET"},
		{"acceso", NewAccess(&stubAccess{}), pathRegistration, http.MethodGet, "POST, DELETE"},
		{"sesión", NewSession(&stubDiscoverer{}), pathSession, http.MethodPost, "GET, PUT"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			rec := runTestCase(t, testCase.handler.ServeHTTP, testCase.path, testCaseRequest{
				name: testCase.name, method: testCase.method,
			})

			if rec.Code != http.StatusMethodNotAllowed {
				t.Fatalf(msgCodeBody, rec.Code, http.StatusMethodNotAllowed, rec.Body.String())
			}

			if allow := rec.Header().Get("Allow"); allow != testCase.allow {
				t.Errorf("Allow = %q, want %q", allow, testCase.allow)
			}
		})
	}
}

// TestCookieOfSessionTieneAtributosOfSeguridad comprueba que la cookie no
// es accesible desde JavaScript, dura lo que la sesión y se vence al
// salir.
func TestCookieOfSessionTieneAtributosOfSeguridad(t *testing.T) {
	t.Parallel()

	service := &stubConsole{autenticado: true, token: tokenTests}
	auth := NewAuth(service)

	rec := httptest.NewRecorder()
	auth.Login(rec, requestWithCookie(http.MethodPost, pathAuthLogin, bodyCredentials, ""))

	if rec.Code != http.StatusOK {
		t.Fatalf("Entrar() estado = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	cookie := cookieOf(t, rec, sessionCookieName)
	if cookie.Value != tokenTests {
		t.Errorf("cookie.Value = %q, want %q", cookie.Value, tokenTests)
	}

	if !cookie.HttpOnly {
		t.Error("cookie.HttpOnly = false, want true")
	}

	if cookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("cookie.SameSite = %v, want Lax", cookie.SameSite)
	}

	if cookie.Path != "/" {
		t.Errorf("cookie.Path = %q, want /", cookie.Path)
	}

	if cookie.MaxAge != int(services.SessionLifetime.Seconds()) {
		t.Errorf("cookie.MaxAge = %d, want %d", cookie.MaxAge, int(services.SessionLifetime.Seconds()))
	}

	output := httptest.NewRecorder()
	auth.Logout(output, requestWithCookie(http.MethodPost, "/api/auth/logout", "", tokenTests))

	cookie = cookieOf(t, output, sessionCookieName)
	if cookie.MaxAge >= 0 {
		t.Errorf("cookie.MaxAge al salir = %d, want vencida", cookie.MaxAge)
	}

	if cookie.Value != "" {
		t.Errorf("cookie.Value al salir = %q, want vacío", cookie.Value)
	}
}

// cookieOf devuelve la cookie indicada de una respuesta grabada.
func cookieOf(t *testing.T, rec *httptest.ResponseRecorder, name string) *http.Cookie {
	t.Helper()

	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}

	t.Fatalf("la respuesta no incluyó la cookie %q", name)

	return nil
}
