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
	rutaAuthEstado     = "/api/auth/estado"
	rutaAuthLogin      = "/api/auth/login"
	tokenPrueba        = "token-1"
	cabeceraSetCookie  = "Set-Cookie"
	cuerpoCredenciales = `{"usuario":"jorge","password":"una-clave-larga"}` // #nosec G101 -- credenciales de prueba.
)

// stubConsola sustituye al servicio de autenticación en las pruebas.
type stubConsola struct {
	configurada   bool
	configurarErr error
	autenticado   bool
	autenticarErr error
	token         string
	tokenErr      error
	usuario       string
	valida        bool
	registrados   []string
}

// Configurada informa el estado preprogramado.
func (s *stubConsola) Configurada() (bool, error) { return s.configurada, s.configurarErr }

// Configurar registra la cuenta recibida.
func (s *stubConsola) Configurar(_ context.Context, usuario, password string) error {
	s.registrados = append(s.registrados, usuario+"|"+password)

	return s.configurarErr
}

// Autenticar devuelve el resultado preprogramado.
func (s *stubConsola) Autenticar(_ context.Context, usuario, _ string) (bool, error) {
	s.registrados = append(s.registrados, usuario)

	return s.autenticado, s.autenticarErr
}

// AbrirSesion devuelve el token preprogramado.
func (s *stubConsola) AbrirSesion(usuario string) (string, error) {
	s.usuario = usuario

	return s.token, s.tokenErr
}

// Validar devuelve el resultado preprogramado.
func (s *stubConsola) Validar(token string) (string, bool) {
	if !s.valida || token != s.token {
		return "", false
	}

	return s.usuario, true
}

// peticionConCookie arma una petición con la cookie de sesión.
func peticionConCookie(metodo, ruta, cuerpo, token string) *http.Request {
	var lector *strings.Reader

	if cuerpo == "" {
		lector = strings.NewReader("")
	} else {
		lector = strings.NewReader(cuerpo)
	}

	peticion := httptest.NewRequestWithContext(context.Background(), metodo, ruta, lector)

	if token != "" {
		peticion.AddCookie(&http.Cookie{ // #nosec G124 -- cookie de prueba sobre HTTP.
			Name: nombreCookieSesion, Value: token,
		})
	}

	return peticion
}

// TestAuthEstado comprueba el informe de estado sin autenticación.
func TestAuthEstado(t *testing.T) {
	t.Parallel()

	servicio := &stubConsola{configurada: true}
	auth := NuevoAuth(servicio)

	rec := httptest.NewRecorder()
	auth.Estado(rec, peticionConCookie(http.MethodGet, rutaAuthEstado, "", ""))

	if rec.Code != http.StatusOK {
		t.Fatalf("Estado() estado = %d, want 200", rec.Code)
	}

	var cuerpo struct {
		Configurada bool   `json:"configurada"`
		Autenticado bool   `json:"autenticado"`
		Usuario     string `json:"usuario"`
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &cuerpo); err != nil {
		t.Fatalf("Estado() cuerpo inválido: %v", err)
	}

	if !cuerpo.Configurada || cuerpo.Autenticado {
		t.Errorf("Estado() = %+v, want configurada y sin sesión", cuerpo)
	}

	rec = httptest.NewRecorder()
	servicio.token = tokenPrueba
	servicio.valida = true
	servicio.usuario = usuarioJorge

	auth.Estado(rec, peticionConCookie(http.MethodGet, rutaAuthEstado, "", tokenPrueba))

	if err := json.Unmarshal(rec.Body.Bytes(), &cuerpo); err != nil {
		t.Fatalf("Estado() cuerpo inválido: %v", err)
	}

	if !cuerpo.Autenticado || cuerpo.Usuario != usuarioJorge {
		t.Errorf("Estado() = %+v, want autenticado como jorge", cuerpo)
	}

	rec = httptest.NewRecorder()
	auth.Estado(rec, peticionConCookie(http.MethodPost, rutaAuthEstado, "", ""))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Estado() con POST = %d, want 405", rec.Code)
	}
}

// TestAuthSetup crea la cuenta y entrega la cookie.
func TestAuthSetup(t *testing.T) {
	t.Parallel()

	servicio := &stubConsola{token: tokenPrueba}
	auth := NuevoAuth(servicio)

	rec := httptest.NewRecorder()
	auth.Configurar(rec, peticionConCookie(http.MethodPost, "/api/auth/setup",
		cuerpoCredenciales, ""))

	if rec.Code != http.StatusCreated {
		t.Fatalf("Configurar() estado = %d, want 201: %s", rec.Code, rec.Body.String())
	}

	if !strings.Contains(rec.Header().Get(cabeceraSetCookie), nombreCookieSesion+"=token-1") {
		t.Errorf("Configurar() cookie = %q, want la cookie de sesión",
			rec.Header().Get(cabeceraSetCookie))
	}
}

// TestAuthSetupRechaza comprueba los errores de la creación de la cuenta.
func TestAuthSetupRechaza(t *testing.T) {
	t.Parallel()

	casos := []struct {
		nombre   string
		servicio *stubConsola
		cuerpo   string
		estado   int
	}{
		{
			"ya configurada",
			&stubConsola{configurarErr: services.ErrConsolaConfigurada},
			cuerpoCredenciales,
			http.StatusConflict,
		},
		{
			"usuario inválido",
			&stubConsola{},
			`{"usuario":"Jorge","password":"una-clave-larga"}`,
			http.StatusBadRequest,
		},
		{
			"contraseña corta",
			&stubConsola{},
			`{"usuario":"jorge","password":"corta"}`,
			http.StatusBadRequest,
		},
		{
			"json inválido",
			&stubConsola{},
			`{`,
			http.StatusBadRequest,
		},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()

			rec := httptest.NewRecorder()
			NuevoAuth(caso.servicio).Configurar(rec,
				peticionConCookie(http.MethodPost, "/api/auth/setup", caso.cuerpo, ""))

			if rec.Code != caso.estado {
				t.Errorf("Configurar() estado = %d, want %d: %s",
					rec.Code, caso.estado, rec.Body.String())
			}
		})
	}
}

// TestAuthLogin comprueba el acceso con credenciales válidas e inválidas.
func TestAuthLogin(t *testing.T) {
	t.Parallel()

	correcto := &stubConsola{autenticado: true, token: tokenPrueba}
	rec := httptest.NewRecorder()
	NuevoAuth(correcto).Entrar(rec, peticionConCookie(http.MethodPost, rutaAuthLogin,
		cuerpoCredenciales, ""))

	if rec.Code != http.StatusOK {
		t.Fatalf("Entrar() estado = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	if !strings.Contains(rec.Header().Get(cabeceraSetCookie), nombreCookieSesion+"=token-1") {
		t.Errorf("Entrar() no instaló la cookie")
	}

	incorrecto := &stubConsola{autenticado: false}
	rec = httptest.NewRecorder()
	NuevoAuth(incorrecto).Entrar(rec, peticionConCookie(http.MethodPost, rutaAuthLogin,
		`{"usuario":"jorge","password":"otra-clave"}`, ""))

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Entrar() estado = %d, want 401", rec.Code)
	}
}

// TestAuthSalir vence la cookie de sesión.
func TestAuthSalir(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	NuevoAuth(&stubConsola{token: tokenPrueba, valida: true}).Salir(rec,
		peticionConCookie(http.MethodPost, "/api/auth/logout", "", tokenPrueba))

	if rec.Code != http.StatusNoContent {
		t.Fatalf("Salir() estado = %d, want 204", rec.Code)
	}

	if cookie := rec.Header().Get(cabeceraSetCookie); !strings.Contains(cookie, "Max-Age=0") {
		t.Errorf("Salir() cookie = %q, want vencida", cookie)
	}
}

// TestConAutenticacion protege la API y deja pasar el acceso y los
// archivos estáticos.
func TestConAutenticacion(t *testing.T) {
	t.Parallel()

	servicio := &stubConsola{token: tokenPrueba, valida: true, usuario: "jorge"}
	manejador := ConAutenticacion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if usuario := UsuarioDe(r.Context()); usuario != "" {
			w.Header().Set("X-Usuario", usuario)
		}

		w.WriteHeader(http.StatusNoContent)
	}), servicio)

	casos := []struct {
		nombre string
		ruta   string
		token  string
		estado int
	}{
		{"api sin cookie", rutaPaquetes, "", http.StatusUnauthorized},
		{"api con cookie válida", rutaPaquetes, tokenPrueba, http.StatusNoContent},
		{"api con cookie inválida", rutaPaquetes, "otra", http.StatusUnauthorized},
		{"acceso", rutaAuthLogin, "", http.StatusNoContent},
		{"interfaz", "/index.html", "", http.StatusNoContent},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()

			rec := httptest.NewRecorder()
			manejador.ServeHTTP(rec, peticionConCookie(http.MethodGet, caso.ruta, "", caso.token))

			if rec.Code != caso.estado {
				t.Errorf("ConAutenticacion() estado = %d, want %d", rec.Code, caso.estado)
			}
		})
	}

	rec := httptest.NewRecorder()
	manejador.ServeHTTP(rec, peticionConCookie(http.MethodGet, rutaPaquetes, "", tokenPrueba))

	if rec.Code != http.StatusNoContent {
		t.Fatalf("ConAutenticacion() estado = %d, want 204", rec.Code)
	}
}
