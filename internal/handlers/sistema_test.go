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
)

// Valores repetidos en las pruebas de los handlers.
const (
	casoValido            = "válido"
	casoMetodoNoPermitido = "método no permitido"
	casoCuerpoInvalido    = "cuerpo inválido"
	unidadCatalogo        = "catalogo.service"
	casoListar            = "listar"
	casoConsultar         = "consultar"
	casoEliminar          = "eliminar"
	casoNombreInvalido    = "nombre inválido"
	usuarioJorge          = "jorge"
	usuarioJorgeMayus     = "Jorge"
	rutaUsuarioJorge      = "/api/users/jorge"
	nombreNano            = "nano"
	servidorPrueba        = "192.168.1.61:8080"
)

// casoPeticion describe una petición de prueba contra un handler.
type casoPeticion struct {
	// nombre identifica el caso en los mensajes de error.
	nombre string
	// metodo es el método HTTP de la petición.
	metodo string
	// cuerpo es el JSON enviado, vacío si no aplica.
	cuerpo string
	// codigo es el estado HTTP esperado.
	codigo int
	// campo es el campo de validación esperado en un 400, si aplica.
	campo string
}

// ejecutarCaso lanza una petición contra el handler y devuelve la
// respuesta grabada.
func ejecutarCaso(t *testing.T, handler http.HandlerFunc, ruta string, caso casoPeticion) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequestWithContext(context.Background(), caso.metodo, ruta, strings.NewReader(caso.cuerpo))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler(rec, req)

	return rec
}

// ejecutarCasoConPath lanza una petición con un parámetro de ruta.
func ejecutarCasoConPath(
	t *testing.T, handler http.HandlerFunc, ruta, clave, valor string, caso casoPeticion,
) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequestWithContext(context.Background(), caso.metodo, ruta, strings.NewReader(caso.cuerpo))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue(clave, valor)

	rec := httptest.NewRecorder()
	handler(rec, req)

	return rec
}

// comprobarCaso valida el código HTTP y, en los errores de validación,
// que el cuerpo nombre el campo rechazado.
func comprobarCaso(t *testing.T, rec *httptest.ResponseRecorder, caso casoPeticion) {
	t.Helper()

	if rec.Code != caso.codigo {
		t.Fatalf("%s: código = %d, want %d (cuerpo %s)", caso.nombre, rec.Code, caso.codigo, rec.Body.String())
	}

	if caso.campo == "" {
		return
	}

	var cuerpo map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &cuerpo); err != nil {
		t.Fatalf("%s: respuesta no es JSON válido: %v", caso.nombre, err)
	}

	if cuerpo["campo"] != caso.campo {
		t.Errorf("%s: campo = %v, want %q", caso.nombre, cuerpo["campo"], caso.campo)
	}

	if cuerpo["ok"] != false {
		t.Errorf("%s: ok = %v, want false", caso.nombre, cuerpo["ok"])
	}
}

// TestPaquetes cubre la colección de paquetes.
func TestPaquetes(t *testing.T) {
	t.Parallel()

	handler := NuevoSistema().Paquetes
	casos := []casoPeticion{
		{nombre: casoListar, metodo: http.MethodGet, codigo: http.StatusNotImplemented},
		{nombre: casoValido, metodo: http.MethodPost, cuerpo: `{"nombre":"nano"}`, codigo: http.StatusNotImplemented},
		{nombre: "nombre con shell", metodo: http.MethodPost, cuerpo: `{"nombre":"nano; rm -rf /"}`, codigo: http.StatusBadRequest, campo: campoNombre},
		{nombre: casoCuerpoInvalido, metodo: http.MethodPost, cuerpo: `{`, codigo: http.StatusBadRequest},
		{nombre: casoMetodoNoPermitido, metodo: http.MethodPut, codigo: http.StatusMethodNotAllowed},
	}

	for _, caso := range casos {
		rec := ejecutarCaso(t, handler, "/api/packages", caso)
		comprobarCaso(t, rec, caso)
	}
}

// TestPaquete cubre la eliminación de un paquete por su nombre.
func TestPaquete(t *testing.T) {
	t.Parallel()

	handler := NuevoSistema().Paquete
	casos := []struct {
		casoPeticion
		nombre string
	}{
		{casoPeticion{nombre: casoEliminar, metodo: http.MethodDelete, codigo: http.StatusNotImplemented}, nombreNano},
		{casoPeticion{nombre: casoNombreInvalido, metodo: http.MethodDelete, codigo: http.StatusBadRequest, campo: campoName}, "nano; rm"},
		{casoPeticion{nombre: casoMetodoNoPermitido, metodo: http.MethodGet, codigo: http.StatusMethodNotAllowed}, nombreNano},
	}

	for _, caso := range casos {
		rec := ejecutarCasoConPath(t, handler, "/api/packages/"+nombreNano, campoName, caso.nombre, caso.casoPeticion)
		comprobarCaso(t, rec, caso.casoPeticion)
	}
}

// TestLlaveRoot cubre la generación y descarga de la llave de root.
func TestLlaveRoot(t *testing.T) {
	t.Parallel()

	handler := NuevoSistema().LlaveRoot
	casos := []casoPeticion{
		{nombre: "descargar", metodo: http.MethodGet, codigo: http.StatusNotImplemented},
		{nombre: "generar", metodo: http.MethodPost, codigo: http.StatusNotImplemented},
		{nombre: casoMetodoNoPermitido, metodo: http.MethodPut, codigo: http.StatusMethodNotAllowed},
	}

	for _, caso := range casos {
		rec := ejecutarCaso(t, handler, "/api/keys/root", caso)
		comprobarCaso(t, rec, caso)
	}
}

// TestIssue cubre la lectura y el reemplazo de /etc/issue.
func TestIssue(t *testing.T) {
	t.Parallel()

	handler := NuevoSistema().Issue
	casos := []casoPeticion{
		{nombre: casoConsultar, metodo: http.MethodGet, codigo: http.StatusNotImplemented},
		{
			nombre: casoValido,
			metodo: http.MethodPut,
			cuerpo: `{"mensaje":"Bienvenido","nombre_host":true,"fecha":true,"hora":true,"ip":true}`,
			codigo: http.StatusNotImplemented,
		},
		{
			nombre: "mensaje largo",
			metodo: http.MethodPut,
			cuerpo: `{"mensaje":"` + strings.Repeat("a", 600) + `","nombre_host":true,"fecha":true,"hora":true,"ip":true}`,
			codigo: http.StatusBadRequest,
			campo:  campoMensaje,
		},
		{nombre: casoCuerpoInvalido, metodo: http.MethodPut, cuerpo: `{`, codigo: http.StatusBadRequest},
		{nombre: casoMetodoNoPermitido, metodo: http.MethodPost, codigo: http.StatusMethodNotAllowed},
	}

	for _, caso := range casos {
		rec := ejecutarCaso(t, handler, "/api/issue", caso)
		comprobarCaso(t, rec, caso)
	}
}

// TestHostname cubre la lectura y el reemplazo del nombre de host.
func TestHostname(t *testing.T) {
	t.Parallel()

	handler := NuevoSistema().Hostname
	casos := []casoPeticion{
		{nombre: casoConsultar, metodo: http.MethodGet, codigo: http.StatusNotImplemented},
		{nombre: casoValido, metodo: http.MethodPut, cuerpo: `{"nombre":"nodo-web-01","ip":"192.168.1.60"}`, codigo: http.StatusNotImplemented},
		{nombre: "nombre con espacio", metodo: http.MethodPut, cuerpo: `{"nombre":"mi equipo","ip":"192.168.1.60"}`, codigo: http.StatusBadRequest, campo: campoNombre},
		{nombre: "IP inválida", metodo: http.MethodPut, cuerpo: `{"nombre":"nodo-web-01","ip":"999.1.1.1"}`, codigo: http.StatusBadRequest, campo: campoIP},
		{nombre: casoCuerpoInvalido, metodo: http.MethodPut, cuerpo: `{`, codigo: http.StatusBadRequest},
		{nombre: casoMetodoNoPermitido, metodo: http.MethodPost, codigo: http.StatusMethodNotAllowed},
	}

	for _, caso := range casos {
		rec := ejecutarCaso(t, handler, "/api/hostname", caso)
		comprobarCaso(t, rec, caso)
	}
}
