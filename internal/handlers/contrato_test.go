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

// TestDecodificarJSONRechazaCuerposInvalidos comprueba que la validación
// del cuerpo ocurre antes de llamar al servicio y que la respuesta es
// JSON incluso en los errores.
func TestDecodificarJSONRechazaCuerposInvalidos(t *testing.T) {
	t.Parallel()

	casos := []struct {
		nombre string
		cuerpo string
	}{
		{"campo desconocido", `{"nombre":"jorge","inesperado":true}`},
		{"cuerpo vacío", ""},
		{"cuerpo mal formado", `{`},
		{"tipo incorrecto", `{"nombre":123}`},
		{"cuerpo demasiado grande", `{"nombre":"` + strings.Repeat("a", maxCuerpo) + `"}`},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()

			servicio := &stubUsuarios{}

			rec := ejecutarCaso(t, NuevoUsuarios(servicio).Coleccion, rutaUsuarios, casoPeticion{
				nombre: caso.nombre, metodo: http.MethodPost, cuerpo: caso.cuerpo,
			})

			if rec.Code != http.StatusBadRequest {
				t.Fatalf(msgCodigoCuerpo, rec.Code, http.StatusBadRequest, rec.Body.String())
			}

			if servicio.creados != 0 {
				t.Errorf("Crear() se llamó %d veces, want 0 con un cuerpo inválido", servicio.creados)
			}

			if tipo := rec.Header().Get(cabeceraContentType); tipo != "application/json; charset=utf-8" {
				t.Errorf("%s = %q, want JSON", cabeceraContentType, tipo)
			}
		})
	}

	servicio := &stubUsuarios{}
	rec := ejecutarCaso(t, NuevoUsuarios(servicio).Coleccion, rutaUsuarios, casoPeticion{
		nombre: "cuerpo válido", metodo: http.MethodPost,
		cuerpo: `{"nombre":"jorge","sudo":true,"llaves":false}`,
	})

	if rec.Code != http.StatusCreated {
		t.Fatalf(msgCodigoCuerpo, rec.Code, http.StatusCreated, rec.Body.String())
	}

	if servicio.creados != 1 {
		t.Errorf("Crear() se llamó %d veces, want 1 con un cuerpo válido", servicio.creados)
	}
}

// TestColeccionesVaciasDevuelvenArreglo protege el contrato del frontend:
// una lista sin elementos debe serializarse como [] y no como null.
func TestColeccionesVaciasDevuelvenArreglo(t *testing.T) {
	t.Parallel()

	sistema := NuevoSistema(&stubPaquetes{}, &stubLlaves{}, &stubIssue{}, &stubHostname{})

	casos := []struct {
		nombre  string
		handler http.Handler
		ruta    string
		campo   string
	}{
		{"máquinas", NuevoMaquinas(&stubMaquinas{}), rutaMaquinas, "maquinas"},
		{"paquetes", http.HandlerFunc(sistema.Paquetes), rutaPaquetes, "paquetes"},
		{"usuarios", http.HandlerFunc(NuevoUsuarios(&stubUsuarios{}).Coleccion), rutaUsuarios, "usuarios"},
		{"unidades", http.HandlerFunc(NuevoSystemd(&stubSystemd{}).Unidades), rutaUnidadesPrueba, "unidades"},
		{"servidores", http.HandlerFunc(NuevoHaproxy(&stubHaproxy{}).Servidores), rutaServidores, "servidores"},
		{"historial", NuevoHistorial(&stubHistorial{}), rutaHistorialPrueba, "operaciones"},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()

			rec := httptest.NewRecorder()
			caso.handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(),
				http.MethodGet, caso.ruta, http.NoBody))

			if rec.Code != http.StatusOK {
				t.Fatalf(msgCodigoCuerpo, rec.Code, http.StatusOK, rec.Body.String())
			}

			if tipo := rec.Header().Get(cabeceraContentType); tipo != "application/json; charset=utf-8" {
				t.Errorf("%s = %q, want JSON", cabeceraContentType, tipo)
			}

			var cuerpo map[string]json.RawMessage
			if err := json.Unmarshal(rec.Body.Bytes(), &cuerpo); err != nil {
				t.Fatalf(msgJSONInvalido, err)
			}

			if obtenido := string(cuerpo[caso.campo]); obtenido != "[]" {
				t.Errorf("%s = %s, want []", caso.campo, obtenido)
			}
		})
	}
}

// TestMetodoNoPermitidoDeclaraAllow comprueba que un 405 incluye los
// métodos admitidos, que es lo que usa el frontend para explicar el error.
func TestMetodoNoPermitidoDeclaraAllow(t *testing.T) {
	t.Parallel()

	sistema := NuevoSistema(&stubPaquetes{}, &stubLlaves{}, &stubIssue{}, &stubHostname{})

	casos := []struct {
		nombre  string
		handler http.Handler
		ruta    string
		metodo  string
		allow   string
	}{
		{"paquetes", http.HandlerFunc(sistema.Paquetes), rutaPaquetes, http.MethodPut, "GET, POST"},
		{"paquete", http.HandlerFunc(sistema.Paquete), rutaPaqueteNano, http.MethodGet, "DELETE"},
		{"usuarios", http.HandlerFunc(NuevoUsuarios(&stubUsuarios{}).Coleccion), rutaUsuarios, http.MethodDelete, "GET, POST"},
		{"usuario", http.HandlerFunc(NuevoUsuarios(&stubUsuarios{}).Usuario), rutaUsuarioJorge, http.MethodPut, "GET, DELETE"},
		{"máquinas", NuevoMaquinas(&stubMaquinas{}), rutaMaquinas, http.MethodPost, "GET"},
		{"unidades", http.HandlerFunc(NuevoSystemd(&stubSystemd{}).Unidades), rutaUnidadesPrueba, http.MethodPost, "GET"},
		{"unidad", http.HandlerFunc(NuevoSystemd(&stubSystemd{}).Unidad), rutaUnidad, http.MethodDelete, "GET, PUT, PATCH"},
		{"servidores", http.HandlerFunc(NuevoHaproxy(&stubHaproxy{}).Servidores), rutaServidores, http.MethodPut, "GET, POST"},
		{"servidor", http.HandlerFunc(NuevoHaproxy(&stubHaproxy{}).Servidor), rutaServidor, http.MethodGet, "DELETE"},
		{"historial", NuevoHistorial(&stubHistorial{}), rutaHistorialPrueba, http.MethodPost, "GET"},
		{"diagnóstico", NuevoDiagnostico(&stubDiagnostico{}), rutaDiagnostico, http.MethodPost, "GET"},
		{"autenticación", http.HandlerFunc(NuevoAuth(&stubConsola{}).Estado), rutaAuthEstado, http.MethodPost, "GET"},
		{"acceso", NuevoAcceso(&stubAcceso{}), rutaRegistro, http.MethodGet, "POST, DELETE"},
		{"sesión", NuevoSesion(&stubDescubridor{}), rutaSesion, http.MethodPost, "GET, PUT"},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()

			rec := ejecutarCaso(t, caso.handler.ServeHTTP, caso.ruta, casoPeticion{
				nombre: caso.nombre, metodo: caso.metodo,
			})

			if rec.Code != http.StatusMethodNotAllowed {
				t.Fatalf(msgCodigoCuerpo, rec.Code, http.StatusMethodNotAllowed, rec.Body.String())
			}

			if allow := rec.Header().Get("Allow"); allow != caso.allow {
				t.Errorf("Allow = %q, want %q", allow, caso.allow)
			}
		})
	}
}

// TestCookieDeSesionTieneAtributosDeSeguridad comprueba que la cookie no
// es accesible desde JavaScript, dura lo que la sesión y se vence al
// salir.
func TestCookieDeSesionTieneAtributosDeSeguridad(t *testing.T) {
	t.Parallel()

	servicio := &stubConsola{autenticado: true, token: tokenPrueba}
	auth := NuevoAuth(servicio)

	rec := httptest.NewRecorder()
	auth.Entrar(rec, peticionConCookie(http.MethodPost, rutaAuthLogin, cuerpoCredenciales, ""))

	if rec.Code != http.StatusOK {
		t.Fatalf("Entrar() estado = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	cookie := cookieDe(t, rec, nombreCookieSesion)
	if cookie.Value != tokenPrueba {
		t.Errorf("cookie.Value = %q, want %q", cookie.Value, tokenPrueba)
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

	if cookie.MaxAge != int(services.VigenciaSesionConsola.Seconds()) {
		t.Errorf("cookie.MaxAge = %d, want %d", cookie.MaxAge, int(services.VigenciaSesionConsola.Seconds()))
	}

	salida := httptest.NewRecorder()
	auth.Salir(salida, peticionConCookie(http.MethodPost, "/api/auth/logout", "", tokenPrueba))

	cookie = cookieDe(t, salida, nombreCookieSesion)
	if cookie.MaxAge >= 0 {
		t.Errorf("cookie.MaxAge al salir = %d, want vencida", cookie.MaxAge)
	}

	if cookie.Value != "" {
		t.Errorf("cookie.Value al salir = %q, want vacío", cookie.Value)
	}
}

// cookieDe devuelve la cookie indicada de una respuesta grabada.
func cookieDe(t *testing.T, rec *httptest.ResponseRecorder, nombre string) *http.Cookie {
	t.Helper()

	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == nombre {
			return cookie
		}
	}

	t.Fatalf("la respuesta no incluyó la cookie %q", nombre)

	return nil
}
