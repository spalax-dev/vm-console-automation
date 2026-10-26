// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"consola-vm/internal/services"
)

const (
	pathAuthState   = "/api/auth/estado"
	pathAuthLogin   = "/api/auth/login"
	tokenTests      = "token-1"
	headerSetCookie = "Set-Cookie"
	bodyCredentials = `{"usuario":"jorge","password":"una-clave-larga"}` // #nosec G101 -- credenciales de prueba.
)

// stubConsole sustituye al servicio de autenticación en las pruebas.
type stubConsole struct {
	configurada     bool
	configureErr    error
	autenticado     bool
	authenticateErr error
	token           string
	tokenErr        error
	user            string
	validates       bool
	registered      []string
}

// IsConfigured informa el estado preprogramado.
func (s *stubConsole) IsConfigured() (bool, error) { return s.configurada, s.configureErr }

// Configure registra la cuenta recibida.
func (s *stubConsole) Configure(_ context.Context, user, password string) error {
	s.registered = append(s.registered, user+"|"+password)

	return s.configureErr
}

// Authenticate devuelve el resultado preprogramado.
func (s *stubConsole) Authenticate(_ context.Context, user, _ string) (bool, error) {
	s.registered = append(s.registered, user)

	return s.autenticado, s.authenticateErr
}

// OpenSession devuelve el token preprogramado.
func (s *stubConsole) OpenSession(user string) (string, error) {
	s.user = user

	return s.token, s.tokenErr
}

// Validate devuelve el resultado preprogramado.
func (s *stubConsole) Validate(token string) (string, bool) {
	if !s.validates || token != s.token {
		return "", false
	}

	return s.user, true
}

// requestWithCookie arma una petición con la cookie de sesión.
func requestWithCookie(method, path, body, token string) *http.Request {
	var reader *strings.Reader

	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}

	request := httptest.NewRequestWithContext(context.Background(), method, path, reader)

	if token != "" {
		request.AddCookie(&http.Cookie{ // #nosec G124 -- cookie de prueba sobre HTTP.
			Name: sessionCookieName, Value: token,
		})
	}

	return request
}

// TestAuthState comprueba el informe de estado sin autenticación.
func TestAuthState(t *testing.T) {
	t.Parallel()

	service := &stubConsole{configurada: true}
	auth := NewAuth(service)

	rec := httptest.NewRecorder()
	auth.State(rec, requestWithCookie(http.MethodGet, pathAuthState, "", ""))

	if rec.Code != http.StatusOK {
		t.Fatalf("Estado() estado = %d, want 200", rec.Code)
	}

	var body struct {
		IsConfigured bool   `json:"configurada"`
		Autenticado  bool   `json:"autenticado"`
		User         string `json:"usuario"`
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("Estado() cuerpo inválido: %v", err)
	}

	if !body.IsConfigured || body.Autenticado {
		t.Errorf("Estado() = %+v, want configurada y sin sesión", body)
	}

	rec = httptest.NewRecorder()
	service.token = tokenTests
	service.validates = true
	service.user = userJorge

	auth.State(rec, requestWithCookie(http.MethodGet, pathAuthState, "", tokenTests))

	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("Estado() cuerpo inválido: %v", err)
	}

	if !body.Autenticado || body.User != userJorge {
		t.Errorf("Estado() = %+v, want autenticado como jorge", body)
	}

	rec = httptest.NewRecorder()
	auth.State(rec, requestWithCookie(http.MethodPost, pathAuthState, "", ""))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Estado() con POST = %d, want 405", rec.Code)
	}
}

// TestAuthSetup crea la cuenta y entrega la cookie.
func TestAuthSetup(t *testing.T) {
	t.Parallel()

	service := &stubConsole{token: tokenTests}
	auth := NewAuth(service)

	rec := httptest.NewRecorder()
	auth.Configure(rec, requestWithCookie(http.MethodPost, "/api/auth/setup",
		bodyCredentials, ""))

	if rec.Code != http.StatusCreated {
		t.Fatalf("Configurar() estado = %d, want 201: %s", rec.Code, rec.Body.String())
	}

	if !strings.Contains(rec.Header().Get(headerSetCookie), sessionCookieName+"=token-1") {
		t.Errorf("Configurar() cookie = %q, want la cookie de sesión",
			rec.Header().Get(headerSetCookie))
	}
}

// TestAuthSetupRejects comprueba los errores de la creación de la cuenta.
func TestAuthSetupRejects(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		service *stubConsole
		body    string
		state   int
	}{
		{
			"ya configurada",
			&stubConsole{configureErr: services.ErrConsoleConfigured},
			bodyCredentials,
			http.StatusConflict,
		},
		{
			"usuario inválido",
			&stubConsole{},
			`{"usuario":"Jorge","password":"una-clave-larga"}`,
			http.StatusBadRequest,
		},
		{
			"contraseña corta",
			&stubConsole{},
			`{"usuario":"jorge","password":"corta"}`,
			http.StatusBadRequest,
		},
		{
			"json inválido",
			&stubConsole{},
			`{`,
			http.StatusBadRequest,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			rec := httptest.NewRecorder()
			NewAuth(testCase.service).Configure(rec,
				requestWithCookie(http.MethodPost, "/api/auth/setup", testCase.body, ""))

			if rec.Code != testCase.state {
				t.Errorf("Configurar() estado = %d, want %d: %s",
					rec.Code, testCase.state, rec.Body.String())
			}
		})
	}
}

// TestAuthLogin comprueba el acceso con credenciales válidas e inválidas.
func TestAuthLogin(t *testing.T) {
	t.Parallel()

	correct := &stubConsole{autenticado: true, token: tokenTests}
	rec := httptest.NewRecorder()
	NewAuth(correct).Login(rec, requestWithCookie(http.MethodPost, pathAuthLogin,
		bodyCredentials, ""))

	if rec.Code != http.StatusOK {
		t.Fatalf("Entrar() estado = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	if !strings.Contains(rec.Header().Get(headerSetCookie), sessionCookieName+"=token-1") {
		t.Errorf("Entrar() no instaló la cookie")
	}

	incorrect := &stubConsole{autenticado: false}
	rec = httptest.NewRecorder()
	NewAuth(incorrect).Login(rec, requestWithCookie(http.MethodPost, pathAuthLogin,
		`{"usuario":"jorge","password":"otra-clave"}`, ""))

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Entrar() estado = %d, want 401", rec.Code)
	}
}

// TestAuthLogout vence la cookie de sesión.
func TestAuthLogout(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	NewAuth(&stubConsole{token: tokenTests, validates: true}).Logout(rec,
		requestWithCookie(http.MethodPost, "/api/auth/logout", "", tokenTests))

	if rec.Code != http.StatusNoContent {
		t.Fatalf("Salir() estado = %d, want 204", rec.Code)
	}

	if cookie := rec.Header().Get(headerSetCookie); !strings.Contains(cookie, "Max-Age=0") {
		t.Errorf("Salir() cookie = %q, want vencida", cookie)
	}
}

// TestWithAutenticacion protege la API y deja pasar el acceso y los
// archivos estáticos.
func TestWithAutenticacion(t *testing.T) {
	t.Parallel()

	service := &stubConsole{token: tokenTests, validates: true, user: "jorge"}
	handler := WithAuthentication(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if user := UserFrom(r.Context()); user != "" {
			w.Header().Set("X-Usuario", user)
		}

		w.WriteHeader(http.StatusNoContent)
	}), service)

	testCases := []struct {
		name  string
		path  string
		token string
		state int
	}{
		{"api sin cookie", pathPackages, "", http.StatusUnauthorized},
		{"api con cookie válida", pathPackages, tokenTests, http.StatusNoContent},
		{"api con cookie inválida", pathPackages, "otra", http.StatusUnauthorized},
		{"acceso", pathAuthLogin, "", http.StatusNoContent},
		{"interfaz", "/index.html", "", http.StatusNoContent},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, requestWithCookie(http.MethodGet, testCase.path, "", testCase.token))

			if rec.Code != testCase.state {
				t.Errorf("ConAutenticacion() estado = %d, want %d", rec.Code, testCase.state)
			}
		})
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, requestWithCookie(http.MethodGet, pathPackages, "", tokenTests))

	if rec.Code != http.StatusNoContent {
		t.Fatalf("ConAutenticacion() estado = %d, want 204", rec.Code)
	}

	if user := rec.Header().Get("X-Usuario"); user != userJorge {
		t.Errorf("ConAutenticacion() usuario del contexto = %q, want %q", user, userJorge)
	}
}
