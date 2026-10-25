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

	"consola-vm/internal/core/domain"
	"consola-vm/internal/services"
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
	rutaPaquetes          = "/api/packages"
	rutaPaqueteNano       = rutaPaquetes + "/" + nombreNano
	operacionInstalarNano = "apt-get install -y nano"
)

// stubPaquetes es un servicio de paquetes falso con resultados
// preprogramados.
type stubPaquetes struct {
	lista     []services.Paquete
	resultado services.Resultado
	err       error
	nombre    string
}

// Listar devuelve la lista preprogramada.
func (s *stubPaquetes) Listar(context.Context) ([]services.Paquete, error) {
	return s.lista, s.err
}

// Instalar registra el nombre y devuelve el resultado preprogramado.
func (s *stubPaquetes) Instalar(_ context.Context, nombre string) (services.Resultado, error) {
	s.nombre = nombre

	return s.resultado, s.err
}

// Eliminar registra el nombre y devuelve el resultado preprogramado.
func (s *stubPaquetes) Eliminar(_ context.Context, nombre string) (services.Resultado, error) {
	s.nombre = nombre

	return s.resultado, s.err
}

// nuevoSistemaPrueba arma el handler de sistema con el servicio falso.
func nuevoSistemaPrueba() *Sistema {
	return NuevoSistema(&stubPaquetes{})
}

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

// TestPaquetesListar cubre el listado de paquetes instalados.
func TestPaquetesListar(t *testing.T) {
	t.Parallel()

	servicio := &stubPaquetes{lista: []services.Paquete{{Nombre: nombreNano, Version: "7.2-1"}}}
	rec := ejecutarCaso(t, NuevoSistema(servicio).Paquetes, rutaPaquetes, casoPeticion{
		nombre: casoListar, metodo: http.MethodGet,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("código = %d, want %d (cuerpo %s)", rec.Code, http.StatusOK, rec.Body.String())
	}

	var cuerpo struct {
		Paquetes []paqueteRespuesta `json:"paquetes"`
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &cuerpo); err != nil {
		t.Fatalf(msgJSONInvalido, err)
	}

	if len(cuerpo.Paquetes) != 1 || cuerpo.Paquetes[0].Nombre != nombreNano {
		t.Errorf("paquetes = %+v, want [%s]", cuerpo.Paquetes, nombreNano)
	}
}

// TestPaquetesInstalar cubre la instalación de un paquete.
func TestPaquetesInstalar(t *testing.T) {
	t.Parallel()

	servicio := &stubPaquetes{resultado: services.Resultado{
		Mensaje:   "Paquete «nano» instalado.",
		Operacion: operacionInstalarNano,
	}}

	rec := ejecutarCaso(t, NuevoSistema(servicio).Paquetes, rutaPaquetes, casoPeticion{
		nombre: casoValido, metodo: http.MethodPost, cuerpo: `{"nombre":"nano"}`,
	})

	if rec.Code != http.StatusCreated {
		t.Fatalf("código = %d, want %d (cuerpo %s)", rec.Code, http.StatusCreated, rec.Body.String())
	}

	if got := rec.Header().Get("Location"); got != rutaPaqueteNano {
		t.Errorf("Location = %q, want %q", got, rutaPaqueteNano)
	}

	if servicio.nombre != nombreNano {
		t.Errorf("nombre = %q, want %q", servicio.nombre, nombreNano)
	}

	var cuerpo map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &cuerpo); err != nil {
		t.Fatalf(msgJSONInvalido, err)
	}

	if cuerpo["ok"] != true {
		t.Errorf("ok = %v, want true", cuerpo["ok"])
	}
}

// TestPaquetesValidacion cubre los errores de entrada de la colección.
func TestPaquetesValidacion(t *testing.T) {
	t.Parallel()

	handler := nuevoSistemaPrueba().Paquetes
	casos := []casoPeticion{
		{nombre: "nombre con shell", metodo: http.MethodPost, cuerpo: `{"nombre":"nano; rm -rf /"}`, codigo: http.StatusBadRequest, campo: campoNombre},
		{nombre: casoCuerpoInvalido, metodo: http.MethodPost, cuerpo: `{`, codigo: http.StatusBadRequest},
		{nombre: casoMetodoNoPermitido, metodo: http.MethodPut, codigo: http.StatusMethodNotAllowed},
	}

	for _, caso := range casos {
		rec := ejecutarCaso(t, handler, rutaPaquetes, caso)
		comprobarCaso(t, rec, caso)
	}
}

// TestPaquetesErrorDeOrden cubre el fallo de APT con código distinto de
// cero.
func TestPaquetesErrorDeOrden(t *testing.T) {
	t.Parallel()

	servicio := &stubPaquetes{
		resultado: services.Resultado{
			Operacion: operacionInstalarNano,
			Stderr:    "E: Unable to locate package nano",
			Codigo:    100,
		},
		err: &domain.ExitError{Code: 100, Command: operacionInstalarNano},
	}

	rec := ejecutarCaso(t, NuevoSistema(servicio).Paquetes, rutaPaquetes, casoPeticion{
		nombre: "fallo de APT", metodo: http.MethodPost, cuerpo: `{"nombre":"nano"}`,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCodigo, rec.Code, http.StatusInternalServerError)
	}

	var cuerpo map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &cuerpo); err != nil {
		t.Fatalf(msgJSONInvalido, err)
	}

	if cuerpo["codigo"] != float64(100) {
		t.Errorf("codigo = %v, want 100", cuerpo["codigo"])
	}

	if cuerpo["stderr"] != "E: Unable to locate package nano" {
		t.Errorf("stderr = %v, want el stderr de APT", cuerpo["stderr"])
	}
}

// TestPaquetesSinSesion cubre la ausencia de sesión con la máquina.
func TestPaquetesSinSesion(t *testing.T) {
	t.Parallel()

	servicio := &stubPaquetes{err: services.ErrSesionNoDisponible}

	rec := ejecutarCaso(t, NuevoSistema(servicio).Paquetes, rutaPaquetes, casoPeticion{
		nombre: "sin sesión", metodo: http.MethodGet,
	})

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("código = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

// TestPaquete cubre la eliminación de un paquete por su nombre.
func TestPaquete(t *testing.T) {
	t.Parallel()

	handler := nuevoSistemaPrueba().Paquete
	casos := []struct {
		casoPeticion
		nombre string
	}{
		{casoPeticion{nombre: casoEliminar, metodo: http.MethodDelete, codigo: http.StatusNoContent}, nombreNano},
		{casoPeticion{nombre: casoNombreInvalido, metodo: http.MethodDelete, codigo: http.StatusBadRequest, campo: campoName}, "nano; rm"},
		{casoPeticion{nombre: casoMetodoNoPermitido, metodo: http.MethodGet, codigo: http.StatusMethodNotAllowed}, nombreNano},
	}

	for _, caso := range casos {
		rec := ejecutarCasoConPath(t, handler, "/api/packages/"+nombreNano, campoName, caso.nombre, caso.casoPeticion)
		comprobarCaso(t, rec, caso.casoPeticion)
	}
}

// TestPaqueteEliminar comprueba que el nombre de la ruta llega al
// servicio y que la respuesta no lleva cuerpo.
func TestPaqueteEliminar(t *testing.T) {
	t.Parallel()

	servicio := &stubPaquetes{}
	handler := NuevoSistema(servicio).Paquete

	rec := ejecutarCasoConPath(t, handler, rutaPaqueteNano, campoName, nombreNano, casoPeticion{
		nombre: casoEliminar, metodo: http.MethodDelete,
	})

	if rec.Code != http.StatusNoContent {
		t.Fatalf("código = %d, want %d", rec.Code, http.StatusNoContent)
	}

	if servicio.nombre != nombreNano {
		t.Errorf("nombre = %q, want %q", servicio.nombre, nombreNano)
	}

	if rec.Body.Len() != 0 {
		t.Errorf("cuerpo = %q, want vacío", rec.Body.String())
	}
}

// TestLlaveRoot cubre la generación y descarga de la llave de root.
func TestLlaveRoot(t *testing.T) {
	t.Parallel()

	handler := nuevoSistemaPrueba().LlaveRoot
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

	handler := nuevoSistemaPrueba().Issue
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

	handler := nuevoSistemaPrueba().Hostname
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
